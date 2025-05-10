import React, { useState, useEffect, useCallback } from 'react';
import './App.css';
import { ConversationList, ChatWindow } from './components'; // Updated import using the barrel file
import SettingsButton from './components/SettingsButton/SettingsButton';
import SettingsModal from './components/SettingsModal/SettingsModal';
import AddContactModal from './components/AddContactModal/AddContactModal'; // ADDED
import { useAuth } from './contexts/AuthContext'; // Import useAuth
// import io from 'socket.io-client'; // WebSocket client REMOVED
import { useTranslation, Trans } from 'react-i18next'; // Import Trans for components within strings
import { initiatePrivateConversation, getConversationMessages, getConversations } from './services/api'; // Import the new API function and getConversationMessages

// WebSocket server URL (adjust if your chat server runs elsewhere)
const CHAT_SERVER_URL = 'ws://localhost:8080'; // From WebSocket_API.md CHAT_SERVER_PORT
const WEBSOCKET_PATH = '/ws/chat';             // From WebSocket_API.md

const DEFAULT_MESSAGE_LIMIT = 20; // Define a constant for message limit per fetch

// Helper function to format conversation data from API
const formatApiConversation = (apiConvo, currentUserId) => {
  let name = apiConvo.name;
  let avatar = apiConvo.avatarUrl; // Prefer apiConvo.avatarUrl if it exists from enriched API
  let targetId = null;
  let isOnline = false;
  // console.log("[App.jsx]","apiConvo:",apiConvo)

  // This part tries to get name/avatar if the API response is minimal (older structure)
  // If your API now directly provides name/avatar for the conversation, much of this can be simplified.
  if (apiConvo.targetId && apiConvo.type === 'private') {
    targetId = apiConvo.targetId;
    // If name/avatar are not directly on apiConvo for private chats, they might be missing
    // The API.md update suggests 'name' and 'avatar' should be top-level now.
    if (!name) name = apiConvo.name; // Should be provided by updated API
    if (!avatar) avatar = apiConvo.avatar; // Should be provided by updated API

  } else if (apiConvo.type === 'private' && apiConvo.participants) {
    const otherParticipant = apiConvo.participants.find(p => p.id !== currentUserId);
    if (otherParticipant) {
      targetId = otherParticipant.id;
      name = name || otherParticipant.nickname || otherParticipant.username;
      avatar = avatar || otherParticipant.avatarUrl;
      isOnline = otherParticipant.isOnline || false;
    }
  } else if (apiConvo.type === 'private' && apiConvo.otherParticipant) { // Legacy structure
    name = name || apiConvo.otherParticipant.nickname || apiConvo.otherParticipant.username;
    avatar = avatar || apiConvo.otherParticipant.avatarUrl;
    targetId = targetId || apiConvo.otherParticipant.id;
    isOnline = apiConvo.otherParticipant.isOnline || false;
  } else if (apiConvo.type === 'group') {
    // For groups, name and avatar should be directly on apiConvo from the updated API
    name = name || apiConvo.name || 'Group Chat';
    avatar = avatar || apiConvo.avatarUrl;
    targetId = apiConvo.id; 
  }

  // Fallback for name if still not determined (e.g. private chat with missing participant data from API)
  if (!name && targetId && apiConvo.type === 'private') {
    name = `User ${targetId}`;
  }

  // MODIFIED: Fallback for avatar - use first letter of name if avatar is missing
  if (!avatar && name) {
    const firstLetter = name.charAt(0).toUpperCase();
    // Use ui-avatars.com for generating an image with the first letter
    // You can customize background, color, font size, etc. as needed.
    avatar = `https://ui-avatars.com/api/?name=${encodeURIComponent(firstLetter)}&size=40&background=random&color=fff`;
  } else if (!avatar) {
    // Ultimate fallback if no name and no avatar
    avatar = `https://i.pravatar.cc/40?u=${apiConvo.id}`; // Default Pravatar if no name for ui-avatars
  }

  return {
    id: apiConvo.id.toString(),
    name: name,
    lastMessage: apiConvo.lastMessage?.content || apiConvo.lastMessage?.text || '', // Adjusted to check for text for older model compatibility
    timestamp: apiConvo.lastMessage?.timestamp
      ? new Date(apiConvo.lastMessage.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: true })
      : (apiConvo.updatedAt ? new Date(apiConvo.updatedAt).toLocaleDateString() : ''),
    avatar: avatar,
    type: apiConvo.type,
    targetId: targetId, // Ensure targetId is correctly assigned
    isOnline: isOnline,
    unreadCount: apiConvo.unreadCount || 0,
    lastMessageTimestamp: apiConvo.lastMessage?.timestamp || apiConvo.updatedAt,
  };
};

function App() {
  const [selectedConversation, setSelectedConversation] = useState(null);
  const [isSettingsModalOpen, setIsSettingsModalOpen] = useState(false);
  const [isAddContactModalOpen, setIsAddContactModalOpen] = useState(false); // ADDED state for new modal
  const [conversations, setConversations] = useState([]); // MODIFIED: Initialize with empty array
  const [messages, setMessages] = useState({});
  const [websocket, setWebsocket] = useState(null); // MODIFIED: socket to websocket
  const [isLoadingConversations, setIsLoadingConversations] = useState(false); // ADDED: For loading state

  // ADDED: States for message loading and pagination
  const [messageOffsets, setMessageOffsets] = useState({}); // { [convoId]: number (current offset) }
  const [hasMoreMessages, setHasMoreMessages] = useState({}); // { [convoId]: boolean }
  const [isLoadingMessages, setIsLoadingMessages] = useState({}); // { [convoId]: boolean }

  const { token, currentUser, isLoading: isAuthLoading } = useAuth();
  const { t } = useTranslation();

  // Automatically open settings modal if not logged in AND auth check is complete
  useEffect(() => {
    if (!isAuthLoading && !currentUser && !isSettingsModalOpen) {
      setIsSettingsModalOpen(true);
    }
    // Optional: If user logs in and modal is still open for auth, and it was opened for auth, close it.
    // This logic can be refined, e.g. by passing a prop to SettingsModal indicating why it was opened.
  }, [isAuthLoading, currentUser, isSettingsModalOpen]);

  // useEffect to log selectedConversation changes
  useEffect(() => {
    console.log('[App.jsx] selectedConversation state updated to:', selectedConversation);
  }, [selectedConversation]);

  // ADDED: Fetch conversations from API
  const fetchConversations = useCallback(async () => {
    if (!token || !currentUser) return;
    setIsLoadingConversations(true);
    try {
      const result = await getConversations(); // Assuming you have an API service function
      if (!result.success) {
        console.error("Failed to fetch conversations:", result.error);
        setConversations([]);
        return;
      }
      const data = result.data || [];
      console.log("[App.jsx]","API data for conversations: ", data);

      const formattedConversations = data.map(apiConvo => formatApiConversation(apiConvo, currentUser.id));
      console.log("[App.jsx]","Formatted conversations for state: ", formattedConversations);

      setConversations(formattedConversations.sort((a, b) => {
        // Ensure lastMessageTimestamp is a valid date string or null/undefined
        const timeA = a.lastMessageTimestamp ? new Date(a.lastMessageTimestamp).getTime() : 0;
        const timeB = b.lastMessageTimestamp ? new Date(b.lastMessageTimestamp).getTime() : 0;
        
        // To keep current sorting (newest at top if ConversationList renders 0..N top to bottom):
        // return timeB - timeA; 
        // If you want newest at the bottom (based on your previous change):
        return timeA - timeB; 
      }));
    } catch (error) {
      console.error("Error during fetchConversations (catch block):", error);
      setConversations([]); 
    } finally {
      setIsLoadingConversations(false);
    }
  }, [currentUser, token]);

  useEffect(() => {
    if (currentUser && token) {
      fetchConversations();
    } else {
      setConversations([]);
      setSelectedConversation(null);
      setMessages({});
    }
  }, [currentUser, token, fetchConversations]);

  // WebSocket connection setup
  useEffect(() => {
    console.log('[App.jsx] WebSocket useEffect triggered. Token:', token, 'CurrentUser:', currentUser);

    if (token && currentUser) {
      const wsUrl = `${CHAT_SERVER_URL}${WEBSOCKET_PATH}?token=${token}`;
      console.log('[App.jsx] Preparing to connect WebSocket. URL:', wsUrl); // Log URL
      const wsInstance = new WebSocket(wsUrl);

      wsInstance.onopen = () => {
        console.log('[App.jsx] WebSocket connection established (onopen event). wsInstance.readyState:', wsInstance.readyState);
        setWebsocket(wsInstance);
      };

      wsInstance.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          console.log('[App.jsx] WebSocket message received:', message);
          // Ensure message has conversationId to correctly store it
          if (message.conversationId) {
            setMessages(prevMessages => {
              const existingMessages = prevMessages[message.conversationId] || [];
              return {
                ...prevMessages,
                [message.conversationId]: [...existingMessages, message],
              };
            });
            // Potentially update conversation's last message and re-sort
            setConversations(prevConversations => {
              const convoIndex = prevConversations.findIndex(c => c.id === message.conversationId);
              if (convoIndex > -1) {
                const updatedConvo = {
                  ...prevConversations[convoIndex],
                  lastMessage: message.content,
                  timestamp: new Date(message.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: true }),
                  lastMessageTimestamp: message.timestamp,
                };
                let newConversations = [...prevConversations];
                newConversations.splice(convoIndex, 1);
                newConversations.unshift(updatedConvo);
                return newConversations.sort((a, b) => {
                    const dateA = new Date(a.lastMessageTimestamp).getTime() || 0;
                    const dateB = new Date(b.lastMessageTimestamp).getTime() || 0;
                    return dateB - dateA;
                });
              }
              return prevConversations;
            });

          } else {
            console.warn('[App.jsx] Received WebSocket message without conversationId:', message);
          }
        } catch (error) {
          console.error('[App.jsx] Error parsing WebSocket message or updating state:', error);
        }
      };

      wsInstance.onerror = (error) => {
        console.error('[App.jsx] WebSocket error (onerror event):', error);
      };

      wsInstance.onclose = (event) => {
        console.log(
          '[App.jsx] WebSocket connection closed (onclose event). Code:', event.code, 
          'Reason:', event.reason, 
          'wasClean:', event.wasClean,
          'wsInstance.readyState at close:', wsInstance.readyState 
        );
        setWebsocket(null); // Ensure websocket state is cleared on close
      };

      return () => {
        console.log('[App.jsx] WebSocket useEffect cleanup function running. wsInstance.readyState at cleanup start:', wsInstance.readyState);
        if (wsInstance.readyState === WebSocket.OPEN || wsInstance.readyState === WebSocket.CONNECTING) {
          console.log('[App.jsx] Closing WebSocket in cleanup.');
          wsInstance.close(1000, "Component unmounting or dependencies changed"); // Add close code and reason
        } else {
          console.log('[App.jsx] WebSocket already closed or in closing state during cleanup. ReadyState:', wsInstance.readyState);
        }
        setWebsocket(null); // Also ensure websocket state is cleared in cleanup if not already by onclose
        console.log('[App.jsx] WebSocket state set to null in cleanup.');
      };
    } else {
      console.log('[App.jsx] WebSocket useEffect: No token or currentUser, ensuring WebSocket is not active.');
      // If no token or user, ensure WebSocket is closed and cleared
      if (websocket) { // Check against the current state variable 'websocket'
        console.log('[App.jsx] Closing existing WebSocket due to no token/currentUser. ReadyState:', websocket.readyState);
        websocket.close(1000, "User logged out or token expired");
        setWebsocket(null); // Clear the state
        console.log('[App.jsx] Existing WebSocket closed and state set to null.');
      }
    }
  }, [token, currentUser]);

  // ADDED: Function to fetch messages for a conversation
  const fetchMessages = useCallback(async (conversationId, offset = 0, limit = DEFAULT_MESSAGE_LIMIT, isInitialFetch = true) => {
    if (!conversationId) return; // Token check is handled by the request function in api.js
    console.log(`[App.jsx] Fetching messages for conversation ${conversationId}, offset: ${offset}, limit: ${limit}, initial: ${isInitialFetch}`);

    setIsLoadingMessages(prev => ({ ...prev, [conversationId]: true }));

    try {
      // MODIFIED: Use getConversationMessages from api.js
      const result = await getConversationMessages(conversationId, limit, offset);

      if (!result.success) {
        console.error(`Failed to fetch messages for conversation ${conversationId}, status: ${result.status}, error: ${result.error}`);
        setHasMoreMessages(prev => ({ ...prev, [conversationId]: false }));
        return;
      }

      const fetchedMessages = result.data || []; // API returns newest to oldest
      console.log(`[App.jsx] Fetched ${fetchedMessages.length} messages for ${conversationId} (API order - newest to oldest):`);
      if (fetchedMessages.length > 0) {
        console.log('[App.jsx] Example of a fetched historical message (first one):', JSON.stringify(fetchedMessages[0]));
        if (fetchedMessages.length > 1) {
          console.log('[App.jsx] Example of another fetched historical message (second one):', JSON.stringify(fetchedMessages[1]));
        }
      }

      // Reverse the fetched messages to store them in chronological order (oldest to newest)
      let chronologicallyFetchedMessages = [...fetchedMessages].reverse();
      
      // Map sentAt to timestamp for consistency with MessageBubble and potentially other parts of the app
      chronologicallyFetchedMessages = chronologicallyFetchedMessages.map(msg => ({
        ...msg,
        timestamp: msg.sentAt, // Ensure timestamp field exists
      }));

      console.log(`[App.jsx] Chronologically ordered messages for ${conversationId} (oldest to newest, with timestamp field mapped):`, chronologicallyFetchedMessages);

      setMessages(prevMessages => {
        const currentMessages = prevMessages[conversationId] || []; // Should be oldest to newest
        // When loading more older messages (isInitialFetch = false), prepend the new (older) messages.
        // When it's an initial fetch, just set the messages.
        const newMessages = isInitialFetch
          ? chronologicallyFetchedMessages
          : [...chronologicallyFetchedMessages, ...currentMessages];
        
        // Ensure no duplicates if any overlap, though offset logic should prevent this.
        // For simplicity, we assume no duplicates for now. A more robust solution might involve checking IDs.

        console.log(`[App.jsx] Updating messages state for ${conversationId}. Initial: ${isInitialFetch}, New count: ${newMessages.length}`);
        return {
          ...prevMessages,
          [conversationId]: newMessages,
        };
      });

      // The offset for the *next* fetch should be the count of *all currently displayed* messages if API uses offset as starting index
      // Or, if offset means "skip N messages", then currentOffset + fetchedMessages.length is correct.
      // Assuming offset is "skip N messages" as per typical pagination.
      setMessageOffsets(prevOffsets => ({ ...prevOffsets, [conversationId]: offset + fetchedMessages.length }));
      setHasMoreMessages(prev => ({ ...prev, [conversationId]: fetchedMessages.length === limit }));

    } catch (error) { // This catch is for unexpected errors not handled by the api.js request function
      console.error(`Critical error fetching messages for conversation ${conversationId}:`, error);
      setHasMoreMessages(prev => ({ ...prev, [conversationId]: false }));
    } finally {
      setIsLoadingMessages(prev => ({ ...prev, [conversationId]: false }));
    }
  }, []); // Removed token from deps as api.js handles it

  const handleSelectConversation = async (conversation) => {
    console.log('[App.jsx] handleSelectConversation called with (name, id, targetId):', conversation?.name, conversation?.id, conversation?.targetId);
    if (!conversation || !conversation.id) {
        console.warn('[App.jsx] handleSelectConversation: Invalid conversation object', conversation);
        setSelectedConversation(null); // Clear selection if invalid
        return;
    }

    setSelectedConversation(conversation);
    
    // Reset messages for the new conversation or clear them before fetching
    // setMessages(prev => ({ ...prev, [conversation.id]: [] })); // Option 1: Clear immediately
    
    console.log(`[App.jsx] Selected conversation ${conversation.id}, type: ${conversation.type}, targetId: ${conversation.targetId}`);
    
    // Fetch initial messages for the selected conversation
    // We ensure it only fetches if not already loading for this convo, though fetchMessages itself handles its loading state.
    if (!isLoadingMessages[conversation.id]) {
      fetchMessages(conversation.id, 0, DEFAULT_MESSAGE_LIMIT, true);
    }
  };

  // 发送消息
  const handleSendMessage = (msg) => {
    if (!msg || !selectedConversation || !websocket || websocket.readyState !== WebSocket.OPEN || !currentUser) {
      console.warn('[App.jsx] Cannot send message. Conditions not met:', { msg, selectedConversation, websocket, currentUser });
      if (websocket && websocket.readyState !== WebSocket.OPEN) {
        console.warn(`[App.jsx] WebSocket not open. ReadyState: ${websocket.readyState}`);
      }
      return;
    }
    const timestamp = new Date().toISOString();
    const newMessagePayload = {
      type: 'text', // As per WebSocket_API.md
      content: msg,
      // receiverId for private chat is the other user's ID. For group chat, it's the GroupID/ConversationID.
      receiverId: selectedConversation.type === 'private'
        ? selectedConversation.targetId.toString() 
        : selectedConversation.id, // Assuming group chat uses conversation.id as receiverId (GroupID)
      conversationId: selectedConversation.id, // Message belongs to this conversation
      timestamp,
      // senderId can be omitted as server will derive it from token, but can be included for clarity or client-side use.
      // senderId: currentUser.id.toString(), 
    };

    console.log('[App.jsx] Sending WebSocket message:', newMessagePayload);
    websocket.send(JSON.stringify(newMessagePayload));

    // 本地乐观更新 for messages state
    const optimisticMessage = {
      id: `msg_${Date.now()}`,
      type: 'text',
      content: msg,
      senderId: currentUser.id.toString(), // Assuming currentUser is available and has id
      conversationId: selectedConversation.id,
      timestamp,
      status: 'sending' // You might want a 'sent' status from server ack later
    };
    setMessages(prevMessages => ({
      ...prevMessages,
      [selectedConversation.id]: [...(prevMessages[selectedConversation.id] || []), optimisticMessage]
    }));

    // Optimistic update for conversations state
    setConversations(prevConversations => {
      const convoIndex = prevConversations.findIndex(c => c.id === selectedConversation.id);
      if (convoIndex > -1) {
        const updatedConvo = {
          ...prevConversations[convoIndex],
          lastMessage: msg,
          timestamp: new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: true }),
          lastMessageTimestamp: timestamp,
        };
        const newConversations = [...prevConversations];
        newConversations.splice(convoIndex, 1);    // Remove old version
        newConversations.unshift(updatedConvo);    // REVERTED: Add updated version to the beginning
        return newConversations; // Newest (this one) is now at the top
      } 
      // This case should ideally not happen if a message is sent within an existing, selected conversation.
      // If it's a new conversation being created by the first message, this logic would need to be more complex,
      // potentially creating a new conversation object here or relying on server to send new convo info.
      console.warn('[App.jsx] Optimistic update: Could not find conversation in list to update:', selectedConversation.id);
      return prevConversations;
    });
  };

  const openSettingsModal = () => setIsSettingsModalOpen(true);
  const closeSettingsModal = () => setIsSettingsModalOpen(false);

  const openAddContactModal = () => setIsAddContactModalOpen(true); // ADDED function
  const closeAddContactModal = () => setIsAddContactModalOpen(false); // ADDED function

  // New handler for initiating chat from ContactList
  const handleInitiateChatWithContact = async (contactId) => {
    if (!token || !currentUser) {
      console.warn('[App.jsx] handleInitiateChatWithContact: No token or currentUser. Aborting.');
      return;
    }
    console.log(`[App.jsx] Attempting to initiate chat with contact ID: ${contactId}`);
    try {
      // const response = await fetch('/api/v1/conversations/private', {
      //   method: 'POST',
      //   headers: {
      //     'Authorization': `Bearer ${token}`,
      //     'Content-Type': 'application/json',
      //   },
      //   body: JSON.stringify({ userId: contactId }),
      // });

      // if (!response.ok) {
      //   let errorData = { message: `Request failed with status ${response.status}` };
      //   try {
      //     errorData = await response.json();
      //   } catch (e) {
      //     console.warn('[App.jsx] Failed to parse error JSON from API');
      //   }
      //   console.error(`[App.jsx] Failed to initiate private conversation with ${contactId}, status: ${response.status}`, errorData);
      //   // TODO: Show user-friendly error
      //   return;
      // }

      // const apiConversation = await response.json();
      
      const result = await initiatePrivateConversation(contactId);

      if (!result.success) {
        console.error(`[App.jsx] Failed to initiate private conversation with ${contactId}, status: ${result.status}`, result.error);
        // TODO: Show user-friendly error (e.g., using a notification system)
        alert(`Error: ${result.error || 'Could not start chat.'}`); // Simple alert for now
        return;
      }
      
      const apiConversation = result.data;
      console.log('[App.jsx] API response for new/existing private conversation:', apiConversation);

      if (!apiConversation) {
        console.error('[App.jsx] API succeeded but returned no conversation data.');
        alert('Error: Could not retrieve conversation data.');
        return;
      }

      const formattedNewConversation = formatApiConversation(apiConversation, currentUser.id);
      console.log('[App.jsx] Formatted new/existing private conversation:', formattedNewConversation);

      // Update conversations list if new, then select it
      setConversations(prevConversations => {
        const existingConvoIndex = prevConversations.findIndex(c => c.id === formattedNewConversation.id);
        let updatedConversations;
        if (existingConvoIndex > -1) {
          console.log('[App.jsx] Conversation already exists, updating it.');
          updatedConversations = [...prevConversations];
          updatedConversations[existingConvoIndex] = formattedNewConversation; // Update existing
        } else {
          console.log('[App.jsx] New conversation, adding to list.');
          updatedConversations = [formattedNewConversation, ...prevConversations]; // REVERTED: Add to beginning
        }
        return updatedConversations.sort((a, b) => {
            const dateA = new Date(a.lastMessageTimestamp).getTime() || 0;
            const dateB = new Date(b.lastMessageTimestamp).getTime() || 0;
            return dateB - dateA; // REVERTED: Sort for newest at the top (descending)
        });
      });
      
      console.log('[App.jsx] Calling handleSelectConversation with (after API call):', formattedNewConversation);
      handleSelectConversation(formattedNewConversation);

    } catch (error) {
      console.error('[App.jsx] Error in handleInitiateChatWithContact:', error);
      // TODO: Show user-friendly error
    }
  };

  // ADDED: Handler for ChatWindow to request more messages
  const handleLoadMoreMessages = useCallback((conversationId) => {
    if (!conversationId || isLoadingMessages[conversationId] || !hasMoreMessages[conversationId]) {
      console.log(`[App.jsx] Cannot load more messages for ${conversationId}. Loading: ${isLoadingMessages[conversationId]}, HasMore: ${hasMoreMessages[conversationId]}`);
      return;
    }
    const currentOffset = messageOffsets[conversationId] || 0;
    fetchMessages(conversationId, currentOffset, DEFAULT_MESSAGE_LIMIT, false);
  }, [isLoadingMessages, hasMoreMessages, messageOffsets, fetchMessages]);

  // Log selectedConversation before rendering ChatWindow
  if (selectedConversation) {
    console.log('[App.jsx] Rendering ChatWindow with selectedConversation (name, id, targetId):', 
                  selectedConversation.name, 
                  selectedConversation.id, 
                  selectedConversation.targetId);
  }

  return (
    <>
      {currentUser ? (
        <div className="app-container">
          <aside className="sidebar">
            {/* Pass isLoadingConversations to ConversationList if it should show a loader */}
            <ConversationList 
              conversations={conversations} 
              onSelectConversation={handleSelectConversation}
              selectedConversationId={selectedConversation?.id}
              isLoading={isLoadingConversations} // Pass loading state
              onOpenAddContactModal={openAddContactModal} // Pass handler to ConversationList
              onInitiateChatWithContact={handleInitiateChatWithContact} // Pass the new handler
            />
            <SettingsButton onClick={openSettingsModal} />
          </aside>
          <main className="chat-area">
            {selectedConversation ? (
              <ChatWindow 
                conversation={selectedConversation} 
                messages={messages[selectedConversation.id] || []} 
                onSendMessage={handleSendMessage} 
                onLoadMoreMessages={handleLoadMoreMessages} 
                isLoadingMoreMessages={isLoadingMessages[selectedConversation.id] || false}
                hasMoreMessages={hasMoreMessages[selectedConversation.id] !== undefined ? hasMoreMessages[selectedConversation.id] : true}
              />
            ) : (
              <div className="no-chat-selected">
                {/* Consider a different message if conversations are loading vs. no conversations vs. no selection */}
                <p>{isLoadingConversations ? t('loading_conversations') : (conversations.length === 0 && !selectedConversation ? t('no_conversations_placeholder') : t('select_conversation_placeholder'))}</p>
              </div>
            )}
          </main>
        </div>
      ) : (
        <div className="app-container--auth"> 
          <div className="auth-prompt">
            <h1>{t('welcome_message_placeholder_title')}</h1>
            <p>
              <Trans i18nKey="login_or_register_to_continue_placeholder">
                Please <button type="button" onClick={openSettingsModal} className="auth-prompt__button">login or register</button> to continue.
              </Trans>
            </p>
          </div>
        </div>
      )}
      <SettingsModal 
        isOpen={isSettingsModalOpen} 
        onClose={closeSettingsModal} 
      />
      <AddContactModal 
        isOpen={isAddContactModalOpen} 
        onClose={closeAddContactModal} 
      /> {/* ADDED new modal instance */}
    </>
  );
}

export default App;
