package config

type TDengineConfigEntry struct {
	TDengine struct {
		Host     string `toml:"Host"`
		Port     string `toml:"Port"`
		UserName string `toml:"UserName"`
		PassWord string `toml:"PassWord"`
		Database string `toml:"Database"`
	} `toml:"TDengine"`
}
