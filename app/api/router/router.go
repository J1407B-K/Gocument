package router

import (
	"Gocument/app/api/global"
	"Gocument/app/api/internal/conn"
	"Gocument/app/api/internal/middle"
	"Gocument/app/api/internal/service"
	"github.com/gin-gonic/gin"
)

func InitRouter() {
	r := gin.Default()

	// WebSocket 路由
	r.GET("/websocket/writer", func(c *gin.Context) {
		conn.BackServer.HandleConnections(c)
	})

	go conn.BackServer.HandleMessages()

	r.POST("/register", service.Register)
	r.POST("/login", service.Login)
	r.GET("/select", service.SelectUserInfo)
	r.GET("/get/avatar", service.GetAvatar)

	p := r.Group("/")
	p.Use(middle.JWTAuthMiddleware())

	p.POST("/upload/avatar", service.UploadAvatar)
	p.POST("/upload/document", service.UploadDocument)
	p.DELETE("/delete/document", service.DeleteDocument)
	p.PUT("/update/document", service.UpdateDocument)
	p.GET("/get/document", service.GetDocument)

	err := r.Run()
	if err != nil {
		global.Logger.Fatal("Initialize router failed" + err.Error())
	}
	global.Logger.Info("Initialize router success")
}
