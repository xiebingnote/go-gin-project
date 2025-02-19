package config

// ClickHouseConfigEntry ClickHouse config entry
type ClickHouseConfigEntry struct {
	ClickHouse struct {
		Host     string `toml:"Host"`     // host
		Port     int64  `toml:"Port"`     // port
		Database string `toml:"Database"` // dbname
		Username string `toml:"UserName"` // username
		Password string `toml:"PassWord"` // password
	} `toml:"ClickHouse"`
}
