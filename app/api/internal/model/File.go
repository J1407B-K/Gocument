package model

import "time"

type File struct {
	FileName   string     `json:"fileName" gorm:"primaryKey;type:varchar(50);character set utf8mb4;collate utf8mb4_unicode_ci"`
	FileURL    string     `json:"fileURL" gorm:"size:255;unique;not null"`
	Username   string     `json:"username" gorm:"not null"`
	Visibility string     `json:"visibility" gorm:"not null;default:'public'"`
	CreatedAt  *time.Time `gorm:"type:datetime(3);default:null;autoCreateTime"` // 创建时间，允许为空
	UpdatedAt  *time.Time `gorm:"type:datetime(3);default:null;autoUpdateTime"` // 更新时间，允许为空，自动更新时间
}
