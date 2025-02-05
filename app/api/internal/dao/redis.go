package dao

import (
	"Gocument/app/api/global"
	"github.com/go-redis/redis/v8"
	"time"
)

func SetRedisKey(key string, value string) error {
	err := global.RedisDB.Set(global.Ctx, key, value, time.Hour*24).Err()
	if err != nil {
		global.Logger.Error("redis set failed" + err.Error())
		return err
	}
	return nil
}

func GetRedisKey(key string) (string, error) {
	val, err := global.RedisDB.Get(global.Ctx, key).Result()
	if err == redis.Nil {
		global.Logger.Info("redis key not found")
		return "", nil
	}
	if err != nil {
		global.Logger.Error("redis get failed" + err.Error())
		return "", err
	}
	return val, nil
}

func DelRedisKey(key string) error {
	err := global.RedisDB.Del(global.Ctx, key).Err()
	if err != nil {
		global.Logger.Error("redis del failed" + err.Error())
		return err
	}
	return nil
}
