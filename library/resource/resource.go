package resource

import (
	"github.com/go-co-op/gocron/v2"
	"github.com/olivere/elastic/v7"
	"go.uber.org/zap"
)

var Corn gocron.Scheduler

var LoggerService *zap.Logger
var ESClient *elastic.Client
