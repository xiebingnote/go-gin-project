package config

// LogConfigEntry log config entry
type LogConfigEntry struct {
	Log struct {
		DefaultLevel string `toml:"DefaultLevel"` // 默认日志级别
		LogDir       string `toml:"LogDir"`       // 日志目录
		LogFileDebug string `toml:"LogFileDebug"` // 日志文件
		LogFileInfo  string `toml:"LogFileInfo"`  // 日志文件
		LogFileWarn  string `toml:"LogFileWarn"`  // 日志文件
		LogFileError string `toml:"LogFileError"` // 日志文件
		MaxSize      int    `toml:"MaxSize"`      // 日志文件最大大小
		MaxAge       int    `toml:"MaxAge"`       // 日志文件最大保存天数
		MaxBackups   int    `toml:"MaxBackups"`   // 日志文件最大备份数量
		LocalTime    bool   `toml:"LocalTime"`    // 是否使用本地时间
		Compress     bool   `toml:"Compress"`     // 是否压缩
	} `toml:"Log"`
}
