package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xiebingnote/go-gin-project/servers"
)

func TestInitializationPanicCleansUp(t *testing.T) {
	cleaned := false
	app := application{
		initialize: func(context.Context) { panic("initialization failed") },
		start: func(context.Context) (*servers.Pair, error) {
			t.Fatal("servers must not start after initialization failed")
			return nil, nil
		},
		cleanup: func(ctx context.Context) error {
			cleaned = true
			if _, ok := ctx.Deadline(); !ok {
				t.Error("cleanup has no deadline")
			}
			return nil
		},
	}
	err := app.run(context.Background(), defaultTimeouts)
	if err == nil || !strings.Contains(err.Error(), "initialization failed") || !cleaned {
		t.Fatalf("run error = %v, cleaned = %v", err, cleaned)
	}
}

func TestStartupFailureCleansUp(t *testing.T) {
	want := errors.New("port already in use")
	cleaned := false
	app := application{
		initialize: func(context.Context) {},
		start: func(ctx context.Context) (*servers.Pair, error) {
			if _, ok := ctx.Deadline(); !ok {
				t.Error("startup has no deadline")
			}
			return nil, want
		},
		cleanup: func(context.Context) error { cleaned = true; return nil },
	}
	if err := app.run(context.Background(), defaultTimeouts); !errors.Is(err, want) || !cleaned {
		t.Fatalf("run error = %v, cleaned = %v", err, cleaned)
	}
}

func TestRuntimeErrorAfterStartupTriggersCleanup(t *testing.T) {
	want := errors.New("listener failed after startup")
	errorsCh := make(chan error, 1)
	started := make(chan struct{})
	cleaned := false
	app := application{
		initialize: func(context.Context) {},
		start: func(context.Context) (*servers.Pair, error) {
			close(started)
			return &servers.Pair{Main: &http.Server{}, Admin: &http.Server{}, Errors: errorsCh}, nil
		},
		cleanup: func(context.Context) error { cleaned = true; return nil },
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- app.run(ctx, defaultTimeouts) }()
	<-started
	// Deliver the failure after the old 100 ms startup-only check.
	time.Sleep(150 * time.Millisecond)
	errorsCh <- want
	close(errorsCh)
	if err := <-result; !errors.Is(err, want) || !cleaned {
		t.Fatalf("run error = %v, cleaned = %v", err, cleaned)
	}
}

func TestGracefulShutdownWaitsBeyondFiveSecondsBeforeCleanup(t *testing.T) {
	requestStarted := make(chan struct{}, 2)
	release := make(chan struct{})
	var completed atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestStarted <- struct{}{}
		<-release
		completed.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	mainServer := httptest.NewServer(handler)
	adminServer := httptest.NewServer(handler)
	defer mainServer.Close()
	defer adminServer.Close()
	// Release handlers before closing test servers, including on test failure.
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()

	started := make(chan struct{})
	cleaned := false
	app := application{
		initialize: func(context.Context) {},
		start: func(context.Context) (*servers.Pair, error) {
			close(started)
			return &servers.Pair{Main: mainServer.Config, Admin: adminServer.Config}, nil
		},
		cleanup: func(ctx context.Context) error {
			cleaned = true
			if completed.Load() != 2 {
				t.Errorf("cleanup ran with only %d requests complete", completed.Load())
			}
			if ctx.Err() != nil {
				t.Errorf("cleanup inherited canceled application context: %v", ctx.Err())
			}
			return nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- app.run(ctx, defaultTimeouts) }()
	<-started
	responses := make(chan error, 2)
	for _, srv := range []*httptest.Server{mainServer, adminServer} {
		go func() {
			response, err := srv.Client().Get(srv.URL)
			if err == nil {
				_, err = io.Copy(io.Discard, response.Body)
				response.Body.Close()
			}
			responses <- err
		}()
	}
	<-requestStarted
	<-requestStarted
	cancel()
	select {
	case err := <-result:
		t.Fatalf("shutdown returned before requests completed: %v", err)
	case <-time.After(5100 * time.Millisecond):
	}
	close(release)
	if err := <-result; err != nil || !cleaned {
		t.Fatalf("run error = %v, cleaned = %v", err, cleaned)
	}
	for i := 0; i < 2; i++ {
		if err := <-responses; err != nil {
			t.Errorf("in-flight request was interrupted: %v", err)
		}
	}
}

func TestShutdownTimeoutKeepsResourcesUsedByHandler(t *testing.T) {
	requestStarted := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-release // Deliberately ignore cancellation to model a stuck handler.
	}))
	defer srv.Close()
	defer close(release)
	cleaned := false
	started := make(chan struct{})
	app := application{
		initialize: func(context.Context) {},
		start: func(context.Context) (*servers.Pair, error) {
			close(started)
			return &servers.Pair{Main: srv.Config, Admin: &http.Server{}}, nil
		},
		cleanup: func(context.Context) error { cleaned = true; return nil },
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	timeouts := defaultTimeouts
	timeouts.ServerShutdown = 30 * time.Millisecond
	go func() { result <- app.run(ctx, timeouts) }()
	<-started
	requestDone := make(chan struct{})
	go func() {
		defer close(requestDone)
		response, err := srv.Client().Get(srv.URL)
		if err == nil {
			response.Body.Close()
		}
	}()
	<-requestStarted
	cancel()
	if err := <-result; !errors.Is(err, context.DeadlineExceeded) || cleaned {
		t.Fatalf("run error = %v, unsafe cleanup = %v", err, cleaned)
	}
	select {
	case <-requestDone:
	case <-time.After(time.Second):
		t.Fatal("timed-out connection was not closed")
	}
}

func TestCleanupFailureIsReturned(t *testing.T) {
	want := errors.New("cleanup failed")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app := application{
		initialize: func(context.Context) { cancel() },
		cleanup:    func(context.Context) error { return want },
	}
	if err := app.run(ctx, defaultTimeouts); !errors.Is(err, want) {
		t.Fatalf("run error = %v, want %v", err, want)
	}
}

func TestMainMissingConfigExitsNonzero(t *testing.T) {
	if os.Getenv("GO_GIN_TEST_MISSING_CONFIG") == "1" {
		main()
		os.Exit(0)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMainMissingConfigExitsNonzero$")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), "GO_GIN_TEST_MISSING_CONFIG=1")
	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("exit error = %v, output = %s", err, output)
	}
	if !strings.Contains(string(output), "Failed to load log configuration file") {
		t.Fatalf("unexpected failure: %s", output)
	}
}
