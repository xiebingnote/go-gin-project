package config

// KafkaConfigEntry kafka配置
type KafkaConfigEntry struct {
	Kafka struct {
		Brokers       []string `toml:"Brokers"`       // Kafka broker address
		ProducerTopic string   `toml:"ProducerTopic"` // 生产者主题
		ConsumerTopic string   `toml:"ConsumerTopic"` // 消费者主题
		GroupID       string   `toml:"GroupID"`       // 消费者组ID
		Version       string   `toml:"Version"`       // Kafka version，eg: "2.8.0"
	} `toml:"Kafka"`

	Advanced struct {
		ProducerMaxRetry       int `toml:"ProducerMaxRetry"`       // 生产者最大重试次数，整数
		ConsumerSessionTimeout int `toml:"ConsumerSessionTimeout"` // 消费者会话超时（ms），整数
		HeartbeatInterval      int `toml:"HeartbeatInterval"`      // 心跳间隔（ms），整数
		MaxProcessingTime      int `toml:"MaxProcessingTime"`      // 最大处理时间（ms），整数
	} `toml:"Advanced"`
}
