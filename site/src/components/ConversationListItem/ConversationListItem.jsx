import React from 'react';
import './ConversationListItem.css';

function ConversationListItem({ conversation, onSelect, isSelected }) {
  return (
    <div 
      className={`conversation-list-item ${isSelected ? 'selected' : ''}`}
      onClick={onSelect}
    >
      <img 
        className="conversation-avatar" 
        src={conversation.avatar || 'https://i.pravatar.cc/40?u=default'} 
        alt={`${conversation.name} avatar`} 
      />
      <div className="conversation-details">
        <div className="conversation-name-time">
          <span className="conversation-name">{conversation.name}</span>
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