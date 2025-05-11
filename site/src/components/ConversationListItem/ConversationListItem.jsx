import React from 'react';
import './ConversationListItem.css';

function ConversationListItem({ conversation, onSelect, isSelected }) {
  // 判断是否为群组会话
  const isGroup = conversation.type === 'group' || conversation.isGroup;
  
  // 获取显示名称和悬停提示
  const displayName = conversation.name || (isGroup ? '群聊' : '未知用户');
  const tooltip = isGroup 
    ? `群组: ${displayName}` 
    : (conversation.username ? `@${conversation.username}` : '');
  
  return (
    <div 
      className={`conversation-list-item ${isSelected ? 'selected' : ''} ${isGroup ? 'group-conversation' : ''}`}
      onClick={onSelect}
    >
      <div className="avatar-container">
      <img 
        className="conversation-avatar" 
        src={conversation.avatar || 'https://i.pravatar.cc/40?u=default'} 
          alt={`${displayName} avatar`} 
      />
        {isGroup && <div className="group-indicator">群</div>}
      </div>
      
      <div className="conversation-details">
        <div className="conversation-name-time">
          <span 
            className="conversation-name" 
            title={tooltip}
          >
            {displayName}
          </span>
          <span className="conversation-timestamp">{conversation.timestamp}</span>
        </div>
        <p className="conversation-last-message">
          {conversation.lastMessage}
        </p>
      </div>
    </div>
  );
}

export default ConversationListItem; 