const API_BASE_URL = 'http://localhost:8081';

async function request(endpoint, options = {}) {
  // Use relative path for endpoints if proxy is configured
  // const url = `${API_BASE_URL}${endpoint}`;
  const url = endpoint; // Assuming proxy handles the full path now

  const token = localStorage.getItem('jwtToken');
  const headers = {
    'Content-Type': 'application/json',
    ...options.headers,
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const config = {
    ...options,
    headers,
  };
  try {
    const response = await fetch(url, config);
    // Try to parse JSON even for non-ok responses, as API might return error details
    const responseData = await response.json().catch(() => null); 

    if (!response.ok) {
      const errorMsg = responseData?.error || `HTTP error! status: ${response.status}`;
      return { error: errorMsg, status: response.status };
    }
    if (response.status === 204) { // No Content
      return { success: true, data: null };
    }
    return { success: true, data: responseData }; // Use parsed data
  } catch (error) {
    console.error('API request error:', url, config, error);
    return { error: error.message || 'Network error or an unexpected issue occurred.', status: null };
  }
}

// --- 认证 API ---
export const registerUser = (userData) => request('/auth/register', { method: 'POST', body: JSON.stringify(userData) });
export const loginUser = (credentials) => request('/auth/login', { method: 'POST', body: JSON.stringify(credentials) });

// --- 用户 API ---
export const getCurrentUserProfile = () => request('/api/v1/users/me', { method: 'GET' });
export const searchUsers = (query) => request(`/api/v1/users/search?query=${encodeURIComponent(query)}`, { method: 'GET' });

// --- 好友请求 API ---
export const sendFriendRequest = (recipientId) => request('/api/v1/friend-requests', { method: 'POST', body: JSON.stringify({ recipientId }) });
export const getPendingFriendRequests = () => request('/api/v1/friend-requests/pending', { method: 'GET' });
export const acceptFriendRequest = (requestId) => request(`/api/v1/friend-requests/${requestId}/accept`, { method: 'POST' });
export const rejectFriendRequest = (requestId) => request(`/api/v1/friend-requests/${requestId}/reject`, { method: 'POST' }); // Or 'DELETE' or 'PUT' depending on API design

// --- Contacts/Friends API --- 
export const getFriendsList = () => request('/api/v1/friends', { method: 'GET' });

// --- Conversation API ---
export const initiatePrivateConversation = (contactId) => request('/api/v1/conversations/private', { method: 'POST', body: JSON.stringify({ targetId: contactId }) });

// ADDED: Function to get all conversations for the current user
export const getConversations = () => request('/api/v1/conversations', { method: 'GET' });

// ADDED: Function to get messages for a conversation
export const getConversationMessages = (conversationId, limit, offset) => {
  let url = `/api/v1/conversations/${conversationId}/messages`;
  const params = new URLSearchParams();
  if (limit !== undefined) params.append('limit', limit);
  if (offset !== undefined) params.append('offset', offset);
  if (params.toString()) {
    url += `?${params.toString()}`;
  }
  return request(url, { method: 'GET' });
};

// --- 更多 API 函数将在此处添加 --- 