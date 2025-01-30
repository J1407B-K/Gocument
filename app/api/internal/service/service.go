package service

import (
	"Gocument/app/api/global"
	"Gocument/app/api/internal/consts"
	"Gocument/app/api/internal/dao"
	"Gocument/app/api/internal/middle"
	"Gocument/app/api/internal/model"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tencentyun/cos-go-sdk-v5"
	"golang.org/x/crypto/bcrypt"
	"mime/multipart"
	"net/http"
	"path/filepath"
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
		err := dao.SetRedisKey(redisName, hashedPassword)
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
		err := dao.SetRedisKey(redisName, hashedPassword)
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

// 之后加入Oauth
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
		var userinMysql *model.User
		userinMysql, err := dao.SelectUser(user.Username)
		if err != nil {
			global.Logger.Error("mysql select user failed" + err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": consts.MysqlQueryFailed,
				"msg":  "mysql select user failed" + err.Error(),
			})
			return
		}

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

		err = dao.SetRedisKey(redisKey, userinMysql.Password)
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

// 提供接口，前端上传用户头像(public)
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
	if ext != ".jpg" {
		global.Logger.Debug("avatar verify failed")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.AvatarQueryFailed,
			"msg":  "avatar verify failed(.jpg)",
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

	//拼接成路径
	cosPath := fmt.Sprintf("avatars/%s/%s", username, username.(string)+ext)

	err = dao.StoreMetaFile(username.(string), cosPath, username.(string)+".jpg", "public")
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
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "upload success",
	})
}

// 上传文档接口(用户创建时调用)
func UploadDocument(c *gin.Context) {
	// 获取文件和需要的权限
	file, err := c.FormFile("document")
	if err != nil {
		global.Logger.Error("get file failed: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.DocxQueryFailed,
			"msg":  "document upload failed: " + err.Error(),
		})
		return
	}

	visibility := c.DefaultPostForm("visibility", "public")

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

	// 生成唯一路径（之后想如何加入时间戳允许重名）
	cosPath := fmt.Sprintf("documents/%s/%s", username, file.Filename)

	err = dao.StoreMetaFile(username.(string), cosPath, file.Filename, visibility)
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

	err = createFileAccess(username.(string), file.Filename)
	if err != nil {
		global.Logger.Error("create file access failed" + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.CreateFileAccessFail,
			"msg":  "create file access failed" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "document uploaded successfully",
	})
}

func DeleteDocument(c *gin.Context) {
	// 获取文件名(包含后缀)
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
	MetaFile, err := dao.SelectMetaFile(filename)
	if err != nil {
		global.Logger.Error("file not found in database: " + err.Error())
		c.JSON(http.StatusNotFound, gin.H{
			"code": consts.FileNotFind,
			"msg":  "file not found",
		})
		return
	}

	username, exist := c.Get("username")
	if !exist {
		global.Logger.Error("not found username in middle")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.NotFoundUserInMiddle,
			"msg":  "not found username in middle",
		})
		return
	}

	//鉴权（是否为拥有者）
	if MetaFile.Username == username.(string) {
		err = deleteFromCOS(global.CosClient, MetaFile.FileURL)
		if err != nil {
			global.Logger.Error("delete from cos failed: " + err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": consts.DeleteFileWrong,
				"msg":  "delete from cos failed:" + err.Error(),
			})
			return
		}
	}
	// 从数据库删除文件元数据
	err = dao.DeleteMetafile(filename)
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
func UpdateDocument(c *gin.Context) {
	//获取文件名
	filename := c.DefaultQuery("filename", "")
	if filename == "" {
		global.Logger.Error("filename is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.FilenameMissing,
			"msg":  "filename is required",
		})
		return
	}

	//新文件名（可选）
	newfilename := c.DefaultQuery("newfilename", filename)

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

	//查修改权限
	fileAccesses, err := selectFileAccess(filename)
	if err != nil {
		global.Logger.Error("check user change file failed: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.UserCannotChangeFile,
			"msg":  "user change file failed: " + err.Error(),
		})
		return
	}

	username, exist := c.Get("username")
	if !exist {
		global.Logger.Error("not found username in middle")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.NotFoundUserInMiddle,
			"msg":  "not found username in middle",
		})
		return
	}

	//找元数据
	Metafile, err := dao.SelectMetaFile(filename)
	if err != nil {
		global.Logger.Error("file not found in database: " + err.Error())
		c.JSON(http.StatusNotFound, gin.H{
			"code": consts.FileNotFind,
			"msg":  "file not found",
		})
		return
	}

	//遍历查询是否有权限修改
	for _, fileAccess := range fileAccesses {
		//有匹配权限
		if fileAccess.Username == username.(string) {
			//先从COS上删除
			err = deleteFromCOS(global.CosClient, Metafile.FileURL)
			if err != nil {
				global.Logger.Error("delete from cos failed: " + err.Error())
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": consts.DeleteFileWrong,
					"msg":  "delete from cos failed:" + err.Error(),
				})
				return
			}

			username, exist := c.Get("username")
			if !exist {
				global.Logger.Error("not found username in middle")
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": consts.NotFoundUserInMiddle,
					"msg":  "not found username in middle",
				})
			}

			//生成新URL
			cosPath := fmt.Sprintf("documents/%s/%s", username, newfilename)

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

			//新文件名保存
			err = dao.UpdateMetaFileName(Metafile, newfilename)
			if err != nil {
				global.Logger.Error("update Metafile url failed: " + err.Error())
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": consts.MysqlSaveWrong,
					"msg":  "update Metafile url failed: " + err.Error(),
				})
			}

			//新URL保存
			err = dao.UpdateMetaFileURL(Metafile, cosPath)
			if err != nil {
				global.Logger.Error("update Metafile url failed: " + err.Error())
				c.JSON(http.StatusInternalServerError, gin.H{
					"code": consts.MysqlSaveWrong,
					"msg":  "update Metafile url failed: " + err.Error(),
				})
			}

			global.Logger.Info("update file successfully")
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "file updated successfully",
			})
			return
		}
	}

	//遍历后仍未找到符合权限
	global.Logger.Debug("User CannotChange file")
	c.JSON(http.StatusNotFound, gin.H{
		"code": consts.UserCannotChangeFile,
		"msg":  "User CannotChange file",
	})

}

func GetDocument(c *gin.Context) {
	filename := c.DefaultQuery("filename", "")
	if filename == "" {
		global.Logger.Error("filename is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.FilenameMissing,
			"msg":  "filename is required",
		})
		return
	}

	usernameNow, exist := c.Get("username")
	if !exist {
		global.Logger.Error("not found username in middle")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.NotFoundUserInMiddle,
			"msg":  "not found username in middle",
		})
		return
	}

	//鉴权
	ok, err := checkFilePermission(filename, usernameNow.(string))
	if !ok {
		if err != nil {
			global.Logger.Error("check permission failed: " + err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": consts.VisibilityWrong,
				"msg":  "check permission failed: " + err.Error(),
			})
			return
		} else {
		}
		global.Logger.Debug("visibility not correct")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.VisibilityNotCorrect,
			"msg":  "visibility not correct",
		})
		return
	}

	metaFile, err := dao.SelectMetaFile(filename)
	if err != nil {
		global.Logger.Error("metafile not found in database: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.FileNotFind,
			"msg":  "metafile not found in database",
		})
		return
	}

	//直接返回COS连接
	cosPath := metaFile.FileURL
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "document get successfully",
		"URL":  cosPath,
	})
}

func GetAvatar(c *gin.Context) {
	username := c.DefaultQuery("username", "")
	if username == "" {
		global.Logger.Error("username is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.UsernameMissing,
			"msg":  "username is required",
		})
		return
	}

	filename := username + ".jpg"

	metaFile, err := dao.SelectMetaFile(filename)
	if err != nil {
		global.Logger.Error("metafile not found in database: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.FileNotFind,
			"msg":  "metafile not found in database",
		})
		return
	}

	//直接返回COS连接
	cosPath := metaFile.FileURL
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "avatar get successfully",
		"URL":  cosPath,
	})
}

func SelectUserInfo(c *gin.Context) {
	username := c.DefaultQuery("username", "")
	if username == "" {
		global.Logger.Error("username is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": consts.UsernameMissing,
			"msg":  "username is required",
		})
		return
	}

	userInfo, err := dao.SelectUser(username)
	userInfo.Password = "-"

	if err != nil {
		global.Logger.Error("user not found in database: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.MysqlQueryFailed,
			"msg":  "user not found in database" + err.Error(),
		})
		return
	}

	MetaFiles, err := dao.SelectMetaFileByUsername(username)
	if err != nil {
		global.Logger.Error("metafile not found in database: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": consts.MysqlQueryFailed,
			"msg":  "metafile not found in database" + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"msg":       "Info selecting successfully",
		"User":      userInfo,
		"MetaFiles": MetaFiles,
	})
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

// 检查查看权限的辅助函数
func checkFilePermission(filename, usernameNow string) (bool, error) {
	metaFile, err := dao.SelectMetaFile(filename)
	if err != nil {
		global.Logger.Error("Metafile not found/failed in database: " + err.Error())
		return false, err
	}

	// 检查权限
	if metaFile.Visibility == "public" {
		return true, nil // 公开文件(所有人都可看，持有fileAccess可以修改)
	}
	if metaFile.Visibility == "private" && metaFile.Username == usernameNow {
		return true, nil // 文件所有者（只有所有者可看可修改）
	}
	if metaFile.Visibility == "restricted" {

		//**持有fileAccess、所有者可以看、修改(之后)

		return false, nil
	} else {
		global.Logger.Error("meta file is not public or private or restricted/something wrong")
		return false, errors.New("meta file is not public or private or restricted/something wrong")
	}
}

func createFileAccess(username, filename string) error {
	ok := dao.CreateFileAccess(username, filename)
	if !ok {
		global.Logger.Error("create file access failed")
		return errors.New("create file access failed")
	}
	return nil
}

// 检查能否修改文档的辅助函数
func selectFileAccess(filename string) ([]model.FileAccess, error) {
	var fileAccessess []model.FileAccess

	fileAccessess, err := dao.SelectFileAccess(filename)
	if err != nil {
		global.Logger.Error("Select fileAccess failed: " + err.Error())
		return nil, err
	}
	return fileAccessess, nil
}

// 修改文档隐私性的辅助函数
func updateMetaFileVisibility(filename, usernameNow, newVisibility string) error {
	metaFile, err := dao.CheckUserAndFilename(filename, usernameNow)
	if err != nil {
		global.Logger.Error("CheckUserAndFilename failed: " + err.Error())
		return err
	}

	err = dao.UpdateMetaFileVisibility(metaFile, newVisibility)
	if err != nil {
		global.Logger.Error("Update MetaFile Visibility failed: " + err.Error())
		return err
	}

	return nil
}
