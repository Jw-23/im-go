package apiserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"im-go/internal/middleware"
	"im-go/internal/models"
	"im-go/internal/services"

	"github.com/gorilla/mux"
)

// ConversationHandler 封装了会话相关的 HTTP 处理器方法。
type ConversationHandler struct {
	convoService   services.ConversationService
	messageService services.MessageService // 用于获取会话消息
	groupService   services.GroupService   // ADDED: For fetching group details
}

// NewConversationHandler 创建一个新的 ConversationHandler 实例。
func NewConversationHandler(convoService services.ConversationService, messageService services.MessageService, groupService services.GroupService) *ConversationHandler {
	return &ConversationHandler{
		convoService:   convoService,
		messageService: messageService,
		groupService:   groupService,
	}
}

// GetUserConversationsHandler 获取当前用户的所有会话列表。
func (h *ConversationHandler) GetUserConversationsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "用户未认证", http.StatusUnauthorized)
		return
	}

	limit := 20 // 默认值
	offset := 0 // 默认值

	rawConversations, err := h.convoService.GetUserConversations(r.Context(), userID, limit, offset)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("获取会话列表失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 打印调试信息
	log.Printf("获取到 %d 个会话", len(rawConversations))

	result := make([]map[string]interface{}, 0, len(rawConversations))
	for _, convo := range rawConversations {
		item := make(map[string]interface{})
		item["id"] = convo.ID
		item["type"] = convo.Type
		item["updatedAt"] = convo.UpdatedAt

		// 添加日志，查看会话类型和目标ID
		log.Printf("处理会话ID: %d, 类型: %s, 目标ID: %d",
			convo.ID, convo.Type, convo.TargetID)

		if convo.Type == models.PrivateConversation {
			participants, err := h.convoService.GetConversationParticipants(r.Context(), convo.ID)
			if err == nil {
				for _, p := range participants {
					if p.UserID != userID {
						item["targetId"] = p.UserID
						otherUser, errUserService := h.convoService.GetUserByID(r.Context(), p.UserID)
						if errUserService == nil && otherUser != nil {
							item["name"] = otherUser.Nickname
							item["username"] = otherUser.Username
							item["avatar"] = otherUser.AvatarURL
						} else {
							log.Printf("Error fetching user %d for convo %d: %v", p.UserID, convo.ID, errUserService)
							item["name"] = fmt.Sprintf("User %d", p.UserID)
						}
						break
					}
				}
			} else {
				log.Printf("Error fetching participants for convo %d: %v", convo.ID, err)
				item["targetId"] = nil
				item["name"] = "Unknown User"
			}
		} else if convo.Type == models.GroupConversation {
			if convo.TargetID > 0 {
				groupID := convo.TargetID
				item["targetId"] = groupID

				// 获取群组信息
				group, errGroupService := h.groupService.GetGroupDetailsByID(r.Context(), groupID)
				if errGroupService == nil && group != nil {
					log.Printf("获取群组信息成功, 群组ID: %d, 名称: %s", groupID, group.Name)
					item["name"] = group.Name
					item["avatar"] = group.AvatarURL
					item["description"] = group.Description

					// 获取群组成员数量
					members, _ := h.groupService.GetGroupMembers(r.Context(), groupID, 1000, 0)
					if members != nil {
						item["memberCount"] = len(members)
					}
				} else {
					log.Printf("Error fetching group %d for convo %d: %v", groupID, convo.ID, errGroupService)
					item["name"] = fmt.Sprintf("Group %d", groupID)
				}
			} else {
				log.Printf("Group conversation %d has nil or invalid TargetID (GroupID)", convo.ID)
				item["name"] = "Unknown Group"
			}
		}

		if convo.LastMessageID != nil && *convo.LastMessageID > 0 {
			lastMsg, errMessageService := h.messageService.GetMessageByID(r.Context(), *convo.LastMessageID)
			if errMessageService == nil && lastMsg != nil {
				lastMessageMap := make(map[string]interface{})
				lastMessageMap["id"] = lastMsg.ID
				lastMessageMap["content"] = lastMsg.Content
				lastMessageMap["timestamp"] = lastMsg.SentAt
				lastMessageMap["senderId"] = fmt.Sprint(lastMsg.SenderID)
				item["lastMessage"] = lastMessageMap
			} else {
				log.Printf("Error fetching last message %d for convo %d: %v", *convo.LastMessageID, convo.ID, errMessageService)
				item["lastMessage"] = nil
			}
		} else {
			item["lastMessage"] = nil
		}

		result = append(result, item)
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// CreateOrGetPrivateConversationRequest 是创建/获取私聊会话的请求结构体。
type CreateOrGetPrivateConversationRequest struct {
	TargetId uint `json:"targetId"`
}

// CreateOrGetPrivateConversationHandler 获取或创建与目标用户的私聊会话。
func (h *ConversationHandler) CreateOrGetPrivateConversationHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "用户未认证", http.StatusUnauthorized)
		return
	}

	var req CreateOrGetPrivateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "请求体无效", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if req.TargetId == 0 {
		writeJSONError(w, "目标用户ID不能为空", http.StatusBadRequest)
		return
	}

	conversation, _, err := h.convoService.GetOrCreatePrivateConversation(r.Context(), userID, req.TargetId)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("获取或创建私聊会话失败: %v", err), http.StatusInternalServerError)
		return
	}

	// 新增：组装 targetId 字段
	item := make(map[string]interface{})
	item["id"] = conversation.ID
	item["type"] = conversation.Type
	item["lastMessageId"] = conversation.LastMessageID
	item["updatedAt"] = conversation.UpdatedAt
	// 获取参与者，找出不是自己的那个人
	participants, err := h.convoService.GetConversationParticipants(r.Context(), conversation.ID)
	if err == nil { // Ensure no error fetching participants
		for _, p := range participants {
			if p.UserID != userID { // This is the other user (targetId)
				item["targetId"] = p.UserID // Essential for frontend logic
				otherUser, errUserService := h.convoService.GetUserByID(r.Context(), p.UserID)
				if errUserService == nil && otherUser != nil {
					item["name"] = otherUser.Nickname     // 使用昵称
					item["username"] = otherUser.Username // 添加用户名
					item["avatar"] = otherUser.AvatarURL
				} else {
					// Fallback if user details cannot be fetched
					log.Printf("CreateOrGetPrivateConversationHandler: Error fetching user %d for convo %d: %v", p.UserID, conversation.ID, errUserService)
					// Do not explicitly set item["name"] to "User X" here;
					// if name is missing, frontend's formatApiConversation will handle fallback.
					// However, ensure targetId is present for the fallback to work correctly.
				}
				break // Found the other participant
			}
		}
	} else {
		log.Printf("CreateOrGetPrivateConversationHandler: Error fetching participants for convo %d: %v", conversation.ID, err)
		// If participants cannot be fetched, targetId might be missing or incorrect,
		// which could affect frontend's ability to display even "User X".
		// Consider how to handle this; for now, targetId might remain nil or be based on req.TargetId if appropriate.
		// If relying on req.TargetId:
		// item["targetId"] = req.TargetId // Fallback targetId if participants fetch fails
	}

	// Ensure lastMessage is also populated if available and needed by formatApiConversation
	// This part is similar to GetUserConversationsHandler
	if conversation.LastMessageID != nil && *conversation.LastMessageID > 0 {
		lastMsg, errMessageService := h.messageService.GetMessageByID(r.Context(), *conversation.LastMessageID)
		if errMessageService == nil && lastMsg != nil {
			lastMessageMap := make(map[string]interface{})
			lastMessageMap["id"] = lastMsg.ID
			lastMessageMap["content"] = lastMsg.Content
			lastMessageMap["timestamp"] = lastMsg.SentAt
			lastMessageMap["senderId"] = fmt.Sprint(lastMsg.SenderID)
			item["lastMessage"] = lastMessageMap
		} else {
			log.Printf("CreateOrGetPrivateConversationHandler: Error fetching last message %d for convo %d: %v", *conversation.LastMessageID, conversation.ID, errMessageService)
			item["lastMessage"] = nil
		}
	} else {
		item["lastMessage"] = nil
	}

	writeJSONResponse(w, http.StatusOK, item)
}

// GetConversationMessagesHandler 获取指定会话的消息列表。
func (h *ConversationHandler) GetConversationMessagesHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "用户未认证", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	conversationIDStr, _ := vars["conversationID"]
	conversationID, err := strconv.ParseUint(conversationIDStr, 10, 32)
	if err != nil {
		writeJSONError(w, "无效的会话ID格式", http.StatusBadRequest)
		return
	}

	// 权限检查：确保用户是此会话的成员 (在 GetConversationDetails 中完成)
	_, err = h.convoService.GetConversationDetails(r.Context(), uint(conversationID), userID)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("无权访问会话或会话不存在: %v", err), http.StatusForbidden)
		return
	}

	// TODO: 从查询参数获取分页信息 (limit, offset)
	limit := 50 // 默认值
	offset := 0 // 默认值

	messages, err := h.messageService.GetMessagesForConversation(r.Context(), uint(conversationID), limit, offset)
	if err != nil {
		writeJSONError(w, fmt.Sprintf("获取会话消息失败: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, http.StatusOK, messages)
}
