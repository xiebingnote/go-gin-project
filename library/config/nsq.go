package config

import "time"

type NsqConfigEntry struct {
	NSQ struct {
		LookupdAddress []string       `toml:"LookupdAddress"`
		Producer       ProducerConfig `toml:"Producer"`
		Consumer       ConsumerConfig `toml:"Consumer"`
	} `toml:"NSQ"`
}

type ProducerConfig struct {
	DialTimeout time.Duration `toml:"DialTimeout"`
	MaxAttempts int           `toml:"MaxAttempts"`
	Concurrency int           `toml:"Concurrency"`
}

type ConsumerConfig struct {
	Topic             string        `toml:"Topic"`
	Channel           string        `toml:"Channel"`
	MaxInFlight       int           `toml:"MaxInFlight"`
	MaxAttempts       int           `toml:"MaxAttempts"`
	RequeueDelay      time.Duration `toml:"RequeueDelay"`
	HeartbeatInterval time.Duration `toml:"HeartbeatInterval"`
}
