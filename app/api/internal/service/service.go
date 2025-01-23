package service

import (
	"Gocument/app/api/global"
	"Gocument/app/api/internal/consts"
	"Gocument/app/api/internal/dao"
	"Gocument/app/api/internal/middle"
	"Gocument/app/api/internal/model"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tencentyun/cos-go-sdk-v5"
	"golang.org/x/crypto/bcrypt"
	"mime/multipart"
	"net/http"
	"path/filepath"
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

	redisKey := "user:" + user.Username

	//判断用户是否存在(Redis)
	exist, err := dao.CheckUserInRedis(user.Username)

	//其他redis查询错误
	if !exist && err != nil {
		global.Logger.Error("redis check failed" + err.Error())
	}

	//存在
	if exist && err == nil {
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

		token, err := middle.GenerateToken(user.Username)
		if err != nil {
			global.Logger.Error("generate token failed" + err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": consts.GenerateTokenFailed,
				"msg":  "generate token failed" + err.Error(),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"code":  0,
			"msg":   "login success",
			"token": token,
		})
		return
	}

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
		var userinMysql model.User
		global.MysqlDB.Where("username = ?", user.Username).First(&userinMysql)

		//比较
		err = bcrypt.CompareHashAndPassword([]byte(userinMysql.Password), []byte(user.Password))
		if err != nil {
			global.Logger.Error("password compare failed" + err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": consts.PasswordCompareWrong,
				"msg":  "password compare failed" + err.Error(),
			})
			return
		}

		err := SetRedisKey(redisKey, userinMysql.Password)
		if err != nil {
			global.Logger.Error("redis set failed" + err.Error())
		}

		token, err := middle.GenerateToken(user.Username)
		if err != nil {
			global.Logger.Error("generate token failed" + err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": consts.GenerateTokenFailed,
				"msg":  "generate token failed" + err.Error(),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"code":  0,
			"msg":   "login success",
			"token": token,
		})
		return
	}
}

// 提供接口，前端上传用户头像
func UploadAvatar(c *gin.Context) {
	//获取文件
	file, err := c.FormFile("avatar")
	if err != nil {
		global.Logger.Error("get file failed" + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.AvatarQueryFailed,
			"msg":  "avatar upload failed" + err.Error(),
		})
		return
	}

	//验证文件类型(csdn学的嘿嘿)
	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".png" && ext != ".jpeg" {
		global.Logger.Debug("avatar verify failed")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.AvatarQueryFailed,
			"msg":  "avatar verify failed(.jpg/.png/.jpeg)",
		})
		return
	}

	//生成上传路径
	username, exist := c.Get("username")
	if !exist {
		global.Logger.Error("not found user in middle")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.NotFoundUserInMiddle,
			"msg":  "not found user in middle",
		})
		return
	}

	//加入时间，拼接成唯一路径
	cosPath := fmt.Sprintf("avatars/%s/%s%d%s", username, file.Filename, time.Now().Unix(), ext)

	err = dao.StoreMetaFile(username.(string), cosPath, file.Filename)
	if err != nil {
		global.Logger.Error("save file failed" + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.MysqlSaveWrong,
			"msg":  "save file failed" + err.Error(),
		})
		return
	}

	err = uploadToCOS(global.CosClient, file, cosPath)
	if err != nil {
		global.Logger.Error("upload to cos failed" + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.UploadFileWrong,
			"msg":  "upload to cos failed" + err.Error(),
		})
		return
	}
	global.Logger.Info("avatar upload success")
}

// 上传文档接口
func UploadDocument(c *gin.Context) {
	// 获取文件
	file, err := c.FormFile("document")
	if err != nil {
		global.Logger.Error("get file failed: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.DocxQueryFailed,
			"msg":  "document upload failed: " + err.Error(),
		})
		return
	}

	// 验证文件类型
	ext := filepath.Ext(file.Filename)
	if ext != ".docx" {
		global.Logger.Debug("document verify failed")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.DocxQueryFailed,
			"msg":  "document verify failed (.docx)",
		})
		return
	}

	// 获取用户名
	username, exist := c.Get("username")
	if !exist {
		global.Logger.Error("not found user in middle")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.NotFoundUserInMiddle,
			"msg":  "not found user in middle",
		})
		return
	}

	// 生成唯一路径
	cosPath := fmt.Sprintf("documents/%s/%s%d%s", username, file.Filename, time.Now().Unix(), ext)

	err = dao.StoreMetaFile(username.(string), cosPath, file.Filename)
	if err != nil {
		global.Logger.Error("store document metadata failed: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.MysqlSaveWrong,
			"msg":  "store document metadata failed: " + err.Error(),
		})
		return
	}

	// 上传到 COS
	err = uploadToCOS(global.CosClient, file, cosPath)
	if err != nil {
		global.Logger.Error("upload to COS failed: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.UploadFileWrong,
			"msg":  "upload to COS failed: " + err.Error(),
		})
		return
	}

	// 返回文件 URL
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "document uploaded successfully",
	})
}

func DeleteFile(c *gin.Context) {
	var file model.File
	// 获取文件名
	filename := c.DefaultQuery("filename", "")
	if filename == "" {
		global.Logger.Error("filename is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.FilenameMissing,
			"msg":  "filename is required",
		})
		return
	}

	//之后加redis
	file, err := dao.SelectMetaFile(filename)
	if err != nil {
		global.Logger.Error("file not found in database: " + err.Error())
		c.JSON(http.StatusNotFound, gin.H{
			"code": consts.FileNotFind,
			"msg":  "file not found",
		})
		return
	}

	err = deleteFromCOS(global.CosClient, file.FileURL)
	if err != nil {
		global.Logger.Error("delete from cos failed: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.DeleteFileWrong,
			"msg":  "delete from cos failed:" + err.Error(),
		})
		return
	}

	// 从数据库删除文件元数据(之后放dao)
	err = global.MysqlDB.Where("file_name = ?", filename).Delete(&model.File{}).Error
	if err != nil {
		global.Logger.Error("delete file metadata failed: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.DeleteFileWrong,
			"msg":  "delete file metadata failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "file deleted successfully",
	})
}

// 更新文档(先删除，后更新)
func UpdateFile(c *gin.Context) {
	var file model.File
	filename := c.DefaultQuery("filename", "")
	if filename == "" {
		global.Logger.Error("filename is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.FilenameMissing,
			"msg":  "filename is required",
		})
		return
	}

	//之后加redis
	file, err := dao.SelectMetaFile(filename)
	if err != nil {
		global.Logger.Error("file not found in database: " + err.Error())
		c.JSON(http.StatusNotFound, gin.H{
			"code": consts.FileNotFind,
			"msg":  "file not found",
		})
		return
	}

	err = deleteFromCOS(global.CosClient, file.FileURL)
	if err != nil {
		global.Logger.Error("delete from cos failed: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.DeleteFileWrong,
			"msg":  "delete from cos failed:" + err.Error(),
		})
		return
	}
}

//以下为辅助函数

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

// 上传到腾讯云 COS
func uploadToCOS(client *cos.Client, fileHeader *multipart.FileHeader, cosPath string) error {
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			global.Logger.Error("close file failed" + err.Error())
		}
	}(file)

	_, err = client.Object.Put(global.Ctx, cosPath, file, nil)
	return err
}

// 删除文件的辅助函数
func deleteFromCOS(client *cos.Client, cosPath string) error {
	// 使用 cos.Client 删除文件
	_, err := client.Object.Delete(global.Ctx, cosPath)
	return err
}
