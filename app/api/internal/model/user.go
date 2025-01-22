package model

type User struct {
	Username string `json:"username" gorm:"primaryKey;type:varchar(50);character set utf8mb4;collate utf8mb4_unicode_ci"`
	Password string `json:"password" gorm:"size:255;not null"`
	Content  string `json:"content" gorm:"size:255"`
}
