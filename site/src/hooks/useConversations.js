import { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { getConversations, initiatePrivateConversation } from '../services/api';

// 格式化 API 返回的会话数据
const formatApiConversation = (apiConvo, currentUserId) => {
  let name = apiConvo.name;
  let avatar = apiConvo.avatarUrl; // Prefer apiConvo.avatarUrl if it exists from enriched API
  let targetId = null;
  let isOnline = false;
  let username = null;
  // console.log("[useConversations]","apiConvo:",apiConvo)

  // 检查会话类型
  const isGroupConversation = apiConvo.type === 'group';
  console.log("[formatApiConversation] 处理会话:", apiConvo.id, "类型:", apiConvo.type, "targetId:", apiConvo.targetId);

  // 处理不同类型的会话数据结构
  if (apiConvo.targetId && apiConvo.type === 'private') {
    targetId = apiConvo.targetId;
    if (!name) name = apiConvo.name;
    if (!avatar) avatar = apiConvo.avatar;
    username = apiConvo.username; // 保存原始用户名

  } else if (apiConvo.type === 'private' && apiConvo.participants) {
    const otherParticipant = apiConvo.participants.find(p => p.id !== currentUserId);
    if (otherParticipant) {
      targetId = otherParticipant.id;
      // 优先使用nickname，如果没有则使用username
      name = name || otherParticipant.nickname || otherParticipant.username;
      username = otherParticipant.username; // 保存原始用户名
      avatar = avatar || otherParticipant.avatarUrl;
      isOnline = otherParticipant.isOnline || false;
    }
  } else if (apiConvo.type === 'private' && apiConvo.otherParticipant) { // Legacy structure
    // 优先使用nickname，如果没有则使用username
    name = name || apiConvo.otherParticipant.nickname || apiConvo.otherParticipant.username;
    username = apiConvo.otherParticipant.username; // 保存原始用户名
    avatar = avatar || apiConvo.otherParticipant.avatarUrl;
    targetId = targetId || apiConvo.otherParticipant.id;
    isOnline = apiConvo.otherParticipant.isOnline || false;
  } else if (apiConvo.type === 'group') {
    // 为群组会话设置目标ID和名称
    targetId = apiConvo.targetId; // 群组ID
    
    // 如果没有群组名称，使用默认名称
    if (!name && apiConvo.targetId) {
      name = `群组 ${apiConvo.targetId}`;
      console.log("[formatApiConversation] 使用默认群组名称:", name);
    } else {
      name = name || "群聊"; // 退化处理
      console.log("[formatApiConversation] 使用提供的群组名称:", name);
    }
  }

  // 名称的后备方案
  if (!name && targetId && apiConvo.type === 'private') {
    name = `用户 ${targetId}`;
  } else if (!name && isGroupConversation) {
    name = `群聊 ${apiConvo.id}`;
  }

  // 头像的后备方案
  if (!avatar && name) {
    const firstLetter = name.charAt(0).toUpperCase();
    avatar = `https://ui-avatars.com/api/?name=${encodeURIComponent(firstLetter)}&size=40&background=random&color=fff`;
  } else if (!avatar) {
    avatar = `https://i.pravatar.cc/40?u=${apiConvo.id}`;
  }

  // 群组类型的特殊处理
  const isGroup = apiConvo.type === 'group';
  
  // 调试输出
  if (isGroup) {
    console.log("[formatApiConversation] 处理群组会话:", {
      id: apiConvo.id,
      name: name,
      targetId: targetId,
      type: apiConvo.type
    });
  }

  return {
    id: apiConvo.id.toString(),
    name: name,
    username: username, // 添加用户名字段
    lastMessage: apiConvo.lastMessage?.content || apiConvo.lastMessage?.text || '',
    timestamp: apiConvo.lastMessage?.timestamp
      ? new Date(apiConvo.lastMessage.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: true })
      : (apiConvo.updatedAt ? new Date(apiConvo.updatedAt).toLocaleDateString() : ''),
    avatar: avatar,
    type: apiConvo.type,
    targetId: targetId,
    isOnline: isOnline,
    isGroup: isGroup, // 明确标记是否为群组
    unreadCount: apiConvo.unreadCount || 0,
    lastMessageTimestamp: apiConvo.lastMessage?.timestamp || apiConvo.updatedAt,
  };
};

export function useConversations() {
  const [conversations, setConversations] = useState([]);
  const [selectedConversation, setSelectedConversation] = useState(null);
  const [isLoadingConversations, setIsLoadingConversations] = useState(false);
  const { token, currentUser } = useAuth();

  // 获取会话列表
  const fetchConversations = useCallback(async () => {
    if (!token || !currentUser) return;
    setIsLoadingConversations(true);
    
    try {
      const result = await getConversations();
      if (!result.success) {
        console.error("获取会话列表失败:", result.error);
        setConversations([]);
        return;
      }
      
      const data = result.data || [];
      console.log("[useConversations]", "API 会话数据:", data);
      
      // 检查是否存在群组类型的会话
      const groupConversations = data.filter(c => c.type === 'group');
      console.log("[useConversations]", "群组会话数量:", groupConversations.length);
      if (groupConversations.length > 0) {
        console.log("[useConversations]", "群组会话数据:", groupConversations);
      }

      const formattedConversations = data.map(apiConvo => {
        const formatted = formatApiConversation(apiConvo, currentUser.id);
        if (apiConvo.type === 'group') {
          console.log("[useConversations]", "格式化群组会话:", apiConvo.id, "=>", formatted);
        }
        return formatted;
      });
      console.log("[useConversations]", "格式化后的会话数据:", formattedConversations);

      // 按最后消息时间排序
      setConversations(formattedConversations.sort((a, b) => {
        const timeA = a.lastMessageTimestamp ? new Date(a.lastMessageTimestamp).getTime() : 0;
        const timeB = b.lastMessageTimestamp ? new Date(b.lastMessageTimestamp).getTime() : 0;
        return timeB - timeA; // 最新的会话在前
      }));
    } catch (error) {
      console.error("获取会话列表出错:", error);
      setConversations([]);
    } finally {
      setIsLoadingConversations(false);
    }
  }, [currentUser, token]);

  // 初始化时加载会话列表
  useEffect(() => {
    if (currentUser && token) {
      fetchConversations();
    } else {
      setConversations([]);
      setSelectedConversation(null);
    }
  }, [currentUser, token, fetchConversations]);

  // 选择会话
  const selectConversation = useCallback((conversation) => {
    console.log('[useConversations] 选择会话:', conversation?.name, conversation?.id, conversation?.targetId);
    if (!conversation || !conversation.id) {
      console.warn('[useConversations] 无效的会话对象', conversation);
      setSelectedConversation(null);
      return;
    }
    setSelectedConversation(conversation);
  }, []);

  // 创建/更新会话列表中的会话
  const updateConversationWithMessage = useCallback((conversationId, message) => {
    if (!conversationId || !message) return;

    setConversations(prevConversations => {
      const convoIndex = prevConversations.findIndex(c => c.id === conversationId);
      if (convoIndex === -1) return prevConversations;

      const updatedConvo = {
        ...prevConversations[convoIndex],
        lastMessage: message.content,
        timestamp: new Date(message.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: true }),
        lastMessageTimestamp: message.timestamp,
      };
      
      const newConversations = [...prevConversations];
      newConversations.splice(convoIndex, 1);
      newConversations.unshift(updatedConvo);
      
      return newConversations.sort((a, b) => {
        const dateA = new Date(a.lastMessageTimestamp).getTime() || 0;
        const dateB = new Date(b.lastMessageTimestamp).getTime() || 0;
        return dateB - dateA;
      });
    });
  }, []);

  // 添加新会话
  const addConversation = useCallback((conversation) => {
    if (!conversation || !conversation.id) {
      console.warn('[useConversations] 无法添加无效的会话:', conversation);
      return;
    }

    setConversations(prevConversations => {
      // 检查会话是否已存在
      const existingIndex = prevConversations.findIndex(c => c.id === conversation.id);
      
      if (existingIndex !== -1) {
        // 如果会话已存在，则更新它
        const updatedConversations = [...prevConversations];
        updatedConversations[existingIndex] = {
          ...updatedConversations[existingIndex],
          ...conversation,
        };
        return updatedConversations;
      } else {
        // 如果是新会话，则添加到列表开头
        return [conversation, ...prevConversations];
      }
    });
  }, []);

  // 发起私聊
  const initiateChatWithContact = useCallback(async (contactId) => {
    if (!token || !currentUser) {
      console.warn('[useConversations] 无法发起会话: 未登录');
      return null;
    }

    console.log(`[useConversations] 尝试与联系人 ID: ${contactId} 开始会话`);
    
    try {
      const result = await initiatePrivateConversation(contactId);
      
      if (!result.success) {
        console.error(`[useConversations] 发起私聊失败 ${contactId}:`, result.error);
        return null;
      }
      
      const apiConversation = result.data;
      console.log('[useConversations] 新建/已有私聊会话响应:', apiConversation);
      
      if (!apiConversation) {
        console.error('[useConversations] API 成功但未返回会话数据');
        return null;
      }
      
      const formattedNewConversation = formatApiConversation(apiConversation, currentUser.id);
      console.log('[useConversations] 格式化后的私聊会话:', formattedNewConversation);
      
      // 更新会话列表
      setConversations(prevConversations => {
        const existingConvoIndex = prevConversations.findIndex(c => c.id === formattedNewConversation.id);
        let updatedConversations;
        
        if (existingConvoIndex > -1) {
          console.log('[useConversations] 会话已存在，更新');
          updatedConversations = [...prevConversations];
          updatedConversations[existingConvoIndex] = formattedNewConversation;
        } else {
          console.log('[useConversations] 新会话，添加到列表');
          updatedConversations = [formattedNewConversation, ...prevConversations];
        }
        
        return updatedConversations.sort((a, b) => {
          const dateA = new Date(a.lastMessageTimestamp).getTime() || 0;
          const dateB = new Date(b.lastMessageTimestamp).getTime() || 0;
          return dateB - dateA;
        });
      });
      
      // 选择并返回新会话
      selectConversation(formattedNewConversation);
      return formattedNewConversation;
      
    } catch (error) {
      console.error('[useConversations] 发起私聊时出错:', error);
      return null;
    }
  }, [currentUser, token, selectConversation]);

  return {
    conversations,
    selectedConversation,
    isLoadingConversations,
    selectConversation,
    updateConversationWithMessage,
    addConversation,
    initiateChatWithContact,
    refreshConversations: fetchConversations
  };
} 