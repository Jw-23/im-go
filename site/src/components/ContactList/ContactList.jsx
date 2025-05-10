import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { getFriendsList } from '../../services/api';
import './ContactList.css'; // Create this CSS file

const ContactList = ({ onSelectContact, isActive }) => {
  const { t } = useTranslation();
  const [contacts, setContacts] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const fetchContacts = useCallback(async () => {
    // Only fetch if the component/tab is active
    if (!isActive) return; 
    
    setIsLoading(true);
    setError('');
    const result = await getFriendsList();
    setIsLoading(false);

    if (result.success) {
      setContacts(result.data || []);
      if (!result.data || result.data.length === 0) {
        // Maybe show a placeholder instead of error
        // setError(t('no_contacts_yet'));
      }
    } else {
      setError(result.error || t('failed_to_fetch_contacts'));
    }
  }, [isActive, t]);

  useEffect(() => {
    fetchContacts();
  }, [fetchContacts]); // Dependency array includes fetchContacts which includes isActive

  const handleContactClick = (contact) => {
    if (onSelectContact) {
      onSelectContact(contact.id); // Pass the contact's ID
    }
  };

  return (
    <div className="contact-list-container">
      {isLoading && <p className="loading-placeholder">{t('loading_contacts')}</p>}
      {error && <p className="error-message">{error}</p>}
      {!isLoading && !error && contacts.length === 0 && (
        <p className="empty-placeholder">{t('no_contacts_placeholder')}</p>
      )}
      {!isLoading && contacts.length > 0 && (
        <ul className="contact-list">
          {contacts.map(contact => (
            <li key={contact.id} className="contact-list-item" onClick={() => handleContactClick(contact)}>
              {/* Basic display - enhance with avatar later */} 
              <div className="contact-avatar">{/* Avatar Placeholder */}</div>
              <div className="contact-info">
                <span className="contact-name">{contact.nickname || contact.username}</span>
                {/* Maybe add status later */} 
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
};

export default ContactList; 