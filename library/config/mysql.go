package config

import "time"

type MySQLConfigEntry struct {
	Name         string        `toml:"Name"`
	ConnTimeOut  time.Duration `toml:"ConnTimeOut"`  // 单位：ms
	WriteTimeOut time.Duration `toml:"WriteTimeOut"` // 单位：ms
	ReadTimeOut  time.Duration `toml:"ReadTimeOut"`  // 单位：ms
	Retry        int           `toml:"Retry"`

	Strategy struct {
		Name string `toml:"Name"`
	} `toml:"Strategy"`

	Resource struct {
		Manual struct {
			Default []struct {
				Host string `toml:"Host"`
				Port int    `toml:"Port"`
			} `toml:"default"`
		} `toml:"Manual"`
	} `toml:"Resource"`

	MySQL struct {
		Username        string `toml:"Username"`
		Password        string `toml:"Password"`
		DBName          string `toml:"DBName"`
		DBDriver        string `toml:"DBDriver"`
		MaxOpenPerIP    int    `toml:"MaxOpenPerIP"`
		MaxIdlePerIP    int    `toml:"MaxIdlePerIP"`
		ConnMaxLifeTime int    `toml:"ConnMaxLifeTime"` // 单位：ms
		SQLLogLen       int    `toml:"SQLLogLen"`
		SQLArgsLogLen   int    `toml:"SQLArgsLogLen"`
		LogIDTransport  bool   `toml:"LogIDTransport"`
		DSNParams       string `toml:"DSNParams"`
	} `toml:"MySQL"`
}
