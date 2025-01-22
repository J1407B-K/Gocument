package global

import (
	"Gocument/app/api/global/config"
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	Ctx       = context.Background()
	Config    *config.Config
	JWTsecret = "lanshan_kq"
	Logger    *zap.Logger
	MysqlDB   *gorm.DB
	RedisDB   *redis.Client
	CosClient *cos.Client
)
