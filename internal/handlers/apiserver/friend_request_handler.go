package apiserver

import (
	"encoding/json"
	"errors"
	"im-go/internal/middleware" // Required for FriendRequestStatusPending etc. if used in handler logic
	"im-go/internal/models"
	"im-go/internal/services"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// FriendRequestHandler handles HTTP requests related to friend requests.
type FriendRequestHandler struct {
	friendService services.FriendRequestService
}

// NewFriendRequestHandler creates a new FriendRequestHandler.
func NewFriendRequestHandler(fs services.FriendRequestService) *FriendRequestHandler {
	return &FriendRequestHandler{friendService: fs}
}

// SendFriendRequestPayload defines the expected JSON body for sending a friend request.
type SendFriendRequestPayload struct {
	RecipientID uint `json:"recipientId"`
}

// SendFriendRequestHandler handles POST /api/v1/friend-requests
func (h *FriendRequestHandler) SendFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	requesterID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "无法从上下文中获取用户ID", http.StatusUnauthorized)
		return
	}

	var payload SendFriendRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, "请求体无效", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if payload.RecipientID == 0 {
		writeJSONError(w, "缺少接收者ID (recipientId)", http.StatusBadRequest)
		return
	}

	err := h.friendService.SendFriendRequest(r.Context(), requesterID, payload.RecipientID)
	if err != nil {
		if errors.Is(err, services.ErrFriendRequestSelf) || errors.Is(err, services.ErrRecipientNotFound) || errors.Is(err, services.ErrAlreadyFriends) {
			writeJSONError(w, err.Error(), http.StatusBadRequest) // Bad request for these known business errors
		} else if errors.Is(err, services.ErrFriendRequestExists) {
			writeJSONError(w, err.Error(), http.StatusConflict) // Conflict if request already exists
		} else {
			log.Printf("Error sending friend request from %d to %d: %v", requesterID, payload.RecipientID, err)
			writeJSONError(w, "发送好友请求失败", http.StatusInternalServerError)
		}
		return
	}
	writeJSONResponse(w, http.StatusAccepted, map[string]string{"message": "好友请求已发送处理"})
}

// AcceptFriendRequestHandler handles POST /api/v1/friend-requests/{requestID}/accept
func (h *FriendRequestHandler) AcceptFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	recipientUserID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "无法从上下文中获取用户ID", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	requestIDStr, ok := vars["requestID"]
	if !ok {
		writeJSONError(w, "缺少好友请求ID", http.StatusBadRequest)
		return
	}
	requestID, err := strconv.ParseUint(requestIDStr, 10, 32)
	if err != nil {
		writeJSONError(w, "无效的好友请求ID格式", http.StatusBadRequest)
		return
	}

	if err := h.friendService.AcceptFriendRequest(r.Context(), recipientUserID, uint(requestID)); err != nil {
		if errors.Is(err, services.ErrFriendRequestNotFound) || errors.Is(err, services.ErrNotRecipientOfRequest) || errors.Is(err, services.ErrRequestNotPending) {
			writeJSONError(w, err.Error(), http.StatusForbidden) // Or StatusBadRequest depending on policy
		} else if errors.Is(err, services.ErrFriendshipExists) {
			// This case might be treated as success by some, or a specific conflict.
			// For now, let's assume it's a conflict if the service layer didn't handle it as a success.
			writeJSONError(w, err.Error(), http.StatusConflict)
		} else {
			log.Printf("Error accepting friend request %d by user %d: %v", requestID, recipientUserID, err)
			writeJSONError(w, "处理好友请求失败", http.StatusInternalServerError)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "好友请求已接受"})
}

// ListPendingRequestsHandler handles GET /api/v1/friend-requests/pending
func (h *FriendRequestHandler) ListPendingRequestsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "无法从上下文中获取用户ID", http.StatusUnauthorized)
		return
	}

	pendingRequests, err := h.friendService.ListPendingRequests(r.Context(), userID)
	if err != nil {
		log.Printf("Error fetching pending requests for user %d: %v", userID, err)
		writeJSONError(w, "获取待处理请求失败", http.StatusInternalServerError)
		return
	}

	// If pendingRequests is nil (which shouldn't happen if service returns empty slice), make it an empty slice for JSON marshalling.
	if pendingRequests == nil {
		pendingRequests = []*models.FriendRequestWithRequester{}
	}

	writeJSONResponse(w, http.StatusOK, pendingRequests)
}

// RejectFriendRequestHandler handles POST /api/v1/friend-requests/{requestID}/reject
func (h *FriendRequestHandler) RejectFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	recipientUserID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "无法从上下文中获取用户ID", http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	requestIDStr, ok := vars["requestID"]
	if !ok {
		writeJSONError(w, "缺少好友请求ID", http.StatusBadRequest)
		return
	}
	requestID, err := strconv.ParseUint(requestIDStr, 10, 32)
	if err != nil {
		writeJSONError(w, "无效的好友请求ID格式", http.StatusBadRequest)
		return
	}

	if err := h.friendService.RejectFriendRequest(r.Context(), recipientUserID, uint(requestID)); err != nil {
		if errors.Is(err, services.ErrFriendRequestNotFound) || errors.Is(err, services.ErrNotRecipientOfRequest) || errors.Is(err, services.ErrRequestNotPending) {
			writeJSONError(w, err.Error(), http.StatusForbidden)
		} else {
			log.Printf("Error rejecting friend request %d by user %d: %v", requestID, recipientUserID, err)
			writeJSONError(w, "处理好友请求失败", http.StatusInternalServerError)
		}
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "好友请求已拒绝"})
}

// ListFriendsHandler handles GET /api/v1/friends
func (h *FriendRequestHandler) ListFriendsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, "无法从上下文中获取用户ID", http.StatusUnauthorized)
		return
	}

	friendsList, err := h.friendService.GetFriendsList(r.Context(), userID)
	if err != nil {
		log.Printf("Error fetching friends list for user %d: %v", userID, err)
		writeJSONError(w, "获取好友列表失败", http.StatusInternalServerError)
		return
	}

	if friendsList == nil {
		friendsList = []*models.UserBasicInfo{} // Ensure empty list, not null, for JSON
	}

	writeJSONResponse(w, http.StatusOK, friendsList)
}

// TODO: Add handlers for rejecting, listing pending requests etc.
