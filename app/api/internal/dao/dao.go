package dao

import (
	"Gocument/app/api/global"
	"Gocument/app/api/internal/model"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

func CreateFileAccess(username, filename string) bool {
	var fileAccess model.FileAccess
	fileAccess.Username = username
	fileAccess.FileName = filename

	global.MysqlDB.Create(&fileAccess)
	return true
}

func CheckUserInRedis(username string) (bool, error) {
	redisName := "user:" + username

	_, err := global.RedisDB.Get(global.Ctx, redisName).Result()
	if err == nil {
		//键值对存在
		return true, nil
	} else if err != redis.Nil {
		//发生其他错误
		global.Logger.Error("redis query wrong")
		return false, err
	}
	//键值对不存在
	return false, nil
}

func CheckUserInMysql(username string) (bool, error) {
	var user model.User
	result := global.MysqlDB.Where("username = ?", username).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			//未查询到记录
			global.Logger.Debug("user does not exist")
			return false, nil
		} else {
			//mysql错误(第三方错误需要转化)
			global.Logger.Error("Mysql failed to query existing user", zap.Error(result.Error))
			return false, result.Error
		}
	} else {
		return true, nil
	}
}

func CheckUserAndFilename(filename, username string) (*model.File, error) {
	var file model.File

	err := global.MysqlDB.Where("file_name = ? AND username = ?", filename, username).First(&file).Error
	if err != nil {
		return &model.File{}, err
	}
	return &file, nil
}

func UserRegister(username string, password string) error {
	var user model.User
	user.Username = username
	user.Password = password

	result := global.MysqlDB.Create(&user)
	if result.Error != nil {
		global.Logger.Error("Mysql failed to create user", zap.Error(result.Error))
		return result.Error
	}
	return nil
}

func StoreMetaFile(username, fileURL, filename, visibility string) error {
	var file model.File
	file.Username = username
	file.FileURL = fileURL
	file.FileName = filename
	file.HelpFileNameUpdater = filename
	file.Visibility = visibility

	result := global.MysqlDB.Create(&file)
	if result.Error != nil {
		global.Logger.Error("Mysql failed to create file", zap.Error(result.Error))
		return result.Error
	}
	return nil
}

func SelectUser(username string) (*model.User, error) {
	var user model.User
	result := global.MysqlDB.Where("username = ?", username).First(&user)
	if result.Error != nil {
		global.Logger.Error("Mysql failed to query existing user", zap.Error(result.Error))
		return &model.User{}, result.Error
	}
	return &user, nil
}

func SelectMetaFile(filename string) (*model.File, error) {
	var file model.File
	//查询file(Mysql)
	err := global.MysqlDB.Where("file_name = ?", filename).First(&file).Error
	if err != nil {
		global.Logger.Error("Mysql failed to query meta file", zap.Error(err))
		return &model.File{}, err
	}
	return &file, nil
}

func SelectMetaFileByUsername(username string) ([]model.File, error) {
	var files []model.File
	err := global.MysqlDB.Where("username = ?", username).Find(&files).Error
	if err != nil {
		global.Logger.Error("Mysql failed to query meta file", zap.Error(err))
		return nil, err
	}
	return files, nil
}

func SelectFileAccess(filename string) ([]model.FileAccess, error) {
	var fileAccesses []model.FileAccess
	//查询所有符合的fileAccess
	err := global.MysqlDB.Model(&model.FileAccess{}).Where("file_name = ?", filename).Find(&fileAccesses).Error
	if err != nil {
		global.Logger.Error("Mysql failed to query meta file", zap.Error(err))
		return []model.FileAccess{}, err
	}
	return fileAccesses, nil
}

func DeleteMetafile(filename string) error {
	var file model.File
	err := global.MysqlDB.Where("file_name = ?", filename).First(&file).Error
	if err != nil {
		global.Logger.Error("Mysql failed to query meta file", zap.Error(err))
		return err
	}
	err = global.MysqlDB.Delete(&file).Error
	if err != nil {
		global.Logger.Error("Mysql failed to delete meta file", zap.Error(err))
		return err
	}
	return nil
}

// 元数据只改变URL
func UpdateMetaFileURL(MetaFile *model.File, NewURL string) error {
	MetaFile.FileURL = NewURL
	result := global.MysqlDB.Save(MetaFile)
	if result.Error != nil {
		global.Logger.Error("Mysql failed to update meta file", zap.Error(result.Error))
		return result.Error
	}
	return nil
}

// 元数据只改变文件名
func UpdateMetaFileName(MetaFile *model.File, NewFileName string) error {
	result := global.MysqlDB.Model(MetaFile).Update("file_name", NewFileName)
	if result.Error != nil {
		global.Logger.Error("Mysql failed to update meta file", zap.Error(result.Error))
		return result.Error
	}
	return nil
}

func UpdateMetaFileVisibility(MetaFile *model.File, NewVisibility string) error {
	MetaFile.Visibility = NewVisibility
	result := global.MysqlDB.Save(MetaFile)
	if result.Error != nil {
		global.Logger.Error("Mysql failed to update meta file", zap.Error(result.Error))
		return result.Error
	}
	return nil
}
