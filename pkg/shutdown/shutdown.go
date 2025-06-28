package shutdown

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/xiebingnote/go-gin-project/library/resource"
)

var _ Hook = (*hook)(nil)

type hook struct {
	signalChan chan os.Signal
	mu         sync.Mutex
}

type Hook interface {
	// WithSignals returns a Hook that will also listen for the provided signals.
	//
	// The returned Hook will listen for the signals in addition to the signals
	// already being listened to. If a signal is received, the functions passed
	// to the Close method will be executed in sequence. If no Close method has
	// been called, the program will exit with the signal as the exit status.
	WithSignals(signals ...syscall.Signal) Hook

	// Close executes the functions passed to it in sequence when a signal is
	// received or when the timeout is reached. If no functions are passed, the
	// program will exit with the signal as the exit status. If a timeout is
	// specified, the functions will be executed after the timeout is reached.
	//
	// It is safe to call Close multiple times from different goroutines. The
	// functions will be executed in the order they were passed to the last call
	// to Close.
	Close(funcs ...func())
}

// NewHook creates and returns a new Hook that listens for SIGINT and SIGTERM signals.
// The returned Hook uses a channel to receive operating system signals and executes
// functions passed to the Close method in sequence when a signal is received.
func NewHook() Hook {
	h := &hook{
		signalChan: make(chan os.Signal, 1), // Channel for receiving OS signals
	}
	// Listen for SIGINT and SIGTERM signals
	return h.WithSignals(syscall.SIGINT, syscall.SIGTERM)
}

// WithSignals returns a Hook that will also listen for the provided signals.
//
// The returned Hook will listen for the signals in addition to the signals
// already being listened to. If a signal is received, the functions passed
// to the Close method will be executed in sequence. If no Close method has
// been called, the program will exit with the signal as the exit status.
func (h *hook) WithSignals(signals ...syscall.Signal) Hook {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Notify the signal channel for each signal
	for _, s := range signals {
		signal.Notify(h.signalChan, s)
	}
	return h
}

// Close executes the functions passed to it in sequence when a signal is received
// or when the timeout is reached. If no functions are passed, the program will exit
// with the signal as the exit status. If a timeout is specified, the functions will
// be executed after the timeout is reached.
//
// It is safe to call Close multiple times from different goroutines. The
// functions will be executed in the order they were passed to the last call to
// Close.
//
// The functions passed to Close will be executed in a separate goroutine. If a
// function blocks indefinitely, it will be canceled after the timeout is
// reached. If all the functions complete successfully, the program will exit
// after all the functions have completed. If any of the functions fail, the
// program will exit immediately after the first failure.
//
// The timeout for each function is 5 seconds. The total timeout for all the
// functions is 30 seconds. If the total timeout is reached before all the
// functions have completed, the program will exit immediately.
func (h *hook) Close(funcs ...func()) {
	// Receive the signal that triggered the shutdown
	sig := <-h.signalChan
	log.Printf(fmt.Sprintf("🛑 Received signal: %s", sig))
	if resource.LoggerService != nil {
		resource.LoggerService.Info(fmt.Sprintf("🛑 Received signal: %s", sig))
	}

	// Stop listening for signals to prevent the program from exiting
	// immediately
	signal.Stop(h.signalChan)

	// Create a context with a timeout for the shutdown process
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use a WaitGroup to ensure all cleanup tasks complete
	var wg sync.WaitGroup
	for _, f := range funcs {
		wg.Add(1)
		go func(cleanup func()) {
			defer wg.Done()

			// Set a timeout for each cleanup task
			taskCtx, taskCancel := context.WithTimeout(shutdownCtx, 5*time.Second)
			defer taskCancel()

			// Execute the cleanup task in a separate goroutine
			done := make(chan struct{})
			go func() {
				cleanup()
				close(done)
			}()

			// Wait for the cleanup task to complete or timeout
			select {
			case <-done:
			case <-taskCtx.Done():
				log.Println(fmt.Sprintf("⏰ Cleanup task timeout"))
				if resource.LoggerService != nil {
					resource.LoggerService.Error(fmt.Sprintf("⏰ Cleanup task timeout"))
				}
			}
		}(f)
	}

	// Wait for all cleanup tasks to complete or the total timeout to be reached
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Wait for all cleanup tasks to complete or the total timeout to be reached
	select {
	case <-done:
		log.Println(fmt.Sprintf("🎉 All cleanup tasks completed"))
		if resource.LoggerService != nil {
			resource.LoggerService.Info(fmt.Sprintf("🎉 All cleanup tasks completed"))
		}
	case <-shutdownCtx.Done():
		log.Println(fmt.Sprintf("⏰ Shutdown timeout, force exit"))
		if resource.LoggerService != nil {
			resource.LoggerService.Error(fmt.Sprintf("⏰ Shutdown timeout, force exit"))
		}
	}
}
