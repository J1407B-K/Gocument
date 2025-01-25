package model

import "time"

type FileAccess struct {
	FileAccessID int       `gorm:"primaryKey;autoIncrement"`
	FileName     string    `gorm:"type:varchar(50);character set utf8mb4;collate utf8mb4_unicode_ci"` // 文件名
	Username     string    `gorm:"type:varchar(50);character set utf8mb4;collate utf8mb4_unicode_ci"` // 白名单用户
	CreatedAt    time.Time `gorm:"autoCreateTime"`                                                    // 创建时间
}
