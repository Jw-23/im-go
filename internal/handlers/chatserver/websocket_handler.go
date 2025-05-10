package chatserver

import (
	"context"
	"fmt"
	"log"
	"net/http"

	// 如果需要在连接时验证 token
	"im-go/internal/auth"
	"im-go/internal/config"
	"im-go/internal/imtypes"
	"im-go/internal/services"
	ws "im-go/internal/websocket"
)

// WebSocketHandler 负责处理 WebSocket 连接请求。
type WebSocketHandler struct {
	hub            *ws.Hub
	messageService services.MessageService
	userService    services.UserService // 可选，例如根据 token 获取用户信息
	cfg            config.Config        // 用于获取 WebSocket 和 Auth 配置
}

// NewWebSocketHandler 创建一个新的 WebSocketHandler 实例。
func NewWebSocketHandler(hub *ws.Hub, msgService services.MessageService, userService services.UserService, cfg config.Config) *WebSocketHandler {
	return &WebSocketHandler{
		hub:            hub,
		messageService: msgService,
		userService:    userService,
		cfg:            cfg,
	}
}

// ServeWS 处理传入的 WebSocket 请求。
// 它将 HTTP 连接升级为 WebSocket 连接，并为该连接创建一个新的客户端。
func (h *WebSocketHandler) ServeWS(w http.ResponseWriter, r *http.Request) {
	// 1. 用户认证 (示例：通过 token 查询参数)
	token := r.URL.Query().Get("token")
	var userID uint = 0 // 默认为0，代表匿名或未认证
	var username string = "anonymous"

	if token != "" {
		claims, err := auth.ValidateToken(token, h.cfg.Auth.JWTSecretKey)
		if err != nil {
			log.Printf("WebSocket 连接尝试失败：令牌无效: %v (令牌: %s)", err, token)
			http.Error(w, fmt.Sprintf("令牌无效: %v", err), http.StatusUnauthorized)
			return
		}
		userID = claims.UserID
		username = claims.Username
		log.Printf("用户 %s (ID: %d) 尝试连接 WebSocket", username, userID)
	} else {
		// 允许匿名连接的场景，或者如果你的应用设计为在 WebSocket 内部进行认证消息交换
		log.Println("匿名用户尝试连接 WebSocket (无令牌)")
		// 如果不允许匿名，则应在此处返回错误:
		// http.Error(w, "缺少认证令牌", http.StatusUnauthorized)
		// return
	}

	// 创建一个回调函数，该函数将捕获 messageService 实例
	rawInputHandler := func(ctx context.Context, input imtypes.RawMessageInput) error {
		if h.messageService == nil {
			log.Println("错误: WebSocketHandler 中的 messageService 未初始化")
			return fmt.Errorf("messageService not available")
		}
		return h.messageService.SendMessage(ctx, input)
	}

	// 将 HTTP 连接升级到 WebSocket
	// 注意：userID 现在会传递给 ServeWsPerConnection，以便 Client 对象可以关联用户
	ws.ServeWsPerConnection(h.hub, rawInputHandler, userID, w, r, h.cfg.WebSocket)
}
