import React, { useState } from 'react';
import './AddContactModal.css';
import { useTranslation } from 'react-i18next';
import { searchUsers, sendFriendRequest } from '../../services/api';
import FriendRequestNotifications from '../FriendRequestNotifications/FriendRequestNotifications';

const AddContactModal = ({ isOpen, onClose }) => {
  const { t } = useTranslation();
  const [searchTerm, setSearchTerm] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [isLoadingSearch, setIsLoadingSearch] = useState(false);
  const [message, setMessage] = useState('');
  const [addingContactId, setAddingContactId] = useState(null);
  const [activeTab, setActiveTab] = useState('add'); // 'add' or 'notifications'

  if (!isOpen) {
    return null;
  }

  const handleSearch = async (e) => {
    e.preventDefault();
    if (!searchTerm.trim() || searchTerm.trim().length < 2) {
        setMessage(t('search_term_too_short'));
        setSearchResults([]);
        return;
    }
    
    setIsLoadingSearch(true);
    setMessage('');
    setSearchResults([]);

    const result = await searchUsers(searchTerm);

    setIsLoadingSearch(false);
    if (result.success) {
      if (result.data.length === 0) {
        setMessage(t('no_users_found_for_search', { searchTerm }));
      }
      setSearchResults(result.data.map(user => ({ ...user, requestSent: false })));
    } else {
      console.error("Search error from API:", result.error);
      setMessage(result.error || t('search_failed_error'));
    }
  };

  const handleAddContact = async (userId) => {
    setAddingContactId(userId);
    setMessage('');

    const result = await sendFriendRequest(userId);

    setAddingContactId(null);
    if (result.success) {
      setMessage(result.data?.message || t('friend_request_sent_successfully'));
      setSearchResults(prevResults =>
        prevResults.map(user =>
          user.id === userId ? { ...user, requestSent: true } : user
        )
      );
    } else {
      console.error("Add contact error from API:", result.error);
      setMessage(result.error || t('add_contact_failed_error'));
    }
  };
  
  const handleCloseModal = () => {
    setSearchTerm('');
    setSearchResults([]);
    setMessage('');
    setIsLoadingSearch(false);
    setAddingContactId(null);
    setActiveTab('add'); // Reset to add tab on close
    onClose(); 
  };

  return (
    <div className="add-contact-modal__overlay" onClick={handleCloseModal}>
      <div className="add-contact-modal__content" onClick={(e) => e.stopPropagation()}>
        <div className="add-contact-modal__tabs">
          <button 
            className={`add-contact-modal__tab-button ${activeTab === 'add' ? 'active' : ''}`}
            onClick={() => setActiveTab('add')}
          >
            {t('add_tab_title')}
          </button>
          <button 
            className={`add-contact-modal__tab-button ${activeTab === 'notifications' ? 'active' : ''}`}
            onClick={() => setActiveTab('notifications')}
          >
            {t('notifications_tab_title')}
          </button>
        </div>
        
        <div className="add-contact-modal__body">
          {activeTab === 'add' && (
            <>
              <form onSubmit={handleSearch} className="user-search-form">
                <input
                  type="text"
                  placeholder={t('search_by_username_placeholder')}
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="user-search-input"
                  disabled={isLoadingSearch}
                />
                <button type="submit" className="user-search-button" disabled={isLoadingSearch}>
                  {isLoadingSearch ? t('searching_button_text') : t('search_button_text')}
                </button>
              </form>
              
              {message && <p className="search-message">{message}</p>}

              {searchResults.length > 0 && (
                <ul className="search-results-list">
                  {searchResults.map(user => (
                    <li key={user.id} className="search-result-item">
                      <div className="user-info">
                        <span className="user-name">{user.nickname || user.username}</span>
                        <span className="user-username">(@{user.username})</span>
                      </div>
                      <button
                        onClick={() => handleAddContact(user.id)}
                        className="add-button"
                        disabled={addingContactId === user.id || user.requestSent}
                      >
                        {addingContactId === user.id
                          ? t('adding_button_text')
                          : user.requestSent
                          ? t('request_sent_button_text')
                          : t('add_button_text')}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}
          {activeTab === 'notifications' && (
            <FriendRequestNotifications isActive={activeTab === 'notifications'} />
          )}
        </div>
      </div>
    </div>
  );
};

export default AddContactModal; 