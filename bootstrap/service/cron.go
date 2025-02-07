package service

import (
	"context"
	"log"
	"time"

	"project/library/resource"

	"github.com/go-co-op/gocron/v2"
)

// InitCron initializes the Corn field of the resource package with a new scheduler.
// The scheduler is configured to use the local time zone.
// If the scheduler creation fails, the function logs the error and exits.
// If the time zone loading fails, the function logs the error and exits.
// After successful creation, the scheduler is started.
func InitCron(_ context.Context) {
	// Attempt to load the local time zone.
	jst, err := time.LoadLocation(time.Local.String())
	if err != nil {
		// Log and exit if loading the time zone fails.
		log.Fatalf("time.LoadLocation(%s) error(%v)", time.Local.String(), err)
		return
	}

	// Create a new scheduler with the loaded time zone.
	resource.Corn, err = gocron.NewScheduler(gocron.WithLocation(jst))
	if err != nil {
		// Log and exit if scheduler creation fails.
		log.Fatalf("gocron.NewScheduler() error(%v)", err)
		return
	}

	// Start the scheduler.
	resource.Corn.Start()
}
