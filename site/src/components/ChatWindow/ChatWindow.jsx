import React, { useEffect, useRef, useCallback } from 'react';
import MessageBubble from '../MessageBubble/MessageBubble';
import MessageInput from '../MessageInput/MessageInput';
import './ChatWindow.css';
import { useAuth } from '../../contexts/AuthContext';

function ChatWindow({ 
  conversation, 
  messages, 
  onSendMessage, 
  onLoadMoreMessages,
  isLoadingMoreMessages,
  hasMoreMessages
}) {
  const messagesEndRef = useRef(null);
  const messageListRef = useRef(null);
  const { currentUser } = useAuth();

  useEffect(() => {
    if (!isLoadingMoreMessages) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [messages, isLoadingMoreMessages, conversation]);

  const handleScroll = useCallback(() => {
    if (messageListRef.current) {
      const { scrollTop } = messageListRef.current;
      if (scrollTop === 0 && !isLoadingMoreMessages && hasMoreMessages) {
        console.log('[ChatWindow] Scrolled to top, loading more messages...');
        onLoadMoreMessages(conversation.id);
      }
    }
  }, [isLoadingMoreMessages, hasMoreMessages, onLoadMoreMessages, conversation?.id]);

  useEffect(() => {
    const listElement = messageListRef.current;
    if (listElement) {
      listElement.addEventListener('scroll', handleScroll);
      return () => listElement.removeEventListener('scroll', handleScroll);
    }
  }, [handleScroll]);

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
        <h2 className="chat-header-name">{conversation.name}</h2>
        {/* Placeholder for call buttons or other actions */}
      </header>
      <div className="message-list" ref={messageListRef}>
        {isLoadingMoreMessages && (
          <div className="loading-more-messages">Loading older messages...</div>
        )}
        {messages.map(msg => (
          <MessageBubble 
            key={msg.id} 
            message={msg} 
            currentUserId={currentUser?.id}
            isSender={String(msg.senderId) === String(currentUser?.id)}
          />
        ))}
        <div ref={messagesEndRef} />
      </div>
      <MessageInput onSendMessage={onSendMessage} />
    </div>
  );
}

export default ChatWindow; 