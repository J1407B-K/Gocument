package initialize

import (
	"Gocument/app/api/global"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func SetupDatabase() {
	SetupMysql()
	SetupRedis()
}

func SetupMysql() {
	mysqlConfig := global.Config.DatabaseConfig.MysqlConfig

	//拼接字符串(更加灵活)
	dsn := mysqlConfig.Username + ":" + mysqlConfig.Password + "@tcp(" + mysqlConfig.Addr + ")/" + mysqlConfig.DB + "?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		global.Logger.Fatal("failed to connect mysql" + err.Error())
	}

	//全局变量赋值
	global.MysqlDB = db
	global.Logger.Info("Initialize mysql success")
}

func SetupRedis() {
	rdb := redis.NewClient(&redis.Options{
		Addr:     global.Config.RedisConfig.Addr,
		Password: global.Config.RedisConfig.Password,
		DB:       global.Config.RedisConfig.DB,
	})

	_, err := rdb.Ping(global.Ctx).Result()

	if err != nil {
		global.Logger.Fatal("redis ping failed" + err.Error())
	}

	//全局变量赋值
	global.RedisDB = rdb

	global.Logger.Info("Initialize redis success")
}
