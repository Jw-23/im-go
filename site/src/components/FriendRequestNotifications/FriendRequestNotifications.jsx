import React, { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { getPendingFriendRequests, acceptFriendRequest, rejectFriendRequest } from '../../services/api';
import './FriendRequestNotifications.css'; // We'll need to create this CSS file

const FriendRequestNotifications = ({ isActive, onClose }) => {
  const { t } = useTranslation();
  const [requests, setRequests] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [processingId, setProcessingId] = useState(null); // To disable buttons during API call

  const fetchRequests = useCallback(async () => {
    if (!isActive) return; // Only fetch if the tab/component is active
    setIsLoading(true);
    setError('');
    const result = await getPendingFriendRequests();
    setIsLoading(false);
    if (result.success) {
      setRequests(result.data || []);
      if (result.data.length === 0) {
        // setError(t('no_pending_friend_requests')); // Or just show a message, not as an error
      }
    } else {
      setError(result.error || t('failed_to_fetch_friend_requests'));
    }
  }, [isActive, t]);

  useEffect(() => {
    fetchRequests();
  }, [fetchRequests]);

  const handleAccept = async (requestId) => {
    setProcessingId(requestId);
    const result = await acceptFriendRequest(requestId);
    setProcessingId(null);
    if (result.success) {
      // Refresh list or update UI optimistically
      fetchRequests(); 
      // Optionally, show a success message
    } else {
      setError(result.error || t('failed_to_accept_friend_request'));
      // Optionally, revert optimistic UI update if any
    }
  };

  const handleReject = async (requestId) => {
    setProcessingId(requestId);
    const result = await rejectFriendRequest(requestId);
    setProcessingId(null);
    if (result.success) {
      // Refresh list or update UI optimistically
      fetchRequests();
      // Optionally, show a success message
    } else {
      setError(result.error || t('failed_to_reject_friend_request'));
      // Optionally, revert optimistic UI update if any
    }
  };

  if (!isActive && !onClose) { // If not used in a tab/modal structure where isActive controls visibility
    return null;
  }

  return (
    <div className="friend-request-notifications">
      {/* Optional: Add a header or title if this component is used standalone */} 
      {/* <h3 className="notifications-title">{t('friend_requests_title')}</h3> */}
      
      {isLoading && <p>{t('loading_notifications')}</p>}
      {error && <p className="notifications-error">{error}</p>}
      
      {!isLoading && !error && requests.length === 0 && (
        <p>{t('no_pending_friend_requests_placeholder')}</p>
      )}

      {requests.length > 0 && (
        <ul className="notifications-list">
          {requests.map(req => (
            <li key={req.id} className="notification-item">
              <div className="notification-info">
                {/* Assuming the API returns requester's username or nickname */} 
                <span className="requester-name">
                  {req.requester?.nickname || req.requester?.username || t('unknown_user')}
                </span>
                <span className="request-time">{new Date(req.createdAt).toLocaleDateString()}</span>
              </div>
              <div className="notification-actions">
                <button 
                  onClick={() => handleAccept(req.id)} 
                  disabled={processingId === req.id}
                  className="action-button accept"
                >
                  {processingId === req.id ? t('processing_button_text') : t('accept_button_text')}
                </button>
                <button 
                  onClick={() => handleReject(req.id)} 
                  disabled={processingId === req.id}
                  className="action-button reject"
                >
                  {processingId === req.id ? t('processing_button_text') : t('reject_button_text')}
                </button>
              </div>
            </li>
          ))}
        </ul>
      )}
      {/* {onClose && <button onClick={onClose}>{t('close_button_text')}</button>} */} 
    </div>
  );
};

export default FriendRequestNotifications; 