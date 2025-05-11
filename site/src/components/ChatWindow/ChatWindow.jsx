import React, { useEffect, useRef, useCallback, useState } from 'react';
import MessageBubble from '../MessageBubble/MessageBubble';
import MessageInput from '../MessageInput/MessageInput';
import './ChatWindow.css';
import { useAuth } from '../../contexts/AuthContext';

// 图标组件
const UsersIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16">
    <path d="M15 14s1 0 1-1-1-4-5-4-5 3-5 4 1 1 1 1h8zm-7.978-1A.261.261 0 0 1 7 12.996c.001-.264.167-1.03.76-1.72C8.312 10.629 9.282 10 11 10c1.717 0 2.687.63 3.24 1.276.593.69.758 1.457.76 1.72l-.008.002a.274.274 0 0 1-.014.002H7.022zM11 7a2 2 0 1 0 0-4 2 2 0 0 0 0 4zm3-2a3 3 0 1 1-6 0 3 3 0 0 1 6 0zM6.936 9.28a5.88 5.88 0 0 0-1.23-.247A7.35 7.35 0 0 0 5 9c-4 0-5 3-5 4 0 .667.333 1 1 1h4.216A2.238 2.238 0 0 1 5 13c0-1.01.377-2.042 1.09-2.904.243-.294.526-.569.846-.816zM4.92 10A5.493 5.493 0 0 0 4 13H1c0-.26.164-1.03.76-1.724.545-.636 1.492-1.256 3.16-1.275zM1.5 5.5a3 3 0 1 1 6 0 3 3 0 0 1-6 0zm3-2a2 2 0 1 0 0 4 2 2 0 0 0 0-4z"/>
  </svg>
);

const ImageIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16">
    <path d="M6.002 5.5a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0z"/>
    <path d="M2.002 1a2 2 0 0 0-2 2v10a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V3a2 2 0 0 0-2-2h-12zm12 1a1 1 0 0 1 1 1v6.5l-3.777-1.947a.5.5 0 0 0-.577.093l-3.71 3.71-2.66-1.772a.5.5 0 0 0-.63.062L1.002 12V3a1 1 0 0 1 1-1h12z"/>
  </svg>
);

const FileIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16">
    <path d="M14 4.5V14a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V2a2 2 0 0 1 2-2h5.5L14 4.5zm-3 0A1.5 1.5 0 0 1 9.5 3V1H4a1 1 0 0 0-1 1v12a1 1 0 0 0 1 1h8a1 1 0 0 0 1-1V4.5h-2z"/>
  </svg>
);

const MentionIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16">
    <path d="M13.106 7.222c0-2.967-2.249-5.032-5.482-5.032-3.35 0-5.646 2.318-5.646 5.702 0 3.493 2.235 5.708 5.762 5.708.862 0 1.689-.123 2.304-.335v-.862c-.43.199-1.354.328-2.29.328-2.926 0-4.813-1.88-4.813-4.798 0-2.844 1.921-4.881 4.594-4.881 2.735 0 4.608 1.688 4.608 4.156 0 1.682-.554 2.769-1.416 2.769-.492 0-.772-.28-.772-.76V5.206H8.923v.834h-.11c-.266-.595-.881-.964-1.6-.964-1.4 0-2.378 1.162-2.378 2.823 0 1.737.957 2.906 2.379 2.906.8 0 1.415-.39 1.709-1.087h.11c.081.67.703 1.148 1.503 1.148 1.572 0 2.57-1.415 2.57-3.643zm-7.177.704c0-1.197.54-1.907 1.456-1.907.93 0 1.524.738 1.524 1.907S8.308 9.84 7.371 9.84c-.895 0-1.442-.725-1.442-1.914z"/>
  </svg>
);

function ChatWindow({ 
  conversation, 
  messages, 
  onSendMessage, 
  onLoadMoreMessages,
  isLoadingMoreMessages,
  hasMoreMessages,
  websocketConnected = false,
  onManageGroupMembers,
  isGroup = false
}) {
  const messagesEndRef = useRef(null);
  const messageListRef = useRef(null);
  const { currentUser } = useAuth();
  const [shouldScrollToBottom, setShouldScrollToBottom] = useState(true);
  const [prevMessagesLength, setPrevMessagesLength] = useState(0);
  const [prevConversationId, setPrevConversationId] = useState(null);
  const [isInitialLoad, setIsInitialLoad] = useState(true);

  // 跟踪对话ID变化
  useEffect(() => {
    if (conversation?.id !== prevConversationId) {
      setPrevConversationId(conversation?.id);
      setIsInitialLoad(true);
    }
  }, [conversation?.id, prevConversationId]);

  // 改进的滚动逻辑
  useEffect(() => {
    // 改进防抖函数实现平滑滚动
    let scrollTimeout;
    
    if (!isLoadingMoreMessages && shouldScrollToBottom && messageListRef.current && messagesEndRef.current) {
      // 优化：只有在消息数增加或对话变化时才滚动
      if (messages.length > prevMessagesLength || conversation?.id !== prevConversationId) {
        // 清除之前的定时器
        clearTimeout(scrollTimeout);
        
        // 延迟执行滚动，给浏览器时间完成DOM更新
        scrollTimeout = setTimeout(() => {
          try {
            // 直接设置滚动位置而不是使用scrollIntoView
            if (messageListRef.current) {
              messageListRef.current.scrollTop = messageListRef.current.scrollHeight;
            }
          } catch (error) {
            console.error("滚动错误:", error);
          }
        }, 100);
        
        setPrevMessagesLength(messages.length);
      }
    }
    
    return () => clearTimeout(scrollTimeout);
  }, [messages.length, isLoadingMoreMessages, conversation?.id, shouldScrollToBottom, prevMessagesLength, prevConversationId]);

  // 确保初始加载和新消息时总是滚动到底部（使用不同的方法避免冲突）
  useEffect(() => {
    let initialScrollTimeout;
    
    if (messageListRef.current && messages.length > 0 && isInitialLoad && conversation?.id) {
      initialScrollTimeout = setTimeout(() => {
        if (messageListRef.current) {
          messageListRef.current.scrollTop = messageListRef.current.scrollHeight;
          setIsInitialLoad(false);
        }
      }, 300); // 稍长的延迟确保渲染完成
    }
    
    return () => clearTimeout(initialScrollTimeout);
  }, [conversation?.id, messages.length, isInitialLoad]);

  // 监听滚动事件，判断用户是否手动滚动
  const handleScroll = useCallback(() => {
    if (!messageListRef.current) return;
    
    const { scrollTop, scrollHeight, clientHeight } = messageListRef.current;
    
    // 检测用户是否已滚动到底部附近（距离底部20px以内）
    const isNearBottom = scrollHeight - scrollTop - clientHeight < 20;
    
    // 更新是否应该自动滚动到底部
    setShouldScrollToBottom(isNearBottom);
    
    // 只有当滚动到顶部且有更多消息且当前没有正在加载消息时才加载更多
    if (scrollTop === 0 && hasMoreMessages && !isLoadingMoreMessages && conversation?.id) {
      console.log('[ChatWindow] 滚动到顶部，加载更多消息...');
        onLoadMoreMessages(conversation.id);
    }
  }, [isLoadingMoreMessages, hasMoreMessages, onLoadMoreMessages, conversation?.id]);

  // 改进滚动事件监听器，添加节流控制
  useEffect(() => {
    const listElement = messageListRef.current;
    if (!listElement) return;
    
    let isScrolling = false;
    const throttledScroll = () => {
      if (isScrolling) return;
      isScrolling = true;
      
      window.requestAnimationFrame(() => {
        handleScroll();
        isScrolling = false;
      });
    };
    
    listElement.addEventListener('scroll', throttledScroll);
    return () => listElement.removeEventListener('scroll', throttledScroll);
  }, [handleScroll]);

  const handleManageGroupMembers = () => {
    if (onManageGroupMembers) {
      onManageGroupMembers(conversation.id);
    }
  };

  if (!conversation) {
    return <div className="chat-window-empty">Select a conversation</div>;
  }

  return (
    <div className="chat-window">
      <header className="chat-window-header">
        <img 
          src={conversation.avatar || 'https://i.pravatar.cc/30?u=default'} 
          alt={`${conversation.name} avatar`} 
          className="chat-header-avatar"
        />
        <div className="chat-header-info">
          <h2 
            className="chat-header-name" 
            title={conversation.type === 'private' ? `@${conversation.username || ''}` : ''}
          >
            {conversation.name}
          </h2>
          {isGroup && (
            <span className="chat-header-members">{conversation.memberCount || '未知'} 成员</span>
          )}
        </div>
        <div className="chat-header-actions">
          <div className={`connection-status ${websocketConnected ? 'connected' : 'disconnected'}`}>
            {websocketConnected ? '已连接' : '未连接'}
          </div>
          {/* {isGroup && (
            <button onClick={handleManageGroupMembers} className="manage-group-btn">
              <UsersIcon /> 管理成员
            </button>
          )} */}
        </div>
      </header>
      
      <div className="message-list" ref={messageListRef}>
        {isLoadingMoreMessages && (
          <div className="loading-more-messages">加载更多消息...</div>
        )}
        {messages.map(msg => (
          <MessageBubble 
            key={msg.id} 
            message={msg} 
            currentUserId={currentUser?.id}
            isSender={String(msg.senderId) === String(currentUser?.id)}
          />
        ))}
        <div ref={messagesEndRef} className="message-end-anchor" />
      </div>
      
      <div className="message-input-container">
        {!websocketConnected && (
          <div className="connection-warning">
            消息服务已断开连接，无法发送消息
          </div>
        )}
        <div className="message-input-toolbar inner-container">
          <button className="toolbar-button" title="发送图片">
            <ImageIcon />
          </button>
          <button className="toolbar-button" title="发送文件">
            <FileIcon />
          </button>
          {isGroup && (
            <button className="toolbar-button" title="@提及成员">
              <MentionIcon />
            </button>
          )}
        </div>
        <MessageInput 
          onSendMessage={(content) => onSendMessage(content, conversation.id)} 
          disabled={!websocketConnected} 
          conversationType={isGroup ? 'group' : 'private'}
          conversationId={conversation.id}
        />
      </div>
    </div>
  );
}

export default ChatWindow; 