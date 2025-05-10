package websocket

import (
	"encoding/json"          // Added for marshaling direct messages
	"im-go/internal/imtypes" // Added for message types
	"log"
	"strconv" // Added for parsing UserID
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients, mapping UserID to Client. Assumes one connection per user ID.
	// If multiple connections per user are allowed, this needs to be map[uint][]*Client
	clients map[uint]*Client // Changed map key from *Client to uint (UserID)

	// Inbound messages from the clients for broadcasting (optional now?)
	broadcast chan []byte // Kept for potential future use (e.g. system broadcasts)

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Messages aimed at a specific user.
	direct chan *imtypes.Message // Added channel for direct messages
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uint]*Client),           // Initialize map with uint key
		direct:     make(chan *imtypes.Message, 256), // Initialize direct channel with buffer
	}
}

// DeliverDirectMessage sends a message to the hub for direct delivery.
func (h *Hub) DeliverDirectMessage(msg *imtypes.Message) {
	// Use non-blocking send to prevent blocking the caller (Kafka consumer)
	select {
	case h.direct <- msg:
	default:
		log.Printf("警告: Hub direct channel is full. Dropping message for receiver %s", msg.ReceiverID)
	}
}

// Run starts the hub and listens for messages on its channels.
func (h *Hub) Run() {
	log.Println("WebSocket Hub Run loop started.")
	for {
		select {
		case client := <-h.register:
			// When registering, store the client by UserID.
			// Handle potential collisions if multiple connections are not allowed or overwrite is desired.
			if existingClient, ok := h.clients[client.UserID]; ok {
				log.Printf("警告: 用户 %d 已有连接，关闭旧连接并注册新连接。", client.UserID)
				// Close the old connection's send channel
				close(existingClient.send)
				// Optionally, could send a close message to the old client first
			}
			h.clients[client.UserID] = client
			log.Printf("客户端已注册: UserID %d", client.UserID)

		case client := <-h.unregister:
			// When unregistering, check if the client being removed is the one we have stored.
			if storedClient, ok := h.clients[client.UserID]; ok && storedClient == client {
				delete(h.clients, client.UserID)
				close(client.send)
				log.Printf("客户端已注销: UserID %d", client.UserID)
			} else {
				// If the client isn't the stored one (e.g., an old connection already replaced), just close its send channel.
				// This prevents closing the send channel of a potentially new, valid client connection for the same UserID.
				// We should ensure the client.conn is also closed, which should happen in the client's readPump/writePump defer.
				log.Printf("尝试注销一个不匹配或已过期的客户端连接: UserID %d", client.UserID)
				// Note: closing client.send here might be redundant if the client's pumps handle it.
				// Consider if closing client.send is strictly necessary here for already replaced clients.
				// close(client.send) // Maybe remove this line if client pumps handle closure.
			}

		case messageBytes := <-h.broadcast: // Kept for potential global broadcasts
			log.Println("Hub 收到广播消息 (向所有客户端发送)")
			for userID, client := range h.clients {
				select {
				case client.send <- messageBytes:
				default:
					log.Printf("广播时客户端 %d 的发送通道已满或关闭，移除客户端。", userID)
					close(client.send)
					delete(h.clients, userID)
				}
			}

		case directMsg := <-h.direct:
			receiverIDUint64, err := strconv.ParseUint(directMsg.ReceiverID, 10, 64)
			if err != nil {
				log.Printf("错误: 无法解析直接消息中的 ReceiverID '%s': %v", directMsg.ReceiverID, err)
				continue // Skip this message
			}
			receiverID := uint(receiverIDUint64)

			if client, ok := h.clients[receiverID]; ok {
				// Serialize the message back to bytes before sending
				msgBytes, err := json.Marshal(directMsg)
				if err != nil {
					log.Printf("错误: 无法序列化直接消息以发送给 UserID %d: %v", receiverID, err)
					continue
				}

				// Non-blocking send to the specific client
				select {
				case client.send <- msgBytes:
					// log.Printf("消息已发送给 UserID %d", receiverID) // DEBUG
				default:
					// If the send buffer is full, we assume the client is slow or disconnected.
					// Close the send channel and remove the client from the map.
					log.Printf("警告: UserID %d 的发送通道已满或关闭，移除客户端。", receiverID)
					close(client.send)
					delete(h.clients, receiverID)
				}
			} else {
				// User is not connected to this hub instance.
				// log.Printf("用户 %d 未连接到此 Hub，无法投递直接消息。", receiverID) // Can be noisy
			}
		}
	}
}
