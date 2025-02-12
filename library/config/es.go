package config

// ESConfigEntry ElasticSearch configuration
type ESConfigEntry struct {
	ES struct {
		Address             []string `toml:"address"`                // ElasticSearch address
		Username            string   `toml:"username"`               // ElasticSearch username
		Password            string   `toml:"password"`               // ElasticSearch password
		MaxIdleConns        int      `toml:"max_idle_conns"`         // ElasticSearch max idle connections
		MaxIdleConnsPerHost int      `toml:"max_idle_conns_perhost"` // ElasticSearch max idle connections per host
		IdleConnTimeout     int32    `toml:"idle_conn_timeout"`      // ElasticSearch idle connection timeout
	} `toml:"es"`
}
