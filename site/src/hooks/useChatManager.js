import { useCallback, useEffect, useRef } from 'react';
import useWebSocket from './useWebSocket';
import { useConversations } from './useConversations';
import { useMessages } from './useMessages';
import { useAuth } from '../contexts/AuthContext';
import { createGroup, fixGroupConversationParticipants } from '../services/api';

export function useChatManager() {
  const { currentUser } = useAuth();
  const { lastMessage, isConnected, sendMessage, websocketError } = useWebSocket();
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
    } else {
      console.warn('[useChatManager] 收到缺少 conversationId 的 WebSocket 消息:', lastMessage);
    }
  }, [lastMessage, addMessage, updateConversationWithMessage]);

  // 处理 WebSocket 错误
  useEffect(() => {
    if (websocketError) {
      console.error('[useChatManager] WebSocket 错误:', websocketError);
      // 这里可以添加错误处理逻辑，如通知、重试等
    }
  }, [websocketError]);

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
    
    // 消息管理
    clearMessages,
    clearAllMessages
  };
} 