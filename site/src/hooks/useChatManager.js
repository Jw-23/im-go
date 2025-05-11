import { useCallback, useEffect, useRef } from 'react';
import useWebSocket from './useWebSocket';
import { useConversations } from './useConversations';
import { useMessages } from './useMessages';
import { useAuth } from '../contexts/AuthContext';
import { createGroup, fixGroupConversationParticipants } from '../services/api';

// 定时刷新配置（毫秒）
const AUTO_REFRESH_INTERVAL = 30000; // 30秒

export function useChatManager() {
  const { currentUser } = useAuth();
  const { lastMessage, isConnected, sendMessage, websocketError, reconnect } = useWebSocket();
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
        
        // 显示系统通知（如果浏览器支持）
        showNotificationForNewMessage(lastMessage);
      }
    } else {
      console.warn('[useChatManager] 收到缺少 conversationId 的 WebSocket 消息:', lastMessage);
    }
  }, [lastMessage, addMessage, updateConversationWithMessage, selectedConversation, refreshConversations]);

  // 显示新消息通知
  const showNotificationForNewMessage = useCallback((message) => {
    // 检查是否已授权通知权限
    if (!("Notification" in window)) {
      console.log('[useChatManager] 浏览器不支持系统通知');
      return;
    }
    
    // 如果通知权限未决定，请求权限
    if (Notification.permission === 'default') {
      Notification.requestPermission();
      return;
    }
    
    // 如果通知权限已授权，显示通知
    if (Notification.permission === 'granted') {
      // 找到对应的会话以获取发送者信息
      const conversation = conversations.find(c => c.id === message.conversationId);
      if (!conversation) return;
      
      // 确定通知标题
      let title;
      if (conversation.type === 'private') {
        title = `新消息来自: ${conversation.name || '私聊'}`;
      } else {
        title = `群组消息: ${conversation.name || '群聊'}`;
      }
      
      // 创建通知
      const notification = new Notification(title, {
        body: message.content.length > 50 ? message.content.substring(0, 50) + '...' : message.content,
        icon: conversation.avatar || '/logo.png', // 默认使用应用logo
        tag: message.conversationId, // 标记通知，相同会话只保留最新的通知
      });
      
      // 点击通知时，切换到对应会话
      notification.onclick = () => {
        window.focus(); // 聚焦到窗口
        if (conversation) {
          selectConversation(conversation);
        }
        notification.close();
      };
      
      // 3秒后自动关闭通知
      setTimeout(() => {
        notification.close();
      }, 3000);
    }
  }, [conversations, selectConversation]);

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

    return () => {
      if (autoRefreshTimerRef.current) {
        clearInterval(autoRefreshTimerRef.current);
        autoRefreshTimerRef.current = null;
      }
    };
  }, [currentUser, isConnected, refreshConversations, reconnect]);

  // 发送消息
  const handleSendMessage = useCallback((content) => {
    if (!content || !selectedConversation || !isConnected || !currentUser) {
      console.warn('[useChatManager] 无法发送消息。条件不满足:', { content, selectedConversation, isConnected, currentUser });
      return;
    }
    
    const timestamp = new Date().toISOString();
    
    // 根据会话类型设置消息payload
    const messagePayload = {
      type: 'text',
      content,
      timestamp,
      conversationId: selectedConversation.id, // 始终包含会话ID
    };
    
    // 确保receiverId设置正确:
    // 对于私聊：receiverId = 对方用户ID
    // 对于群聊：receiverId = 会话ID
    if (selectedConversation.type === 'private') {
      // 私聊：receiverId是目标用户ID
      messagePayload.receiverId = selectedConversation.targetId.toString();
      console.log('[useChatManager] 准备发送私聊消息:', messagePayload);
    } else {
      // 群聊：receiverId使用会话ID (与conversationId一致，服务器会处理)
      messagePayload.receiverId = selectedConversation.id;
      console.log('[useChatManager] 准备发送群聊消息:', messagePayload);
    }

    console.log('[useChatManager] 发送 WebSocket 消息:', messagePayload);
    sendMessage(messagePayload);

    // 创建并添加乐观消息
    createOptimisticMessage(content, selectedConversation.id);
    
    // 更新会话列表中的会话项目
    updateConversationWithMessage(selectedConversation.id, {
      content,
      timestamp
    });
  }, [selectedConversation, isConnected, currentUser, sendMessage, createOptimisticMessage, updateConversationWithMessage]);

  // 加载更多消息
  const handleLoadMoreMessages = useCallback((conversationId) => {
    if (!conversationId || isLoadingMessages[conversationId] || !hasMoreMessages[conversationId]) {
      console.log(`[useChatManager] 无法加载更多消息: ${conversationId}. 加载中: ${isLoadingMessages[conversationId]}, 有更多消息: ${hasMoreMessages[conversationId]}`);
      return;
    }
    
    const currentOffset = messageOffsets[conversationId] || 0;
    fetchMessages(conversationId, currentOffset, 20, false);
  }, [isLoadingMessages, hasMoreMessages, messageOffsets, fetchMessages]);

  // 初始化和选择会话（结合了选择和消息加载）
  const handleSelectConversation = useCallback((conversation) => {
    console.log('[useChatManager] 选择会话:', conversation?.name, conversation?.id, conversation?.type);
    if (!conversation || !conversation.id) {
      console.warn('[useChatManager] 无效的会话对象', conversation);
      return;
    }

    // 额外检查群组会话
    if (conversation.type === 'group' || conversation.isGroup) {
      console.log('[useChatManager] 选择群组会话:', conversation);
    }

    selectConversation(conversation);
  }, [selectConversation]);

  // 创建群组
  const handleCreateGroup = useCallback(async (groupData) => {
    if (!currentUser) {
      console.warn('[useChatManager] 用户未登录，无法创建群组');
      return { success: false, error: '用户未登录' };
    }

    try {
      // 确保memberIds是数组并且包含有效的数字
      console.log('[useChatManager] 原始memberIds:', groupData.memberIds, 
                 '类型:', Array.isArray(groupData.memberIds) ? 'array' : typeof groupData.memberIds);
      
      // 确保所有ID都是数字类型且大于0
      const memberIds = Array.isArray(groupData.memberIds) 
        ? groupData.memberIds
            .map(id => typeof id === 'number' ? id : Number(id))
            .filter(id => !isNaN(id) && id > 0)
        : [];
        
      console.log('[useChatManager] 处理后的memberIds:', memberIds,
                 '元素类型:', memberIds.map(id => typeof id));
      
      console.log('[useChatManager] 创建群组:', {
        ...groupData,
        memberIds: memberIds
      });
      
      // 准备群组数据
      const groupRequest = {
        name: groupData.name,
        memberIds: memberIds,
        description: groupData.description || '',
        avatarUrl: groupData.avatarUrl || '',
        isPublic: groupData.isPublic || false,
      };
      
      // 调用API创建群组
      console.log('[useChatManager] 发送创建群组请求:', JSON.stringify(groupRequest));
      const result = await createGroup(groupRequest);
      
      if (!result.success) {
        console.error('[useChatManager] 创建群组失败:', result.error);
        return result;
      }
      
      const newGroup = result.data;
      console.log('[useChatManager] 群组创建成功:', newGroup);
      
      // 刷新会话列表以获取新的群组会话
      console.log('[useChatManager] 刷新会话列表...');
      await refreshConversations();
      
      // 查找新创建的群组会话
      const newGroupConvo = conversations.find(c => 
        c.type === 'group' && c.targetId === newGroup.id
      );
      
      console.log('[useChatManager] 新群组会话查找结果:', newGroupConvo);
      
      if (newGroupConvo) {
        // 如果找到了对应的会话，选择它
        console.log('[useChatManager] 选择新创建的群组会话:', newGroupConvo.id);
        selectConversation(newGroupConvo);
        
        // 修复参与者错误问题 - 在创建群组成功后，确保所有参与者被正确添加
        try {
          await fixGroupConversationParticipants(newGroup.id);
          console.log('[useChatManager] 修复群组会话参与者成功');
        } catch (fixError) {
          console.warn('[useChatManager] 修复群组会话参与者失败:', fixError);
        }
        
        return { success: true, data: newGroup };
      } else {
        // 如果没有找到对应的会话，创建一个临时的
        console.log('[useChatManager] 未找到群组会话，创建临时会话');
        const tempGroupConvo = {
          id: `temp_${Date.now()}`, // 临时ID，避免与真实会话ID冲突
          type: 'group',
          name: newGroup.name,
          avatar: newGroup.avatarUrl,
          targetId: newGroup.id,
          lastMessage: null,
          memberCount: memberIds.length + 1, // 创建者 + 被邀请的成员
          updatedAt: new Date().toISOString(),
          unreadCount: 0
        };
        
        // 添加新会话到会话列表
        console.log('[useChatManager] 添加临时群组会话:', tempGroupConvo);
        addConversation(tempGroupConvo);
        
        // 选择新创建的会话
        selectConversation(tempGroupConvo);
        
        // 再次刷新会话列表获取真实会话
        setTimeout(async () => {
          try {
            await refreshConversations();
            // 再次尝试找到真实会话
            const realConversation = conversations.find(c => 
              c.type === 'group' && c.targetId === newGroup.id
            );
            if (realConversation) {
              selectConversation(realConversation);
            }
            
            // 修复参与者错误问题
            await fixGroupConversationParticipants(newGroup.id);
          } catch (err) {
            console.error('[useChatManager] 延迟刷新会话失败:', err);
          }
        }, 1000);
        
        return { success: true, data: newGroup };
      }
    } catch (error) {
      console.error('[useChatManager] 创建群组过程中发生错误:', error);
      return { success: false, error: error.message || '创建群组失败' };
    }
  }, [currentUser, addConversation, selectConversation, refreshConversations, conversations, fixGroupConversationParticipants]);

  return {
    // 状态
    conversations,
    selectedConversation,
    messages: selectedConversation ? messages[selectedConversation.id] || [] : [],
    isLoadingConversations,
    isLoadingMessages: selectedConversation ? isLoadingMessages[selectedConversation.id] || false : false,
    hasMoreMessages: selectedConversation ? hasMoreMessages[selectedConversation.id] !== undefined ? hasMoreMessages[selectedConversation.id] : true : false,
    websocketConnected: isConnected,
    websocketError,
    
    // 动作
    sendMessage: handleSendMessage,
    loadMoreMessages: handleLoadMoreMessages,
    selectConversation: handleSelectConversation,
    initiateChatWithContact,
    refreshConversations,
    createGroup: handleCreateGroup,
    reconnectWebSocket: reconnect, // 暴露WebSocket重连函数
    
    // 消息管理
    clearMessages,
    clearAllMessages
  };
} 