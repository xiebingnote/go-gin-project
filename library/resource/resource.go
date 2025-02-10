package resource

import (
	"github.com/IBM/sarama"
	"github.com/casbin/casbin/v2"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/go-co-op/gocron/v2"
	"github.com/olivere/elastic/v7"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var Corn gocron.Scheduler
var ESClient *elastic.Client
var LoggerService *zap.Logger
var MySQLClient *gorm.DB
var RedisClient *redis.Client
var KafkaProducer sarama.SyncProducer
var KafkaConsumer sarama.ConsumerGroup
var Enforcer *casbin.Enforcer
var TestStringMapSet *mapset.Set[string]
var TestCMap *cmap.ConcurrentMap[string, string]
