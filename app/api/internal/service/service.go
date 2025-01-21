package service

import (
	"Gocument/app/api/global"
	"Gocument/app/api/internal/consts"
	"Gocument/app/api/internal/dao"
	"Gocument/app/api/internal/model"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"time"
)

func Register(c *gin.Context) {
	var user model.User
	err := c.ShouldBindJSON(&user)
	if err != nil {
		global.Logger.Error("bind user failed" + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.ShouldBindFailed,
			"msg":  "bind user failed" + err.Error(),
		})
		return
	}

	//加密
	hashedPassword, ok := HashedLock(user.Password)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.PasswordHashedWrong,
			"msg":  "password hashed failed",
		})
		return
	}

	redisName := "user:" + user.Username
	//判断用户是否存在(Redis)
	exist, err := dao.CheckUserInRedis(user.Username)
	//用户已存在
	if exist && err == nil {
		global.Logger.Error("user is exist")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.UserAlreadyExist,
			"msg":  "user is exist",
		})
		return
	}

	//其他错误
	if !exist && err != nil {
		global.Logger.Error("redis query failed" + err.Error())
	}

	//判断用户是否存在(Mysql)
	exist, err = dao.CheckUserInMysql(user.Username)

	if err != nil {
		global.Logger.Error("mysql check failed" + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.MysqlQueryFailed,
			"msg":  "mysql check failed" + err.Error(),
		})
		return
	}

	if exist {
		//缓存中没有，Mysql中有
		err := SetRedisKey(redisName, hashedPassword)
		if err != nil {
			global.Logger.Error("redis set failed" + err.Error())
		}

		global.Logger.Info("user already exist")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.UserAlreadyExist,
			"msg":  "user already exist",
		})
		return

	} else {
		//注册用户
		err = dao.UserRegister(user.Username, hashedPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": consts.RegisterFailed,
				"msg":  "user register failed" + err.Error(),
			})
			return
		}

		//保存到redis
		err := SetRedisKey(redisName, hashedPassword)
		if err != nil {
			global.Logger.Error("redis set failed" + err.Error())
		}

		//成功
		global.Logger.Info("user register success")
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"msg":  "user register success",
		})
	}
}

func Login(c *gin.Context) {
	var user model.User
	err := c.ShouldBindJSON(&user)
	if err != nil {
		global.Logger.Error("bind user failed" + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.ShouldBindFailed,
			"msg":  "bind user failed" + err.Error(),
		})
		return
	}

	//判断用户是否存在(Redis)
	exist, err := dao.CheckUserInRedis(user.Username)
	//其他redis查询错误
	if !exist && err != nil {
		global.Logger.Error("redis check failed" + err.Error())
	}
	//不存在
	if !exist {
		global.Logger.Debug("user does not exist")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.UserNotExist,
			"msg":  "user does not exist",
		})
		return
	}

	redisKey := "user:" + user.Username

	//判断用户是否存在(Mysql)
	exist, err = dao.CheckUserInMysql(user.Username)

	//查询出错
	if err != nil {
		global.Logger.Error("mysql check failed" + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.MysqlQueryFailed,
			"msg":  "mysql check failed" + err.Error(),
		})
		return
	}

	//存在，开始登录逻辑
	if exist {
		//提取value
		hashedPassword, err := global.RedisDB.Get(global.Ctx, redisKey).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": consts.RedisQueryFailed,
				"msg":  "redis query failed" + err.Error(),
			})
		}

		//比较
		err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password))
		if err != nil {
			global.Logger.Error("password compare failed" + err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": consts.PasswordCompareWrong,
				"msg":  "password compare failed" + err.Error(),
			})
			return
		}
	}
}

func HashedLock(p string) (string, bool) {
	hashedP, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		global.Logger.Error("bcrypt hashed failed" + err.Error())
		return "", false
	}
	return string(hashedP), true
}

func SetRedisKey(key string, value string) error {
	err := global.RedisDB.Set(global.Ctx, key, value, time.Hour*24).Err()
	if err != nil {
		global.Logger.Error("redis set failed" + err.Error())
		return err
	}
	return nil
}
