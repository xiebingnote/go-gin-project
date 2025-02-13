package bootstrap

import (
	"context"

	"go-gin-project/library/config"

	"github.com/BurntSushi/toml"
)

// InitConfig initializes the application configuration by loading and
// decoding various service configuration files in TOML format.
//
// This function loads configuration files for different services including
// logging, server, Elasticsearch, MySQL, Redis, Kafka, NSQ, MongoDB, and
// PostgresSQL.
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
		panic(err.Error())
	}

	// Load Server configuration
	if _, err := toml.DecodeFile("./conf/server.toml", &config.ServerConfig); err != nil {
		// The server configuration file could not be decoded. Panic with the error message.
		panic(err.Error())
	}

	// Load Elasticsearch configuration
	if _, err := toml.DecodeFile("./conf/servicer/es.toml", &config.ESConfig); err != nil {
		// The Elasticsearch configuration file could not be decoded. Panic with the error message.
		panic(err.Error())
	}

	if _, err := toml.DecodeFile("./conf/servicer/etcd.toml", &config.EtcdConfig); err != nil {
		// The etcd configuration file could not be decoded. Panic with the error message.
		panic(err.Error())
	}

	// Load Kafka configuration
	if _, err := toml.DecodeFile("./conf/servicer/kafka.toml", &config.KafkaConfig); err != nil {
		// The Kafka configuration file could not be decoded. Panic with the error message.
		panic(err.Error())
	}

	// Load MongoDB configuration
	if _, err := toml.DecodeFile("./conf/servicer/mongodb.toml", &config.MongoConfig); err != nil {
		// The MongoDB configuration file could not be decoded. Panic with the error message.
		panic(err.Error())
	}

	// Load MySQL configuration
	if _, err := toml.DecodeFile("./conf/servicer/mysql.toml", &config.MySQLConfig); err != nil {
		// The MySQL configuration file could not be decoded. Panic with the error message.
		panic(err.Error())
	}

	// Load NSQ configuration
	if _, err := toml.DecodeFile("./conf/servicer/nsq.toml", &config.NsqConfig); err != nil {
		// The NSQ configuration file could not be decoded. Panic with the error message.
		panic(err.Error())
	}

	// Load PostgresSQL configuration
	if _, err := toml.DecodeFile("./conf/servicer/postgresql.toml", &config.PostgresqlConfig); err != nil {
		// The PostgresSQL configuration file could not be decoded. Panic with the error message.
		panic(err.Error())
	}

	// Load Redis configuration
	if _, err := toml.DecodeFile("./conf/servicer/redis.toml", &config.RedisConfig); err != nil {
		// The Redis configuration file could not be decoded. Panic with the error message.
		panic(err.Error())
	}

}
