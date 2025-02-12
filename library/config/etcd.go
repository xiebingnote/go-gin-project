package config

import "time"

// EtcdConfigEntry etcd配置
type EtcdConfigEntry struct {
	Etcd struct {
		Endpoints   []string      `toml:"Endpoints"`   // ETCD地址
		DialTimeout time.Duration `toml:"DialTimeout"` // ETCD连接超时时间
		Username    string        `toml:"UserName"`    // ETCD用户名
		Password    string        `toml:"PassWord"`    // ETCD密码
	} `toml:"ETCD"`
}
