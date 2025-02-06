package resource

import (
	"github.com/go-co-op/gocron/v2"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var Corn gocron.Scheduler
var ESClient *elastic.Client
var LoggerService *zap.Logger
var MySQLClient *gorm.DB
