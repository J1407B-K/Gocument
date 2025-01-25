package model

import (
	"gorm.io/gorm"
	"time"
)

type File struct {
	FileID              int        `json:"fileID" gorm:"primaryKey;autoIncrement"`
	FileName            string     `json:"fileName" gorm:"unique;type:varchar(50);character set utf8mb4;collate utf8mb4_unicode_ci"`
	HelpFileNameUpdater string     `json:"help_file_name_updater" gorm:"type:varchar(50);character set utf8mb4;collate utf8mb4_unicode_ci"`
	FileURL             string     `json:"fileURL" gorm:"size:255;unique;not null"`
	Username            string     `json:"username" gorm:"not null"` //创建者
	Visibility          string     `json:"visibility" gorm:"not null;default:'public'"`
	CreatedAt           *time.Time `gorm:"type:datetime(3);default:null;autoCreateTime"` // 创建时间，允许为空
	UpdatedAt           *time.Time `gorm:"type:datetime(3);default:null;autoUpdateTime"` // 更新时间，允许为空，自动更新时间
}

func (f *File) BeforeUpdate(tx *gorm.DB) (err error) {
	// 保存更新前的文件名
	f.HelpFileNameUpdater = f.FileName

	return nil
}

func (f *File) AfterUpdate(tx *gorm.DB) (err error) {
	// 使用更新前的文件名（OldFileName）来更新 FileAccess
	if err := tx.Model(&FileAccess{}).Where("file_name = ?", f.HelpFileNameUpdater).Update("file_name", f.FileName).Error; err != nil {
		return err
	}
	return nil
}
