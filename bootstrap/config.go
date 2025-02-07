package bootstrap

import (
	"project/library/config"

	"github.com/BurntSushi/toml"
)

// initConfig initializes the configuration by loading the configuration files.
//
// This function reads configuration files for various services like logging,
// server, Elasticsearch, MySQL, Redis, and Kafka. Each configuration is decoded
// from a TOML file and assigned to the corresponding global configuration variable.
// If any error occurs during decoding, the function panics with the error message.
func initConfig() {
	// Load Log configuration
	if _, err := toml.DecodeFile("/Users/Mac/GolandProjects/project/conf/log/log.toml", &config.LogConfig); err != nil {
		panic(err.Error())
	}

	// Load Server configuration
	if _, err := toml.DecodeFile("/Users/Mac/GolandProjects/project/conf/server.toml", &config.ServerConfig); err != nil {
		panic(err.Error())
	}

	// Load Elasticsearch configuration
	if _, err := toml.DecodeFile("/Users/Mac/GolandProjects/project/conf/servicer/es.toml", &config.ESConfig); err != nil {
		panic(err.Error())
	}

	// Load MySQL configuration
	if _, err := toml.DecodeFile("/Users/Mac/GolandProjects/project/conf/servicer/mysql.toml", &config.MySQLConfig); err != nil {
		panic(err.Error())
	}

	// Load Redis configuration
	if _, err := toml.DecodeFile("/Users/Mac/GolandProjects/project/conf/servicer/redis.toml", &config.RedisConfig); err != nil {
		panic(err.Error())
	}

	// Load Kafka configuration
	if _, err := toml.DecodeFile("/Users/Mac/GolandProjects/project/conf/servicer/kafka.toml", &config.KafkaConfig); err != nil {
		panic(err.Error())
	}
}
