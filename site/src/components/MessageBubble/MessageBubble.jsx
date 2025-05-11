import React from 'react';
import {
  FaFileAlt, // Icon for generic files
  FaImage,   // Icon for images
  FaCheckCircle, // 发送成功图标
  FaExclamationCircle, // 发送失败图标
  FaClock // 发送中图标
} from 'react-icons/fa';
import './MessageBubble.css';

const MessageBubble = ({ message, isSender }) => {
  const bubbleClass = isSender ? 'message-bubble sender' : 'message-bubble receiver';
  const timeClass = isSender ? 'time-sender' : 'time-receiver';

  // Helper function to format time (example)
  const formatTime = (timestamp) => {
    if (!timestamp) return '';
    return new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false });
  };

  const renderContent = () => {
    switch (message.type) {
      case 'text':
        return <span className="message-text">{message.content}</span>;
      case 'image':
        return (
          <div className="message-media image">
            <img src={message.content} alt={message.fileName || 'Image'} />
            {message.fileName && <span className="file-name">{message.fileName}</span>}
            {/* Maybe add download button later */} 
          </div>
        );
      case 'file':
        return (
          <div className="message-media file">
            <FaFileAlt className="file-icon" />
            <div className="file-details">
                <span className="file-name">{message.fileName || 'File'}</span>
                {message.fileSize && <span className="file-size">({(message.fileSize / 1024 / 1024).toFixed(2)} MB)</span>}
            </div>
            {/* Link to download the file - assuming content is the URL */}
            <a href={message.content} target="_blank" rel="noopener noreferrer" className="download-link">Download</a>
          </div>
        );
      // Add cases for other message types like 'system' if needed
      default:
        return <span className="message-text">Unsupported message type</span>;
    }
  };

  // 渲染消息状态指示器
  const renderStatusIndicator = () => {
    if (!isSender) return null; // 只有发送者的消息才显示状态

    switch (message.status) {
      case 'sending':
        return <FaClock className="message-status-icon sending" title="发送中" />;
      case 'failed':
        return <FaExclamationCircle className="message-status-icon failed" title="发送失败" />;
      case 'sent':
      default:
        return <FaCheckCircle className="message-status-icon sent" title="已发送" />;
    }
  };

  return (
    <div className={bubbleClass}>
      <div className="message-content">
        {renderContent()}
      </div>
      <div className="message-meta">
        <span className={timeClass}>{formatTime(message.timestamp)}</span>
        {renderStatusIndicator()}
      </div>
    </div>
  );
};

export default MessageBubble; 