package config

import "time"

// CronConfigEntry is the configuration entry for cron scheduler.
type CronConfigEntry struct {
	Cron struct {
		TimeZone                string        `toml:"TimeZone"`                // Time zone for the scheduler
		AutoStart               bool          `toml:"AutoStart"`               // Enable scheduler auto-start
		MaxConcurrentJobs       int           `toml:"MaxConcurrentJobs"`       // Maximum number of concurrent jobs
		JobTimeout              time.Duration `toml:"JobTimeout"`              // Job execution timeout in seconds
		EnableRecovery          bool          `toml:"EnableRecovery"`          // Enable job recovery on startup
		HealthCheckInterval     time.Duration `toml:"HealthCheckInterval"`     // Health check interval in seconds
		EnableDetailedLogging   bool          `toml:"EnableDetailedLogging"`   // Enable detailed logging
		JobHistoryRetention     time.Duration `toml:"JobHistoryRetention"`     // Job history retention in hours
	} `toml:"Cron"`
}
