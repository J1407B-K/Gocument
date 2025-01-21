package model

type User struct {
	Username string `json:"username" gorm:"primaryKey;size:255;not null"`
	Password string `json:"password" gorm:"size:255;unique;not null"`
}
