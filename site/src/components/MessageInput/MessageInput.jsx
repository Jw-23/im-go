import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import './MessageInput.css';
import { FaPaperPlane } from 'react-icons/fa'; // Send icon

const MessageInput = ({ onSendMessage }) => {
  const { t } = useTranslation();
  const [messageText, setMessageText] = useState('');

  const handleInputChange = (e) => {
    setMessageText(e.target.value);
    // TODO: Adjust textarea height dynamically if needed
  };

  const handleSend = (e) => {
    e.preventDefault(); // Prevent form submission if wrapped in form
    const trimmedMessage = messageText.trim();
    if (trimmedMessage) {
      console.log("[MessageInput.jsx]","trimmedMessage:",trimmedMessage)
      onSendMessage(trimmedMessage); // Pass the text content up
      setMessageText(''); // Clear input after sending
    }
  };

  // Allow sending with Enter key, Shift+Enter for newline
  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend(e);
    }
  };

  return (
    <div className="message-input-container">
      {/* TODO: Add buttons for attachments later */}
      <textarea
        className="message-textarea"
        placeholder={t('type_your_message_placeholder')} // Need translation
        value={messageText}
        onChange={handleInputChange}
        onKeyDown={handleKeyDown}
        rows="1" // Start with one row, can expand
      />
      <button 
        className="send-button" 
        onClick={handleSend} 
        disabled={!messageText.trim()}
        aria-label={t('send_message_button_label')} // Need translation
      >
        <FaPaperPlane />
      </button>
    </div>
  );
};

export default MessageInput; 