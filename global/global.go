package global

import (
	"blog/config"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	Config *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Es     *elasticsearch.TypedClient
	Log    *zap.SugaredLogger
	AddrDB *xdb.Searcher
)
