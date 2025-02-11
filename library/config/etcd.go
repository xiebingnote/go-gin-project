package config

import "time"

type EtcdConfigEntry struct {
	Etcd struct {
		Endpoints   []string      `toml:"Endpoints"`
		DialTimeout time.Duration `toml:"DialTimeout"`
		Username    string        `toml:"UserName"`
		Password    string        `toml:"PassWord"`
	} `toml:"ETCD"`
}
