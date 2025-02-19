package config

type MongoDBConfigEntry struct {
	Mongo struct {
		Host     string `toml:"Host"`     // host
		Port     int64  `toml:"Port"`     // port
		Username string `toml:"Username"` // username
		Password string `toml:"Password"` // password
		DBName   string `toml:"DBName"`   //
	} `toml:"MongoDB""`
}
