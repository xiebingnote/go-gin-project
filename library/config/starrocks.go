package config

// StarRocksConfigEntry
type StarRocksConfigEntry struct {
	StarRocks struct {
		EndPoints   []string `toml:"EndPoints"`   // StarRocks地址
		HttpPort    int      `toml:"HttpPort"`    // StarRocks HTTP端口
		FEPort      int      `toml:"FEPort"`      // StarRocks FE端口
		UserName    string   `toml:"UserName"`    // StarRocks 用户名
		PassWord    string   `toml:"PassWord"`    // StarRocks 密码
		Parallelism int      `toml:"Parallelism"` // StarRocks 并发数
		FileDir     string   `toml:"FileDir"`     // StarRocks 文件目录
	} `toml:"StarRocks"`

	FlinkCDC struct {
		Path string `toml:"Path"` // FlinkCdc 路径
		Cmd  string `toml:"Cmd"`  // FlinkCdc 命令
	} `toml:"FlinkCdc"`
}
