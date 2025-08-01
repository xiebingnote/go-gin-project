package request

// CreateFlinkCdcToMySQLYamlReq 创建 FlinkCdc 配置文件
type CreateFlinkCdcToMySQLYamlReq struct {
	JobName  string `json:"job_name"`  // job_name
	Tables   string `json:"tables"`    // table
	SinkName string `json:"sink_name"` // sink_name
}

// AddFlinkCdcJobReq 添加 FlinkCdc 任务
type AddFlinkCdcJobReq struct {
	FileName string `json:"file_name"` // FlinkCdc 配置文件名
}
