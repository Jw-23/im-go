import React, { useState } from 'react';
import './ChatLayout.css';
import { ConversationList, ChatWindow } from '..';
import SettingsButton from '../SettingsButton/SettingsButton';
import { Trans } from 'react-i18next';

// 图标组件
const PlusIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16">
    <path d="M8 4a.5.5 0 0 1 .5.5v3h3a.5.5 0 0 1 0 1h-3v3a.5.5 0 0 1-1 0v-3h-3a.5.5 0 0 1 0-1h3v-3A.5.5 0 0 1 8 4z"/>
  </svg>
);

const UsersIcon = () => (
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16">
    <path d="M15 14s1 0 1-1-1-4-5-4-5 3-5 4 1 1 1 1h8zm-7.978-1A.261.261 0 0 1 7 12.996c.001-.264.167-1.03.76-1.72C8.312 10.629 9.282 10 11 10c1.717 0 2.687.63 3.24 1.276.593.69.758 1.457.76 1.72l-.008.002a.274.274 0 0 1-.014.002H7.022zM11 7a2 2 0 1 0 0-4 2 2 0 0 0 0 4zm3-2a3 3 0 1 1-6 0 3 3 0 0 1 6 0zM6.936 9.28a5.88 5.88 0 0 0-1.23-.247A7.35 7.35 0 0 0 5 9c-4 0-5 3-5 4 0 .667.333 1 1 1h4.216A2.238 2.238 0 0 1 5 13c0-1.01.377-2.042 1.09-2.904.243-.294.526-.569.846-.816zM4.92 10A5.493 5.493 0 0 0 4 13H1c0-.26.164-1.03.76-1.724.545-.636 1.492-1.256 3.16-1.275zM1.5 5.5a3 3 0 1 1 6 0 3 3 0 0 1-6 0zm3-2a2 2 0 1 0 0 4 2 2 0 0 0 0-4z"/>
  </svg>
);

// 聊天应用的主要布局组件
const ChatLayout = ({ 
  // 状态属性
  conversations, 
  selectedConversation, 
  messages, 
  isLoadingConversations,
  isLoadingMessages,
  hasMoreMessages,
  websocketConnected,
  contacts = [], // 联系人列表
  isLoadingContacts = false, // 联系人加载状态
  
  // 事件处理函数
  onSelectConversation,
  onSendMessage,
  onLoadMoreMessages,
  onOpenAddContactModal,
  onInitiateChatWithContact,
  onOpenSettings,
  onOpenCreateGroupModal, // 创建群聊事件
  onManageGroupMembers, // 管理群成员事件
  
  // 国际化
  t
}) => {
  const [activeTab, setActiveTab] = useState('chats');

  // 联系人列表组件
  const ContactListComponent = () => (
    <div className="contact-list">
      {isLoadingContacts ? (
        <div className="loading-contacts">
          <p>加载联系人中...</p>
        </div>
      ) : contacts.length === 0 ? (
        <div className="empty-contacts">
          <p>暂无联系人</p>
          <button onClick={onOpenAddContactModal} className="add-contact-btn">添加联系人</button>
        </div>
      ) : (
        <>
          <div className="contacts-header">
            <span>联系人 ({contacts.length})</span>
            <button onClick={onOpenAddContactModal} className="add-contact-btn small">
              <PlusIcon /> 添加
            </button>
          </div>
          {contacts.map((contact) => (
            <div 
              key={contact.id} 
              className="contact-item"
              onClick={() => onInitiateChatWithContact(contact.id)}
            >
              <img 
                src={contact.avatar || `https://ui-avatars.com/api/?name=${encodeURIComponent((contact.name || contact.username || 'User').charAt(0))}&size=40&background=random&color=fff`} 
                alt={`${contact.name || contact.username || 'User'} avatar`} 
                className="contact-avatar"
              />
              <div className="contact-info">
                <div className="contact-name">{contact.name || contact.username || contact.nickname || `用户${contact.id}`}</div>
                {contact.status && <div className="contact-status">{contact.status}</div>}
              </div>
            </div>
          ))}
        </>
      )}
    </div>
  );

  return (
    <div className="app-container">
      <aside className="sidebar">
        <div className="sidebar-header">
          <div className="tabs">
            <button 
              className={activeTab === 'chats' ? 'active' : ''} 
              onClick={() => setActiveTab('chats')}
            >
              聊天
            </button>
            <button 
              className={activeTab === 'contacts' ? 'active' : ''} 
              onClick={() => setActiveTab('contacts')}
            >
              联系人
            </button>
          </div>
          <button className="create-group-btn" onClick={onOpenCreateGroupModal}>
            <PlusIcon /> 新建群聊
          </button>
        </div>
        
        {activeTab === 'chats' ? (
          <ConversationList 
            conversations={conversations} 
            onSelectConversation={onSelectConversation}
            selectedConversationId={selectedConversation?.id}
            isLoading={isLoadingConversations}
            onOpenAddContactModal={onOpenAddContactModal}
            onInitiateChatWithContact={onInitiateChatWithContact}
          />
        ) : (
          <ContactListComponent />
        )}
        <SettingsButton onClick={onOpenSettings} />
      </aside>
      <main className="chat-area">
        {selectedConversation ? (
          <ChatWindow 
            conversation={selectedConversation} 
            messages={messages} 
            onSendMessage={onSendMessage} 
            onLoadMoreMessages={onLoadMoreMessages} 
            isLoadingMoreMessages={isLoadingMessages}
            hasMoreMessages={hasMoreMessages}
            websocketConnected={websocketConnected}
            onManageGroupMembers={onManageGroupMembers}
            isGroup={selectedConversation.type === 'group'}
          />
        ) : (
          <div className="no-chat-selected">
            <p>{isLoadingConversations 
              ? '加载会话中...' 
              : (conversations.length === 0 
                ? '暂无会话，请选择一个联系人开始聊天' 
                : '请选择一个会话开始聊天')}
            </p>
          </div>
        )}
      </main>
    </div>
  );
};

// 登录提示组件
export const AuthPrompt = ({ t, onOpenSettings }) => (
  <div className="app-container--auth"> 
    <div className="auth-prompt">
      <h1>{t('welcome_message_placeholder_title')}</h1>
      <p>
        <Trans i18nKey="login_or_register_to_continue_placeholder">
          Please <button type="button" onClick={onOpenSettings} className="auth-prompt__button">login or register</button> to continue.
        </Trans>
      </p>
    </div>
  </div>
);

export default ChatLayout; 