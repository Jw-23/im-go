import { useState, useCallback, useEffect, useRef } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { getConversationMessages } from '../services/api';

const DEFAULT_MESSAGE_LIMIT = 20;

export function useMessages() {
  const [messages, setMessages] = useState({});
  const [messageOffsets, setMessageOffsets] = useState({});
  const [hasMoreMessages, setHasMoreMessages] = useState({});
  const [isLoadingMessages, setIsLoadingMessages] = useState({});
  const { currentUser } = useAuth();
  
  // 用于防止重复请求
  const pendingRequests = useRef({});
  const lastFetchTimestamps = useRef({});

  // 清除特定会话的消息
  const clearMessages = useCallback((conversationId) => {
    setMessages(prev => {
      const newMessages = { ...prev };
      delete newMessages[conversationId];
      return newMessages;
    });
    setMessageOffsets(prev => {
      const newOffsets = { ...prev };
      delete newOffsets[conversationId];
      return newOffsets;
    });
    setHasMoreMessages(prev => {
      const newHasMore = { ...prev };
      delete newHasMore[conversationId];
      return newHasMore;
    });
    setIsLoadingMessages(prev => {
      const newLoading = { ...prev };
      delete newLoading[conversationId];
      return newLoading;
    });
    
    // 清除相关缓存
    delete pendingRequests.current[conversationId];
    delete lastFetchTimestamps.current[conversationId];
  }, []);

  // 清除所有会话的消息
  const clearAllMessages = useCallback(() => {
    setMessages({});
    setMessageOffsets({});
    setHasMoreMessages({});
    setIsLoadingMessages({});
    
    // 清除所有请求缓存
    pendingRequests.current = {};
    lastFetchTimestamps.current = {};
  }, []);

  // 加载会话消息
  const fetchMessages = useCallback(async (conversationId, offset = 0, limit = DEFAULT_MESSAGE_LIMIT, isInitialFetch = true) => {
    if (!conversationId) return;
    
    // 防止重复请求: 检查是否有相同参数的请求正在进行中
    const requestKey = `${conversationId}-${offset}-${limit}`;
    if (pendingRequests.current[requestKey]) {
      console.log(`[useMessages] 跳过重复请求: ${requestKey}`);
      return;
    }
    
    // 防抖: 检查最近是否已请求过相同offset的数据 (2秒内)
    const now = Date.now();
    const lastFetch = lastFetchTimestamps.current[requestKey] || 0;
    if (now - lastFetch < 2000) {
      console.log(`[useMessages] 请求过于频繁，跳过: ${requestKey}`);
      return;
    }
    
    console.log(`[useMessages] 获取会话 ${conversationId} 的消息, 偏移量: ${offset}, 限制数: ${limit}, 是否初始加载: ${isInitialFetch}`);

    setIsLoadingMessages(prev => ({ ...prev, [conversationId]: true }));
    
    // 标记请求进行中
    pendingRequests.current[requestKey] = true;
    lastFetchTimestamps.current[requestKey] = now;

    try {
      const result = await getConversationMessages(conversationId, limit, offset);

      if (!result.success) {
        console.error(`获取会话 ${conversationId} 的消息失败, 状态: ${result.status}, 错误: ${result.error}`);
        setHasMoreMessages(prev => ({ ...prev, [conversationId]: false }));
        return;
      }

      const fetchedMessages = result.data || [];
      if (fetchedMessages.length === 0) {
        console.log(`[useMessages] 会话 ${conversationId} 没有更多消息`);
        setHasMoreMessages(prev => ({ ...prev, [conversationId]: false }));
        return;
      }
      
      console.log(`[useMessages] 获取到 ${conversationId} 的 ${fetchedMessages.length} 条消息`);

      // 反转消息顺序，按时间顺序排列（从旧到新）
      let chronologicallyFetchedMessages = [...fetchedMessages].reverse();
      
      // 映射 sentAt 到 timestamp 字段以保持一致性
      chronologicallyFetchedMessages = chronologicallyFetchedMessages.map(msg => ({
        ...msg,
        timestamp: msg.sentAt || msg.timestamp, // 确保 timestamp 字段存在
      }));

      setMessages(prevMessages => {
        const currentMessages = prevMessages[conversationId] || [];
        
        // 检查消息是否已存在
        const messageIds = new Set(currentMessages.map(msg => msg.id));
        const uniqueNewMessages = chronologicallyFetchedMessages.filter(
          msg => !messageIds.has(msg.id)
        );
        
        if (uniqueNewMessages.length === 0) {
          console.log(`[useMessages] 所有消息已存在，不更新消息列表`);
          return prevMessages;
        }
        
        // 当加载更多较旧的消息时（isInitialFetch = false），将新的（较旧的）消息前置
        // 当是初始加载时，仅设置消息
        const newMessages = isInitialFetch
          ? uniqueNewMessages
          : [...uniqueNewMessages, ...currentMessages];

        console.log(`[useMessages] 更新 ${conversationId} 的消息状态. 初始: ${isInitialFetch}, 新增消息数: ${uniqueNewMessages.length}`);
        return {
          ...prevMessages,
          [conversationId]: newMessages,
        };
      });

      // 只有在实际有新消息时才更新偏移量
      if (fetchedMessages.length > 0) {
        setMessageOffsets(prevOffsets => ({ ...prevOffsets, [conversationId]: offset + fetchedMessages.length }));
      }
      
      setHasMoreMessages(prev => ({ ...prev, [conversationId]: fetchedMessages.length === limit }));

    } catch (error) {
      console.error(`获取会话 ${conversationId} 的消息时发生严重错误:`, error);
      setHasMoreMessages(prev => ({ ...prev, [conversationId]: false }));
    } finally {
      setIsLoadingMessages(prev => ({ ...prev, [conversationId]: false }));
      // 请求完成，移除标记
      delete pendingRequests.current[requestKey];
    }
  }, []);

  // 添加新消息（通过 WebSocket 接收或本地发送）
  const addMessage = useCallback((message) => {
    if (!message || !message.conversationId) {
      console.warn('[useMessages] 无法添加消息: 缺少会话ID', message);
      return;
    }

    setMessages(prevMessages => {
      const conversationId = message.conversationId;
      const existingMessages = prevMessages[conversationId] || [];
      
      // 检查消息是否已存在，避免重复
      const messageExists = existingMessages.some(msg => 
        (msg.id && msg.id === message.id) || 
        (msg.timestamp === message.timestamp && msg.senderId === message.senderId && msg.content === message.content)
      );
      
      if (messageExists) {
        console.log('[useMessages] 消息已存在，跳过添加', message);
        return prevMessages;
      }

      return {
        ...prevMessages,
        [conversationId]: [...existingMessages, message]
      };
    });
  }, []);

  // 创建一个新的乐观消息并添加到状态中
  const createOptimisticMessage = useCallback((content, conversationId) => {
    if (!content || !conversationId || !currentUser) {
      console.warn('[useMessages] 无法创建乐观消息: 参数不足', { content, conversationId, currentUser });
      return null;
    }

    const timestamp = new Date().toISOString();
    const optimisticMessage = {
      id: `msg_${Date.now()}`,
      type: 'text',
      content,
      senderId: currentUser.id.toString(),
      conversationId,
      timestamp,
      status: 'sending'
    };

    addMessage(optimisticMessage);
    return optimisticMessage;
  }, [addMessage, currentUser]);

  return {
    messages,
    isLoadingMessages,
    hasMoreMessages,
    messageOffsets,
    fetchMessages,
    addMessage,
    createOptimisticMessage,
    clearMessages,
    clearAllMessages
  };
} 