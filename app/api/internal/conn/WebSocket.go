package conn

import (
	"Gocument/app/api/global"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"net/http"
	"sync"
	"time"
)

type Server struct {
	Clients   map[string]*Client // 所有连接的客户端
	Broadcast chan []byte        // 广播消息的通道
	Lock      sync.Mutex         // 保护 Clients 的并发锁
}

type Client struct {
	ID     string          // 客户端唯一标识
	Conn   *websocket.Conn // 客户端与服务端的 WebSocket 连接
	Server *Server         // 所属的服务器实例
}

var (
	Upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	BackServer = &Server{
		Clients:   make(map[string]*Client),
		Broadcast: make(chan []byte),
	}
)

// 处理 WebSocket 连接
func (s *Server) HandleConnections(c *gin.Context) {
	conn, err := Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		global.Logger.Error("WebSocket upgrade error", zap.Error(err))
		return
	}

	// 确保在函数退出时关闭连接
	defer func() {
		s.Lock.Lock()
		delete(s.Clients, c.ClientIP())
		s.Lock.Unlock()
		if err := conn.Close(); err != nil {
			global.Logger.Error("Failed to close WebSocket connection", zap.Error(err))
		}
		global.Logger.Info("Client disconnected: " + c.ClientIP())
	}()

	// 实例化客户端
	client := &Client{
		ID:     c.ClientIP(),
		Conn:   conn,
		Server: s,
	}

	// 将客户端加入到服务器
	s.Lock.Lock()
	s.Clients[client.ID] = client
	s.Lock.Unlock()

	global.Logger.Info("Client connected: " + client.ID)

	// 启动 Goroutine 接收消息
	go func(client *Client) {
		for {
			_, msg, err := client.Conn.ReadMessage()
			if err != nil {
				global.Logger.Error("Error reading message from client "+client.ID, zap.Error(err))
				break
			}
			global.Logger.Info("Message received from client " + client.ID + ": " + string(msg))
			s.Broadcast <- msg
		}

		// 客户端断开清理
		s.Lock.Lock()
		delete(s.Clients, client.ID)
		s.Lock.Unlock()
		global.Logger.Info("Client disconnected: " + client.ID)
	}(client)

	// 定期发送 ping 消息保持心跳
	go func() {
		for {
			time.Sleep(30 * time.Second)
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				global.Logger.Error("Failed to send ping", zap.Error(err))
				return
			}
		}
	}()

	// 阻塞当前函数，直到连接关闭
	select {}
}

// 广播消息
func (s *Server) HandleMessages() {
	for {
		msg := <-s.Broadcast
		disconnectedClients := []string{}

		s.Lock.Lock()
		for id, client := range s.Clients {
			if err := client.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				global.Logger.Error("Error sending message to client "+id, zap.Error(err))
				disconnectedClients = append(disconnectedClients, id)
			}
		}

		// 处理断开连接的客户端
		for _, id := range disconnectedClients {
			if client, exists := s.Clients[id]; exists {
				client.Conn.Close()
				delete(s.Clients, id)
			}
		}
		s.Lock.Unlock()
	}
}
