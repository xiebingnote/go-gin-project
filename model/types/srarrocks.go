package types

const (
	TablesExclude   = `information_schema\.*`
	ServerTimeZone  = "UTC"
	ScanStartupMode = "initial"
	SourcdType      = "mysql"
	SinkType        = "starrocks"
)

type StarRocksConfig struct {
	Source struct {
		Type            string `yaml:"type"`
		Hostname        string `yaml:"hostname"`
		Port            int    `yaml:"port"`
		Username        string `yaml:"username"`
		Password        string `yaml:"password"`
		Tables          string `yaml:"tables"`
		TablesExclude   string `yaml:"tables.exclude"`
		ServerID        int    `yaml:"server-id"`
		ServerTimeZone  string `yaml:"server-time-zone"`
		ScanStartupMode string `yaml:"scan.startup.mode"`
	} `yaml:"source"`

	Sink struct {
		Type     string `yaml:"type"`
		Name     string `yaml:"name"`
		JdbcURL  string `yaml:"jdbc-url"`
		LoadURL  string `yaml:"load-url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"sink"`

	Pipeline struct {
		Name        string `yaml:"name"`
		Parallelism int    `yaml:"parallelism"`
	} `yaml:"pipeline"`
}
