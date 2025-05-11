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

// --- 群组 API ---
export const createGroup = async (groupData) => {
  console.log('[API] 创建群组请求参数:', groupData);
  const requestBody = {
    name: groupData.name,
    description: groupData.description || '',
    avatarUrl: groupData.avatarUrl || '',
    isPublic: groupData.isPublic || false,
    joinCondition: groupData.joinCondition || '',
    memberIds: groupData.memberIds || []
  };
  console.log('[API] 最终请求体:', JSON.stringify(requestBody));
  
  const response = await request('/api/v1/groups', { 
    method: 'POST', 
    body: JSON.stringify(requestBody)
  });
  
  console.log('[API] 创建群组响应:', response);
  return response;
};

export const getGroupDetails = (groupId) => request(`/api/v1/groups/${groupId}`, { method: 'GET' });

export const joinGroup = (groupId) => request(`/api/v1/groups/${groupId}/join`, { method: 'POST' });

export const leaveGroup = (groupId) => request(`/api/v1/groups/${groupId}/leave`, { method: 'POST' });

export const getGroupMembers = (groupId) => request(`/api/v1/groups/${groupId}/members`, { method: 'GET' });

// 添加修复群组会话参与者的API
export const fixGroupConversationParticipants = (groupId) => {
  console.log(`[API] 修复群组 ${groupId} 的会话参与者`);
  return request(`/api/v1/groups/${groupId}/fix-participants`, { method: 'POST' });
};

// --- Conversation API ---
export const initiatePrivateConversation = async (contactId) => {
  console.log(`[API] 尝试获取/创建与联系人${contactId}的私聊会话`);
  console.log(`[API] 向服务器发送请求前，先检查是否存在与${contactId}的现有会话`);
  // 先尝试获取会话列表
  const conversationsResult = await getConversations();
  if (conversationsResult.success && conversationsResult.data) {
    // 在现有会话中寻找与该联系人的私聊
    const existingConversation = conversationsResult.data.find(conv => 
      conv.type === 'private' && 
      ((conv.participants && conv.participants.some(p => p.id === contactId)) || 
       (conv.otherParticipant && conv.otherParticipant.id === contactId))
    );
    
    if (existingConversation) {
      console.log(`[API] 已找到与联系人${contactId}的现有会话:`, existingConversation.id);
      return { success: true, data: existingConversation };
    }
    console.log(`[API] 未找到与联系人${contactId}的现有会话，将创建新会话`);
  }
  
  // 如果没有找到现有会话，则创建新的
  const response = await request('/api/v1/conversations/private', { 
    method: 'POST', 
    body: JSON.stringify({ targetId: contactId }) 
  });
  console.log(`[API] 创建私聊会话响应:`, response);
  return response;
};

// ADDED: Function to get all conversations for the current user
export const getConversations = async () => {
  console.log('[API] 正在请求会话列表...');
  const response = await request('/api/v1/conversations', { method: 'GET' });
  console.log('[API] 会话列表响应:', response);
  return response;
};

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