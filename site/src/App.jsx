import React, { useState, useEffect } from 'react';
import './App.css';
import { ChatLayout, AuthPrompt } from './components/Layout';
import SettingsModal from './components/SettingsModal/SettingsModal';
import AddContactModal from './components/AddContactModal/AddContactModal';
import CreateGroupModal from './components/CreateGroupModal/CreateGroupModal';
import GroupInfoModal from './components/GroupInfoModal/GroupInfoModal';
import { useAuth } from './contexts/AuthContext';
import { useTranslation } from 'react-i18next';
import { useChatManager } from './hooks/useChatManager';
import { getFriendsList, fixGroupConversationParticipants } from './services/api';

function App() {
  // 状态
  const [isSettingsModalOpen, setIsSettingsModalOpen] = useState(false);
  const [isAddContactModalOpen, setIsAddContactModalOpen] = useState(false);
  const [isCreateGroupModalOpen, setIsCreateGroupModalOpen] = useState(false);
  const [isGroupInfoModalOpen, setIsGroupInfoModalOpen] = useState(false);
  const [selectedGroupId, setSelectedGroupId] = useState(null);
  const [contacts, setContacts] = useState([]);
  const [isLoadingContacts, setIsLoadingContacts] = useState(false);
  
  // Hooks
  const { token, currentUser, isLoading: isAuthLoading } = useAuth();
  const { t } = useTranslation();
  const {
    // 状态
    conversations,
    selectedConversation,
    messages,
    isLoadingConversations,
    isLoadingMessages,
    hasMoreMessages,
    websocketConnected,
    
    // 动作
    sendMessage,
    loadMoreMessages,
    selectConversation,
    initiateChatWithContact,
    createGroup
  } = useChatManager();

  // 如果未登录并且认证检查完成，自动打开设置模态框
  useEffect(() => {
    if (!isAuthLoading && !currentUser && !isSettingsModalOpen) {
      setIsSettingsModalOpen(true);
    }
  }, [isAuthLoading, currentUser, isSettingsModalOpen]);

  // 获取联系人列表
  useEffect(() => {
    const fetchContacts = async () => {
      if (!currentUser) return;
      
      setIsLoadingContacts(true);
    try {
        const result = await getFriendsList();
        if (result.success) {
          setContacts(result.data || []);
        } else {
          console.error('获取联系人列表失败:', result.error);
      }
      } catch (error) {
        console.error('获取联系人列表出错:', error);
    } finally {
        setIsLoadingContacts(false);
      }
    };

    fetchContacts();
  }, [currentUser]);

  // Modal 控制
  const openSettingsModal = () => setIsSettingsModalOpen(true);
  const closeSettingsModal = () => setIsSettingsModalOpen(false);
  const openAddContactModal = () => setIsAddContactModalOpen(true);
  const closeAddContactModal = () => setIsAddContactModalOpen(false);
  const openCreateGroupModal = () => setIsCreateGroupModalOpen(true);
  const closeCreateGroupModal = () => setIsCreateGroupModalOpen(false);
  
  // 群聊管理
  const openGroupInfoModal = (groupId) => {
    setSelectedGroupId(groupId);
    setIsGroupInfoModalOpen(true);
  };
  const closeGroupInfoModal = () => setIsGroupInfoModalOpen(false);
  
  // 创建群聊
  const handleCreateGroup = async (groupData) => {
    if (!currentUser) {
      console.error('未登录，无法创建群组');
      return;
    }
    
    try {
      // 使用useChatManager中的createGroup功能
      const result = await createGroup(groupData);

      if (!result.success) {
        console.error('创建群聊失败:', result.error);
        // 这里可以添加错误通知
        return;
      }
      
      console.log('群聊创建成功:', result.data);
      // 关闭创建群聊模态框
      setIsCreateGroupModalOpen(false);

    } catch (error) {
      console.error('创建群聊过程中发生错误:', error);
      // 这里可以添加错误通知
    }
  };

  // 群成员管理
  const handleAddGroupMembers = (groupId, memberIds) => {
    console.log('添加群成员:', groupId, memberIds);
    // 添加成员逻辑
  };
  
  const handleRemoveGroupMember = (groupId, memberId) => {
    console.log('移除群成员:', groupId, memberId);
    // 移除成员逻辑
  };
  
  const handleLeaveGroup = (groupId) => {
    console.log('退出群聊:', groupId);
    // 退出群聊逻辑
  };
  
  const handleUpdateGroupName = (groupId, newName) => {
    console.log('更新群名称:', groupId, newName);
    // 更新群名称逻辑
  };

  // 修复群组会话参与者
  const handleFixGroupConversation = async (groupId) => {
    try {
      console.log('修复群组会话:', groupId);
      const result = await fixGroupConversationParticipants(groupId);
      
      if (result.success) {
        console.log('修复成功:', result.data);
        alert('群组会话修复成功，现在可以正常接收群消息了！');
        // 可能需要刷新会话列表
      } else {
        console.error('修复失败:', result.error);
        alert('修复失败: ' + (result.error || '未知错误'));
      }
    } catch (error) {
      console.error('修复过程中发生错误:', error);
      alert('修复过程中发生错误，请稍后重试');
    }
  };

  return (
    <>
      {currentUser ? (
        <ChatLayout 
          // 状态属性
              conversations={conversations} 
          selectedConversation={selectedConversation}
          messages={messages}
          isLoadingConversations={isLoadingConversations}
          isLoadingMessages={isLoadingMessages}
          hasMoreMessages={hasMoreMessages}
          websocketConnected={websocketConnected}
          contacts={contacts}
          isLoadingContacts={isLoadingContacts}
          
          // 事件处理函数
          onSelectConversation={selectConversation}
          onSendMessage={sendMessage}
          onLoadMoreMessages={loadMoreMessages}
          onOpenAddContactModal={openAddContactModal}
          onInitiateChatWithContact={initiateChatWithContact}
          onOpenSettings={openSettingsModal}
          onOpenCreateGroupModal={openCreateGroupModal}
          onManageGroupMembers={openGroupInfoModal}
        />
      ) : (
        <AuthPrompt 
          t={t} 
          onOpenSettings={openSettingsModal} 
        />
      )}
      
      <SettingsModal 
        isOpen={isSettingsModalOpen} 
        onClose={closeSettingsModal} 
      />
      
      <AddContactModal 
        isOpen={isAddContactModalOpen} 
        onClose={closeAddContactModal} 
      />
      
      <CreateGroupModal
        isOpen={isCreateGroupModalOpen}
        onClose={closeCreateGroupModal}
        contacts={contacts}
        onCreateGroup={handleCreateGroup}
      />
      
      <GroupInfoModal
        isOpen={isGroupInfoModalOpen}
        onClose={closeGroupInfoModal}
        groupId={selectedGroupId}
        groupInfo={selectedConversation?.type === 'group' && selectedConversation?.id === selectedGroupId ? selectedConversation : null}
        contacts={contacts}
        onAddMembers={handleAddGroupMembers}
        onRemoveMember={handleRemoveGroupMember}
        onLeaveGroup={handleLeaveGroup}
        onUpdateGroupName={handleUpdateGroupName}
        onFixGroupConversation={handleFixGroupConversation}
      />
    </>
  );
}

export default App;
