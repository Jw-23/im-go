import { useCallback, useEffect, useRef } from 'react';
import useWebSocket from './useWebSocket';
import { useConversations } from './useConversations';
import { useMessages } from './useMessages';
import { useAuth } from '../contexts/AuthContext';
import useNotification from './useNotification';
import { createGroup, fixGroupConversationParticipants } from '../services/api';

// 定时刷新配置（毫秒）
const AUTO_REFRESH_INTERVAL = 30000; // 30秒

export function useChatManager() {
  const { currentUser } = useAuth();
  const { showNotification } = useNotification();
  const { lastMessage, isConnected, sendMessage, websocketError, reconnect } = useWebSocket(showNotification);
  const { 
    conversations, 
    selectedConversation, 
    isLoadingConversations, 
    selectConversation, 
    updateConversationWithMessage, 
    initiateChatWithContact,
    refreshConversations,
    addConversation
  } = useConversations();
  
  const { 
    messages, 
    isLoadingMessages, 
    hasMoreMessages, 
    messageOffsets, 
    fetchMessages, 
    addMessage, 
    createOptimisticMessage, 
    clearMessages,
    clearAllMessages
  } = useMessages();

  // 防止重复加载
  const loadingConversationRef = useRef(null);
  const initialLoadCompletedRef = useRef({});

  // 添加定时刷新逻辑
  const autoRefreshTimerRef = useRef(null);
  const lastRefreshTimeRef = useRef(Date.now());

  // 当选择会话改变时加载消息
  useEffect(() => {
    if (!selectedConversation?.id) return;
    
    // 避免重复加载同一个会话
    if (loadingConversationRef.current === selectedConversation.id) {
      return;
    }
    
    // 避免已经加载过的会话再次加载
    if (messages[selectedConversation.id]?.length > 0 && initialLoadCompletedRef.current[selectedConversation.id]) {
      console.log(`[useChatManager] 会话 ${selectedConversation.id} 已经加载过消息，跳过初始加载`);
      return;
    }
    
    if (!isLoadingMessages[selectedConversation.id]) {
      console.log(`[useChatManager] 选择会话改变, 加载会话 ${selectedConversation.id} 的初始消息`);
      loadingConversationRef.current = selectedConversation.id;
      
      // 设置适当的延迟，避免在会话切换期间立即触发请求
      setTimeout(() => {
        if (selectedConversation?.id === loadingConversationRef.current) {
          fetchMessages(selectedConversation.id, 0, 20, true);
          initialLoadCompletedRef.current[selectedConversation.id] = true;
        }
        loadingConversationRef.current = null;
      }, 300);
    }
  }, [selectedConversation?.id, fetchMessages, isLoadingMessages, messages]);

  // 处理接收到的 WebSocket 消息
  useEffect(() => {
    if (!lastMessage) return;
    
    if (lastMessage.conversationId) {
      console.log('[useChatManager] 处理 WebSocket 消息:', lastMessage);
      
      // 将消息添加到消息列表
      addMessage(lastMessage);
      
      // 更新会话的最后一条消息
      updateConversationWithMessage(lastMessage.conversationId, lastMessage);
      
      // 检查是否需要自动刷新会话列表
      // 如果是不在当前选中会话的新消息，将触发自动刷新
      if (!selectedConversation || lastMessage.conversationId !== selectedConversation.id) {
        // 自动刷新会话列表
        console.log('[useChatManager] 收到新消息，自动刷新会话列表');
        refreshConversations();
      }
    } else {
      console.warn('[useChatManager] 收到缺少 conversationId 的 WebSocket 消息:', lastMessage);
    }
  }, [lastMessage, addMessage, updateConversationWithMessage, selectedConversation, refreshConversations]);

  // 处理 WebSocket 错误
  useEffect(() => {
    if (websocketError) {
      console.error('[useChatManager] WebSocket 错误:', websocketError);
      // 这里可以添加错误处理逻辑，如通知、重试等
      // 如果websocket断开连接，尝试重新连接
      if (!isConnected) {
        console.log('[useChatManager] WebSocket断开连接，尝试重新连接...');
        // 可以在5秒后尝试刷新会话列表
        const reconnectTimer = setTimeout(() => {
          refreshConversations();
        }, 5000);
        
        return () => {
          clearTimeout(reconnectTimer);
        };
      }
    }
  }, [websocketError, isConnected, refreshConversations]);

  // 设置定时刷新
  useEffect(() => {
    if (!currentUser) return;

    // 清除旧的定时器
    if (autoRefreshTimerRef.current) {
      clearInterval(autoRefreshTimerRef.current);
    }

    // 创建新的定时刷新
    autoRefreshTimerRef.current = setInterval(() => {
      // 如果WebSocket连接断开或者距离上次刷新超过30秒，则刷新会话列表
      const now = Date.now();
      if (!isConnected || now - lastRefreshTimeRef.current >= AUTO_REFRESH_INTERVAL) {
        console.log('[useChatManager] 定时刷新会话列表', 
                    isConnected ? '（定时刷新）' : '（WebSocket断开，自动刷新）');
        refreshConversations().then(() => {
          lastRefreshTimeRef.current = Date.now();
        });

        // 如果WebSocket断开，尝试重连
        if (!isConnected && reconnect) {
          console.log('[useChatManager] 检测到WebSocket断开，尝试重连');
          reconnect();
        }
      }
    }, AUTO_REFRESH_INTERVAL);

    // 清理
    return () => {
      if (autoRefreshTimerRef.current) {
        clearInterval(autoRefreshTimerRef.current);
      }
    };
  }, [currentUser, isConnected, refreshConversations, reconnect]);

  // 清理所有数据
  useEffect(() => {
    if (!currentUser) {
      // 用户登出时，清理状态
      clearAllMessages();
      loadingConversationRef.current = null;
      initialLoadCompletedRef.current = {};
    }
  }, [currentUser, clearAllMessages]);

  // 发送消息
  const handleSendMessage = useCallback(async (content, conversationId, messageType = 'text') => {
    if (!currentUser || !content || !conversationId) {
      console.error('[useChatManager] 发送消息缺少必要参数:', { 
        hasCurrentUser: !!currentUser, 
        content, 
        conversationId 
      });
      return { success: false, error: '发送消息缺少必要参数' };
    }

    // 创建乐观消息
    const optimisticMessage = createOptimisticMessage(
      conversationId,
      content,
      messageType,
      currentUser.id
    );

    // 添加乐观消息到消息列表
    addMessage(optimisticMessage);

    // 通过WebSocket发送消息
    try {
      const messagePayload = {
        type: 'message',
        conversation_id: conversationId,
        content: content,
        message_type: messageType
      };

      console.log('[useChatManager] 发送WebSocket消息:', messagePayload);
      
      // 发送消息并获取结果
      const sendSuccessful = sendMessage(messagePayload);
      
      // 如果消息实际发送成功或WebSocket已连接，则认为操作成功
      // 这是因为我们已经采用了乐观更新的方式，即使消息暂时未发送成功，也先在UI上显示
      if (sendSuccessful || isConnected) {
        // 更新会话的最后一条消息（使用乐观消息）
        updateConversationWithMessage(conversationId, optimisticMessage);
        return { success: true, message: optimisticMessage, isSent: sendSuccessful };
      } else {
        // WebSocket断开且发送失败
        console.warn('[useChatManager] 消息未能发送 - WebSocket未连接:', optimisticMessage.id);
        
        // 更新消息状态为"发送失败"
        setMessages(prev => {
          const conversationMessages = prev[conversationId] || [];
          return {
            ...prev,
            [conversationId]: conversationMessages.map(msg => 
              msg.id === optimisticMessage.id 
                ? { ...msg, status: 'failed' } 
                : msg
            )
          };
        });
        
        return { 
          success: false, 
          message: optimisticMessage, 
          error: '消息发送失败：连接已断开。请稍后重试', 
          isSent: false 
        };
      }
    } catch (error) {
      console.error('[useChatManager] 发送消息失败:', error);
      return { success: false, error: error.message || '发送消息失败' };
    }
  }, [currentUser, createOptimisticMessage, addMessage, sendMessage, updateConversationWithMessage, isConnected, setMessages]);

  // 加载更多消息
  const loadMoreMessages = useCallback(async (conversationId) => {
    if (!conversationId) return;

    const offset = messageOffsets[conversationId] || 0;
    await fetchMessages(conversationId, offset, 20, false);
  }, [fetchMessages, messageOffsets]);

  // 处理创建群组
  const handleCreateGroup = useCallback(async (groupData) => {
    if (!currentUser) {
      return { success: false, error: '创建群组前请先登录' };
    }

    try {
      const result = await createGroup({
        name: groupData.name,
        member_ids: groupData.member_ids
      });

      if (result.success && result.data) {
        // 刷新会话列表
        await refreshConversations();
        
        return { success: true, data: result.data };
      } else {
        return { success: false, error: result.error || '创建群组失败' };
      }
    } catch (error) {
      console.error('[useChatManager] 创建群组过程中发生错误:', error);
      return { success: false, error: error.message || '创建群组过程中发生未知错误' };
    }
  }, [currentUser, refreshConversations]);

  // 处理修复群组会话
  const handleFixGroupConversation = useCallback(async (groupId) => {
    try {
      console.log('[useChatManager] 修复群组会话:', groupId);
      const result = await fixGroupConversationParticipants(groupId);
      
      if (result.success) {
        // 刷新会话列表
        await refreshConversations();
        return { success: true, data: result.data };
      } else {
        return { success: false, error: result.error || '修复群组会话失败' };
      }
    } catch (error) {
      console.error('[useChatManager] 修复群组会话过程中发生错误:', error);
      return { success: false, error: error.message || '修复群组会话过程中发生未知错误' };
    }
  }, [refreshConversations]);

  return {
    // 状态
    conversations,
    selectedConversation,
    messages: selectedConversation ? messages[selectedConversation.id] || [] : [],
    isLoadingConversations,
    isLoadingMessages: selectedConversation ? isLoadingMessages[selectedConversation.id] || false : false,
    hasMoreMessages: selectedConversation ? hasMoreMessages[selectedConversation.id] !== undefined ? hasMoreMessages[selectedConversation.id] : true : false,
    websocketConnected: isConnected,
    
    // 动作
    sendMessage: handleSendMessage,
    loadMoreMessages,
    selectConversation,
    initiateChatWithContact,
    createGroup: handleCreateGroup,
    fixGroupConversation: handleFixGroupConversation,
    
    // WebSocket相关
    reconnectWebSocket: reconnect
  };
} 