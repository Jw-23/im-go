import React, { useState } from 'react';
import ConversationListItem from '../ConversationListItem/ConversationListItem';
import ContactList from '../ContactList/ContactList';
import './ConversationList.css';
import { useTranslation } from 'react-i18next';
import { FaUserFriends, FaPlus, FaComments } from 'react-icons/fa';

function ConversationList({
  conversations,
  onSelectConversation,
  selectedConversationId,
  isLoading,
  onOpenAddContactModal,
  onInitiateChatWithContact
}) {
  const { t } = useTranslation();
  const [searchTerm, setSearchTerm] = useState('');
  const [viewMode, setViewMode] = useState('conversations');

  const filteredConversations = viewMode === 'conversations' && Array.isArray(conversations)
    ? conversations.filter(convo =>
        convo && typeof convo.name === 'string' &&
        convo.name.toLowerCase().includes(searchTerm.toLowerCase())
      )
    : [];

  const handleSelectContact = (contactUserId) => {
    if (onInitiateChatWithContact) {
      onInitiateChatWithContact(contactUserId);
    }
  };

  return (
    <div className="conversation-list">
      <div className="conversation-list-header">
        <input
          type="text"
          placeholder={
            viewMode === 'conversations' 
              ? t('search_or_start_new_chat') 
              : t('search_contacts')
          }
          className="conversation-search-input"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
        />
        <div className="header-actions">
          <button
            className={`view-mode-button ${viewMode === 'conversations' ? 'active' : ''}`}
            onClick={() => setViewMode('conversations')}
            title={t('show_conversations')}
          >
            <FaComments />
          </button>
          <button
            className={`view-mode-button ${viewMode === 'contacts' ? 'active' : ''}`}
            onClick={() => setViewMode('contacts')}
            title={t('show_contacts')}
          >
            <FaUserFriends />
          </button>
          <button
            className="add-conversation-button"
            onClick={onOpenAddContactModal}
            title={t('add_contact_or_group')}
          >
            <FaPlus />
          </button>
        </div>
      </div>
      
      <div className="conversation-list-items">
        {viewMode === 'conversations' && (
          <>
            {isLoading && <div className="loading-conversations-placeholder">{t('loading_conversations')}</div>}
            {!isLoading && (!Array.isArray(conversations) || conversations.length === 0) && (
              <div className="no-conversations-placeholder">{t('no_conversations_yet')}</div>
            )}
            {!isLoading && Array.isArray(conversations) && conversations.length > 0 && filteredConversations.length === 0 && searchTerm !== '' && (
              <div className="no-conversations-placeholder">{t('no_search_results_for', { searchTerm: searchTerm })}</div>
            )}
            {!isLoading && Array.isArray(filteredConversations) &&
              filteredConversations.map(convo => (
                <ConversationListItem
                  key={convo && convo.id ? convo.id : Math.random()}
                  conversation={convo}
                  onSelect={() => onSelectConversation(convo)}
                  isSelected={convo && convo.id === selectedConversationId}
                />
              ))}
          </>
        )}
        {viewMode === 'contacts' && (
          <ContactList 
            onSelectContact={handleSelectContact}
            isActive={viewMode === 'contacts'}
          />
        )}
      </div>
    </div>
  );
}

export default ConversationList; 