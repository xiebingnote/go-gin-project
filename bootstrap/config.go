package bootstrap

import (
	"context"

	"github.com/xiebingnote/go-gin-project/library/config"

	"github.com/BurntSushi/toml"
)

// InitConfig initializes the application configuration by loading and
// decoding various service configuration files in TOML format.
//
// This function loads configuration files for different services including
// logging, server, Elasticsearch, Etcd, Kafka, MongoDB, MySQL, NSQ, Redis,
// ClickHouse and PostgresSQL.
//
// It decodes the configurations and assigns them to their
// respective global configuration variables in the config package.
//
// The function expects configuration files to be located in the "./conf"
// directory with specific subdirectories for each service.
//
// If any error occurs during the decoding of the configuration files,
// the function will panic, providing the error message.
func InitConfig(_ context.Context) {
	// Load Log configuration
	if _, err := toml.DecodeFile("./conf/log/log.toml", &config.LogConfig); err != nil {
		// The log configuration file could not be decoded. Panic with the error message.
		panic("Failed to load log configuration file: " + err.Error())
	}

	// Load Server configuration
	if _, err := toml.DecodeFile("./conf/server.toml", &config.ServerConfig); err != nil {
		// The server configuration file could not be decoded. Panic with the error message.
		panic("Failed to load server configuration file: " + err.Error())
	}

	// Load ClickHouse configuration
	if _, err := toml.DecodeFile("./conf/service/clickhouse.toml", &config.ClickHouseConfig); err != nil {
		// The ClickHouse configuration file could not be decoded. Panic with the error message.
		panic("Failed to load ClickHouse configuration file: " + err.Error())
	}

	// Load Elasticsearch configuration
	if _, err := toml.DecodeFile("./conf/service/elasticsearch.toml", &config.ElasticSearchConfig); err != nil {
		// The Elasticsearch configuration file could not be decoded. Panic with the error message.
		panic("Failed to load Elasticsearch configuration file: " + err.Error())
	}

	// Load etcd configuration
	if _, err := toml.DecodeFile("./conf/service/etcd.toml", &config.EtcdConfig); err != nil {
		// The etcd configuration file could not be decoded. Panic with the error message.
		panic("Failed to load etcd configuration file: " + err.Error())
	}

	// Load Kafka configuration
	if _, err := toml.DecodeFile("./conf/service/kafka.toml", &config.KafkaConfig); err != nil {
		// The Kafka configuration file could not be decoded. Panic with the error message.
		panic("Failed to load Kafka configuration file: " + err.Error())
	}

	// Load Manticore Search configuration
	if _, err := toml.DecodeFile("./conf/service/manticore.toml", &config.ManticoreConfig); err != nil {
		// The Manticore configuration file could not be decoded. Panic with the error message.
		panic("Failed to load Manticore configuration file: " + err.Error())
	}

	// Load MongoDB configuration
	if _, err := toml.DecodeFile("./conf/service/mongodb.toml", &config.MongoConfig); err != nil {
		// The MongoDB configuration file could not be decoded. Panic with the error message.
		panic("Failed to load MongoDB configuration file: " + err.Error())
	}

	// Load MySQL configuration
	if _, err := toml.DecodeFile("./conf/service/mysql.toml", &config.MySQLConfig); err != nil {
		// The MySQL configuration file could not be decoded. Panic with the error message.
		panic("Failed to load MySQL configuration file: " + err.Error())
	}

	// Load NSQ configuration
	if _, err := toml.DecodeFile("./conf/service/nsq.toml", &config.NsqConfig); err != nil {
		// The NSQ configuration file could not be decoded. Panic with the error message.
		panic("Failed to load NSQ configuration file: " + err.Error())
	}

	// Load PostgresSQL configuration
	if _, err := toml.DecodeFile("./conf/service/postgresql.toml", &config.PostgresqlConfig); err != nil {
		// The PostgresSQL configuration file could not be decoded. Panic with the error message.
		panic("Failed to load PostgresSQL configuration file: " + err.Error())
	}

	// Load Redis configuration
	if _, err := toml.DecodeFile("./conf/service/redis.toml", &config.RedisConfig); err != nil {
		// The Redis configuration file could not be decoded. Panic with the error message.
		panic("Failed to load Redis configuration file: " + err.Error())
	}

	if _, err := toml.DecodeFile("./conf/service/tdengine.toml", &config.TDengineConfig); err != nil {
		// The TDengine configuration file could not be decoded. Panic with the error message.
		panic("Failed to load TDengine configuration file: " + err.Error())
	}

	// Load Cron configuration
	if _, err := toml.DecodeFile("./conf/service/cron.toml", &config.CronConfig); err != nil {
		// The Cron configuration file could not be decoded. Panic with the error message.
		panic("Failed to load Cron configuration file: " + err.Error())
	}
}
