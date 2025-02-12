package config

import "time"

// MySQLConfigEntry MySQL config entry
type MySQLConfigEntry struct {
	Name         string        `toml:"Name"`
	ConnTimeOut  time.Duration `toml:"ConnTimeOut"`  // 单位：ms
	WriteTimeOut time.Duration `toml:"WriteTimeOut"` // 单位：ms
	ReadTimeOut  time.Duration `toml:"ReadTimeOut"`  // 单位：ms
	Retry        int           `toml:"Retry"`

	Strategy struct {
		Name string `toml:"Name"` // strategy name
	} `toml:"Strategy"`

	Resource struct {
		Manual struct {
			Default []struct {
				Host string `toml:"Host"` // host
				Port int    `toml:"Port"` // port
			} `toml:"default"`
		} `toml:"Manual"`
	} `toml:"Resource"`

	MySQL struct {
		Username        string `toml:"Username"`        // username
		Password        string `toml:"Password"`        // password
		DBName          string `toml:"DBName"`          // dbname
		DBDriver        string `toml:"DBDriver"`        // dbdriver
		MaxOpenPerIP    int    `toml:"MaxOpenPerIP"`    // max open per ip
		MaxIdlePerIP    int    `toml:"MaxIdlePerIP"`    // max idle per ip
		ConnMaxLifeTime int    `toml:"ConnMaxLifeTime"` // 单位：ms
		SQLLogLen       int    `toml:"SQLLogLen"`       // sql log len
		SQLArgsLogLen   int    `toml:"SQLArgsLogLen"`   // sql args log len
		LogIDTransport  bool   `toml:"LogIDTransport"`  // log id transport
		DSNParams       string `toml:"DSNParams"`       // dsn params
	} `toml:"MySQL"`
}
