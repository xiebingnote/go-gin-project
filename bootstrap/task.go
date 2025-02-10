package bootstrap

import (
	"context"
	"log"
	"sync"
)

// TaskStart initializes and starts a series of tasks.
//
// This function logs the initiation of task processing and then calls the Task function
// to execute any one-off task.
// The context parameter is used to control the lifecycle
// of the tasks, allowing them to be canceled or timed out if needed.
func TaskStart(ctx context.Context) {
	log.Println("TaskStart.")

	Task(ctx)
}

// Task runs once-off tasks.
//
// The function is a noop right now, but can be used to run one-off task in the
// future.
func Task(ctx context.Context) {

	//cron: = timer.NewSimpleCron(time.Minute * 1)
	//cron.AddJob(func() {
	//
	//})

	one := sync.Once{}
	one.Do(func() {

	})
}
