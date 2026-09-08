package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/go-co-op/gocron/v2"
	"github.com/xiebingnote/go-gin-project/library/resource"
	"go.uber.org/zap"
)

type shutdownProducer struct {
	sarama.SyncProducer
	closeFn func() error
}

func (p *shutdownProducer) Close() error { return p.closeFn() }

type shutdownScheduler struct {
	gocron.Scheduler
	closeFn func() error
}

func (s *shutdownScheduler) Shutdown() error { return s.closeFn() }

func TestCloseDrainsCronBeforeClosingDependencies(t *testing.T) {
	oldP, oldS, oldL := resource.KafkaProducer, resource.Corn, resource.LoggerService
	t.Cleanup(func() { resource.KafkaProducer, resource.Corn, resource.LoggerService = oldP, oldS, oldL })
	drained := make(chan struct{})
	resource.Corn = &shutdownScheduler{closeFn: func() error {
		if resource.KafkaProducer == nil || resource.LoggerService == nil {
			t.Error("closed dependencies while cron was active")
		}
		close(drained)
		return nil
	}}
	closed := false
	resource.KafkaProducer = &shutdownProducer{closeFn: func() error {
		select {
		case <-drained:
		default:
			t.Error("producer closed before cron drained")
		}
		closed = true
		return nil
	}}
	resource.LoggerService = zap.NewNop()
	if err := Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !closed {
		t.Fatal("producer was not closed")
	}
	if resource.LoggerService != nil {
		t.Fatal("logger was not closed last")
	}
}

func TestCloseDeadlineBoundsBlockedKafkaAndRetainsResources(t *testing.T) {
	oldP, oldL := resource.KafkaProducer, resource.LoggerService
	t.Cleanup(func() { resource.KafkaProducer, resource.LoggerService = oldP, oldL })
	entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	producer := &shutdownProducer{closeFn: func() error { close(entered); <-release; close(finished); return nil }}
	resource.KafkaProducer = producer
	logger := zap.NewNop()
	resource.LoggerService = logger
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- Close(ctx) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("close did not start")
	}
	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("lost deadline error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cleanup ignored deadline")
	}
	if resource.KafkaProducer != producer || resource.LoggerService != logger {
		t.Fatal("cleared resources before drain completed")
	}
	close(release)
	<-finished
	// The detached driver cleanup must not mutate resource pointers after return.
	if resource.KafkaProducer != producer {
		t.Fatal("late cleanup mutated globals")
	}
}

func TestCloseCronFailureRetainsDependencies(t *testing.T) {
	oldP, oldS, oldL := resource.KafkaProducer, resource.Corn, resource.LoggerService
	t.Cleanup(func() { resource.KafkaProducer, resource.Corn, resource.LoggerService = oldP, oldS, oldL })
	failure := errors.New("jobs still running")
	resource.Corn = &shutdownScheduler{closeFn: func() error { return failure }}
	producer := &shutdownProducer{closeFn: func() error { t.Error("closed dependency while jobs were running"); return nil }}
	resource.KafkaProducer = producer
	logger := zap.NewNop()
	resource.LoggerService = logger
	if err := Close(context.Background()); !errors.Is(err, failure) {
		t.Fatalf("lost drain error: %v", err)
	}
	if resource.KafkaProducer != producer || resource.LoggerService != logger {
		t.Fatal("cleared active dependencies")
	}
}
