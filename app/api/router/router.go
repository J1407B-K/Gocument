package router

import (
	"Gocument/app/api/global"
	"Gocument/app/api/internal/service"
	"github.com/gin-gonic/gin"
)

func InitRouter() {
	r := gin.Default()

	r.POST("/register", service.Register)
	r.POST("/login", service.Login)

	err := r.Run()
	if err != nil {
		global.Logger.Fatal("Initialize router failed" + err.Error())
	}
	global.Logger.Info("Initialize router success")
}
