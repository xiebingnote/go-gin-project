package config

// ElasticSearchConfigEntry ElasticSearch configuration
type ElasticSearchConfigEntry struct {
	ElasticSearch struct {
		Address             []string `toml:"Address"`             // ElasticSearch address
		Username            string   `toml:"UserName"`            // ElasticSearch username
		Password            string   `toml:"PassWord"`            // ElasticSearch password
		MaxIdleConns        int      `toml:"MaxIdleConns"`        // ElasticSearch max idle connections
		MaxIdleConnsPerHost int      `toml:"MaxIdleConnsPerhost"` // ElasticSearch max idle connections per host
		IdleConnTimeout     int32    `toml:"IdleConnTimeout"`     // ElasticSearch idle connection timeout
	} `toml:"ElasticSearch"`
}
