package resource

import (
	"database/sql"
	"github.com/IBM/sarama"
	"github.com/casbin/casbin/v2"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/go-co-op/gocron/v2"
	manticore "github.com/manticoresoftware/manticoresearch-go"
	"github.com/nsqio/go-nsq"
	"github.com/olivere/elastic/v7"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	// ClickHouseClient is the ClickHouse client
	ClickHouseClient *sql.DB

	// Corn is the cron scheduler
	Corn gocron.Scheduler

	// ElasticSearchClient is the Elasticsearch client
	ElasticSearchClient *elastic.Client

	// EtcdClient is the etcd client
	EtcdClient *clientv3.Client

	// LoggerService is the logger
	LoggerService *zap.Logger

	// ManticoreClient is the Manticore client
	ManticoreClient *manticore.APIClient

	MongoDBClient *mongo.Database

	// MySQLClient is the MySQL client
	MySQLClient *gorm.DB

	// NsqProducer is the NSQ producer
	NsqProducer []*nsq.Producer

	// NsqConsumer is the NSQ consumer
	NsqConsumer *nsq.Consumer

	// PostgresqlClient is the Postgresql client
	PostgresqlClient *gorm.DB

	// RedisClient is the Redis client
	RedisClient *redis.Client

	// KafkaProducer is the Kafka producer
	KafkaProducer sarama.SyncProducer

	// KafkaConsumer is the Kafka consumer
	KafkaConsumer sarama.Consumer

	// KafkaConsumerGroup is the Kafka consumer
	KafkaConsumerGroup sarama.ConsumerGroup

	// Enforcer is the Casbin enforcer
	Enforcer *casbin.Enforcer

	// TDengineClient is the TDengine client
	TDengineClient *sql.DB

	// TestStringMapSet is the test string map set
	TestStringMapSet *mapset.Set[string]

	// TestCMap is the test concurrent map
	TestCMap *cmap.ConcurrentMap[string, string]
)
