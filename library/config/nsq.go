package config

import "time"

// NsqConfigEntry NSQ config entry
type NsqConfigEntry struct {
	NSQ struct {
		LookupdAddress []string       `toml:"NsqLookupdAddress"` // NSQ lookupd address
		Producer       ProducerConfig `toml:"Producer"`          // NSQ producer
		Consumer       ConsumerConfig `toml:"Consumer"`          // NSQ consumer
	} `toml:"NSQ"`
}

// ProducerConfig NSQ producer config
type ProducerConfig struct {
	DialTimeout time.Duration `toml:"DialTimeout"` // NSQ producer dial timeout
	MaxAttempts int           `toml:"MaxAttempts"` // NSQ producer max attempts
	Concurrency int           `toml:"Concurrency"` // NSQ producer concurrency
}

// ConsumerConfig NSQ consumer config
type ConsumerConfig struct {
	Topic             string        `toml:"Topic"`             // NSQ consumer topic
	Channel           string        `toml:"Channel"`           // NSQ consumer channel
	MaxInFlight       int           `toml:"MaxInFlight"`       // NSQ consumer max in flight
	MaxAttempts       int           `toml:"MaxAttempts"`       // NSQ consumer max attempts
	RequeueDelay      time.Duration `toml:"RequeueDelay"`      // NSQ consumer requeue delay
	HeartbeatInterval time.Duration `toml:"HeartbeatInterval"` // NSQ consumer heartbeat interval
}
