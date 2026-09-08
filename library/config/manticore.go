package config

// ManticoreConfigEntry Manticore config entry
type ManticoreConfigEntry struct {
	Manticore struct {
		Endpoints []string `toml:"endpoints"` // host or full HTTP(S) URL; credentials require HTTPS
		Port      int      `toml:"Port"`      // default port for bare hosts
		UserName  string   `toml:"UserName"`  // username
		PassWord  string   `toml:"PassWord"`  // password
	} `toml:"Manticore"`
}
