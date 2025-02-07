package config

type LogConfigEntry struct {
	Log struct {
		DefaultLevel string `toml:"DefaultLevel"`
		LogDir       string `toml:"LogDir"`
		LogFileDebug string `toml:"LogFileDebug"`
		LogFileInfo  string `toml:"LogFileInfo"`
		LogFileWarn  string `toml:"LogFileWarn"`
		LogFileError string `toml:"LogFileError"`
		MaxSize      int    `toml:"MaxSize"`
		MaxAge       int    `toml:"MaxAge"`
		MaxBackups   int    `toml:"MaxBackups"`
		LocalTime    bool   `toml:"LocalTime"`
		Compress     bool   `toml:"Compress"`
	} `toml:"Log"`
}
