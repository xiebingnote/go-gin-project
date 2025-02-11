package config

var (
	// ESConfig ElasticSearch config entry
	ESConfig *ESConfigEntry

	// EtcdConfig etcd config entry
	EtcdConfig *EtcdConfigEntry

	// KafkaConfig kafka config entry
	KafkaConfig *KafkaConfigEntry

	// LogConfig log config entry
	LogConfig *LogConfigEntry

	// MongoConfig mongodb config entry
	MongoConfig *MongoDBConfigEntry

	// MySQLConfig mysql config entry
	MySQLConfig *MySQLConfigEntry

	// NsqConfig nsq config entry
	NsqConfig *NsqConfigEntry

	// PostgresqlConfig postgresql config entry
	PostgresqlConfig *PostgresqlConfigEntry

	// RedisConfig redis config entry
	RedisConfig *RedisConfigEntry

	// ServerConfig server config entry
	ServerConfig *ServerConfigEntry
)
