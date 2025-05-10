package websocket

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"im-go/internal/config"  // 用于 WebSocketConfig
	"im-go/internal/imtypes" // 导入新的 imtypes 包

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte("\n")
	space   = []byte(" ")
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// Authenticated User ID for this client.
	UserID uint `json:"userId"`

	// Callback to handle incoming messages, converting them to RawMessageInput
	handleMessage func(ctx context.Context, input imtypes.RawMessageInput) error `json:"-"`
}

// readPump pumps messages from the websocket connection to the handleMessage callback.
func (c *Client) readPump(wsCfg config.WebSocketConfig) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(int64(wsCfg.MaxMessageSizeBytes))
	c.conn.SetReadDeadline(time.Now().Add(time.Duration(wsCfg.PongWaitSeconds) * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(time.Duration(wsCfg.PongWaitSeconds) * time.Second))
		return nil
	})

	for {
		messageType, rawWebsocketMessage, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket 错误 (客户端: %d): %v", c.UserID, err)
			} else {
				log.Printf("WebSocket 读取消息错误 (客户端: %d): %v", c.UserID, err)
			}
			break
		}

		if messageType != websocket.TextMessage {
			log.Printf("警告: 客户端 %d 发送了非文本消息类型: %d", c.UserID, messageType)
			continue
		}

		var clientReceivedWsMsg imtypes.Message
		if err := json.Unmarshal(rawWebsocketMessage, &clientReceivedWsMsg); err != nil {
			log.Printf("错误: 无法反序列化来自客户端 %d 的JSON: %v, 原始消息: %s", c.UserID, err, string(rawWebsocketMessage))
			continue
		}

		switch clientReceivedWsMsg.Type {
		case "text":
			// Content 是普通文本，直接使用
			log.Printf("收到文本消息: %s", clientReceivedWsMsg.Content)
			// ... 处理文本消息 ...

		case "image": // 假设 "image" 类型表示 Content 是 Base64 编码的图片数据字符串
			imgData, err := base64.StdEncoding.DecodeString(clientReceivedWsMsg.Content)
			if err != nil {
				log.Printf("错误: 无法解码来自客户端 %d 的图片消息 Base64内容: %v", c.UserID, err)
				continue
			}
			// 现在 imgData 是 []byte 类型，可以进行处理 (如保存文件等)
			log.Printf("收到图片数据，大小: %d bytes", len(imgData))
			// ... 处理图片字节 ...

		case "file": // 类似地，如果文件内容也是 Base64 编码的字符串
			fileData, err := base64.StdEncoding.DecodeString(clientReceivedWsMsg.Content)
			if err != nil {
				log.Printf("错误: 无法解码来自客户端 %d 的文件消息 Base64内容: %v", c.UserID, err)
				continue
			}
			log.Printf("收到文件数据，大小: %d bytes, 文件名: %s", len(fileData), clientReceivedWsMsg.FileName)
			// ... 处理文件字节 ...

		default:
			log.Printf("收到未知类型的消息: %s", clientReceivedWsMsg.Type)
		}

		// 转换为 RawMessageInput DTO
		rawInputDto := imtypes.RawMessageInput{ // 使用 imtypes.RawMessageInput
			ID:         clientReceivedWsMsg.ID,
			Type:       string(clientReceivedWsMsg.Type),
			Content:    []byte(clientReceivedWsMsg.Content),
			SenderID:   strconv.FormatUint(uint64(c.UserID), 10), // 服务端填充认证过的 SenderID
			ReceiverID: clientReceivedWsMsg.ReceiverID,
			Timestamp:  time.Now(), // 服务端接收时间
			FileName:   clientReceivedWsMsg.FileName,
			FileSize:   clientReceivedWsMsg.FileSize,
		}

		if c.handleMessage != nil {
			if err := c.handleMessage(context.Background(), rawInputDto); err != nil {
				log.Printf("错误: 客户端 %d 通过 handleMessage 发送消息失败: %v", c.UserID, err)
			}
		} else {
			log.Printf("警告: Client %d 的 handleMessage 未初始化，消息未处理。", c.UserID)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
func (c *Client) writePump(wsCfg config.WebSocketConfig) {
	ticker := time.NewTicker(time.Duration(wsCfg.PingPeriodSeconds) * time.Second)
	newlineBytes := []byte("\n") // 定义 newline
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(time.Duration(wsCfg.WriteWaitSeconds) * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message) // 发送第一条消息

			// 尝试聚合发送队列中的其他消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newlineBytes) // 使用定义的 newlineBytes
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(time.Duration(wsCfg.WriteWaitSeconds) * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWsPerConnection 处理来自对等方的 websocket 请求。
func ServeWsPerConnection(hub *Hub, rawInputHandler func(ctx context.Context, input imtypes.RawMessageInput) error, userID uint, w http.ResponseWriter, r *http.Request, wsCfg config.WebSocketConfig) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  int(wsCfg.MaxMessageSizeBytes),
		WriteBufferSize: int(wsCfg.MaxMessageSizeBytes),
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ServeWsPerConnection - Upgrade失败:", err)
		return
	}
	client := &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, 256),
		UserID:        userID,
		handleMessage: rawInputHandler, // 使用新的回调函数
	}
	client.hub.register <- client

	go client.writePump(wsCfg)
	go client.readPump(wsCfg)

	log.Printf("客户端已连接: UserID %d", userID)
}

// 注意：旧的 ServeWs 函数如果不再使用，可以移除或标记为弃用。
// func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) { ... }
