package service

import (
	"context"
	"sync"
)

// waitForClose bounds a driver operation that does not accept a context. The
// callback must capture its resources and must not read or mutate global state:
// a driver may finish after the caller's deadline.
func waitForClose(ctx context.Context, closeFn func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- closeFn() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// healthWorker is started and stopped by the application's serial lifecycle.
// Joining it before closing resources prevents health checks racing with close.
type healthWorker struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func (w *healthWorker) start(ctx context.Context, run func(context.Context)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.done != nil {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	w.cancel, w.done = cancel, done
	go func() {
		defer close(done)
		run(workerCtx)
	}()
}

func (w *healthWorker) stop(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.done == nil {
		return nil
	}
	w.cancel()
	select {
	case <-w.done:
		w.cancel, w.done = nil, nil
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

var cronHealthWorker, redisHealthWorker, kafkaHealthWorker healthWorker

// StopHealthChecks must finish before the scheduler, clients or logger close.
func StopHealthChecks(ctx context.Context) error {
	for _, worker := range []*healthWorker{&cronHealthWorker, &redisHealthWorker, &kafkaHealthWorker} {
		if err := worker.stop(ctx); err != nil {
			return err
		}
	}
	return nil
}
