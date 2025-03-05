package config

// ManticoreConfigEntry Manticore config entry
type ManticoreConfigEntry struct {
	Manticore struct {
		Endpoints []string `toml:"endpoints"` // host
		Port      int      `toml:"Port"`      // port
		UserName  string   `toml:"UserName"`  // username
		PassWord  string   `toml:"PassWord"`  // password
	} `toml:"Manticore"`
}
