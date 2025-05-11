import React, { useState, useEffect } from 'react';
import './GroupInfoModal.css';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../../contexts/AuthContext';

const GroupInfoModal = ({ 
  isOpen, 
  onClose, 
  groupId,
  groupInfo,
  contacts = [],
  onAddMembers,
  onRemoveMember,
  onLeaveGroup,
  onUpdateGroupName,
  onFixGroupConversation
}) => {
  const { t } = useTranslation();
  const { currentUser } = useAuth();
  const [activeTab, setActiveTab] = useState('info');
  const [groupName, setGroupName] = useState('');
  const [isEditing, setIsEditing] = useState(false);
  const [error, setError] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedContacts, setSelectedContacts] = useState([]);

  // 判断当前用户是否为群主
  const isGroupOwner = groupInfo?.ownerId === currentUser?.id;

  useEffect(() => {
    if (groupInfo?.name) {
      setGroupName(groupInfo.name);
    }
  }, [groupInfo]);

  if (!isOpen || !groupInfo) return null;

  // 当前群成员ID列表
  const currentMemberIds = groupInfo.members?.map(member => member.id) || [];
  
  // 过滤出可添加的联系人（不在群内的联系人）
  const availableContacts = contacts.filter(contact => 
    !currentMemberIds.includes(contact.id) && 
    contact.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // 处理更新群名
  const handleUpdateGroupName = () => {
    if (!groupName.trim()) {
      setError('群聊名称不能为空');
      return;
    }
    
    onUpdateGroupName(groupId, groupName.trim());
    setIsEditing(false);
    setError('');
  };

  // 选择/取消选择联系人
  const toggleContact = (contactId) => {
    setSelectedContacts(prev => {
      if (prev.includes(contactId)) {
        return prev.filter(id => id !== contactId);
      } else {
        return [...prev, contactId];
      }
    });
  };

  // 添加成员
  const handleAddMembers = () => {
    if (selectedContacts.length === 0) {
      setError('请选择至少一个联系人');
      return;
    }
    
    onAddMembers(groupId, selectedContacts);
    setSelectedContacts([]);
    setSearchQuery('');
    setError('');
    setActiveTab('members');
  };

  return (
    <div className="modal-overlay">
      <div className="group-info-modal">
        <div className="modal-header">
          <h2>群聊信息</h2>
          <button className="close-button" onClick={onClose}>&times;</button>
        </div>
        
        <div className="modal-tabs">
          <button 
            className={activeTab === 'info' ? 'active' : ''}
            onClick={() => setActiveTab('info')}
          >
            基本信息
          </button>
          <button 
            className={activeTab === 'members' ? 'active' : ''}
            onClick={() => setActiveTab('members')}
          >
            成员管理
          </button>
          <button 
            className={activeTab === 'add' ? 'active' : ''}
            onClick={() => setActiveTab('add')}
          >
            添加成员
          </button>
        </div>
        
        <div className="modal-content">
          {activeTab === 'info' && (
            <div className="group-basic-info">
              <div className="group-avatar">
                <img 
                  src={groupInfo.avatar || `https://ui-avatars.com/api/?name=${encodeURIComponent(groupInfo.name.charAt(0))}&size=80&background=random&color=fff`}
                  alt={`${groupInfo.name} avatar`}
                />
              </div>
              
              <div className="group-name-section">
                {isEditing ? (
                  <div className="edit-name-form">
                    <input
                      type="text"
                      value={groupName}
                      onChange={(e) => setGroupName(e.target.value)}
                      className="form-control"
                    />
                    {error && <div className="error-message">{error}</div>}
                    <div className="button-group">
                      <button 
                        className="cancel-button"
                        onClick={() => {
                          setIsEditing(false);
                          setGroupName(groupInfo.name || '');
                          setError('');
                        }}
                      >
                        取消
                      </button>
                      <button 
                        className="save-button"
                        onClick={handleUpdateGroupName}
                      >
                        保存
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="group-name-display">
                    <h3>{groupInfo.name}</h3>
                    {isGroupOwner && (
                      <button 
                        className="edit-button"
                        onClick={() => setIsEditing(true)}
                      >
                        编辑
                      </button>
                    )}
                  </div>
                )}
              </div>
              
              <div className="group-stats">
                <div className="stat-item">
                  <span className="stat-label">创建时间</span>
                  <span className="stat-value">
                    {new Date(groupInfo.createdAt).toLocaleDateString()}
                  </span>
                </div>
                <div className="stat-item">
                  <span className="stat-label">成员数量</span>
                  <span className="stat-value">{groupInfo.members?.length || 0}</span>
                </div>
                <div className="stat-item">
                  <span className="stat-label">群主</span>
                  <span className="stat-value">
                    {groupInfo.owner?.name || '未知'}
                  </span>
                </div>
              </div>
              
              {isGroupOwner && (
                <div className="fix-group-conversation">
                  <button 
                    className="fix-button"
                    onClick={() => {
                      if (window.confirm('是否要修复群组会话？如果你在群聊中无法收到消息，此操作可能会有所帮助。')) {
                        onFixGroupConversation && onFixGroupConversation(groupId);
                      }
                    }}
                  >
                    修复群组消息接收问题
                  </button>
                </div>
              )}
              
              <div className="leave-group">
                <button 
                  className="leave-button"
                  onClick={() => {
                    if (window.confirm('确定要退出该群聊吗？')) {
                      onLeaveGroup(groupId);
                      onClose();
                    }
                  }}
                >
                  退出群聊
                </button>
              </div>
            </div>
          )}
          
          {activeTab === 'members' && (
            <div className="group-members">
              <div className="members-count">
                共 {groupInfo.members?.length || 0} 位成员
              </div>
              
              <div className="members-list">
                {groupInfo.members?.length > 0 ? (
                  groupInfo.members.map(member => (
                    <div key={member.id} className="member-item">
                      <img 
                        src={member.avatar || `https://ui-avatars.com/api/?name=${encodeURIComponent(member.name.charAt(0))}&size=40&background=random&color=fff`}
                        alt={`${member.name} avatar`}
                        className="member-avatar"
                      />
                      <div className="member-info">
                        <div className="member-name">
                          {member.name}
                          {member.id === groupInfo.ownerId && (
                            <span className="owner-badge">群主</span>
                          )}
                          {member.id === currentUser?.id && (
                            <span className="self-badge">我</span>
                          )}
                        </div>
                      </div>
                      {isGroupOwner && member.id !== currentUser?.id && (
                        <button 
                          className="remove-button"
                          onClick={() => {
                            if (window.confirm(`确定要将 ${member.name} 移出群聊吗？`)) {
                              onRemoveMember(groupId, member.id);
                            }
                          }}
                        >
                          移出
                        </button>
                      )}
                    </div>
                  ))
                ) : (
                  <div className="no-members">暂无成员信息</div>
                )}
              </div>
            </div>
          )}
          
          {activeTab === 'add' && (
            <div className="add-members">
              <div className="search-section">
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="搜索联系人..."
                  className="form-control"
                />
                <div className="selected-count">
                  已选择 {selectedContacts.length} 人
                </div>
              </div>
              
              <div className="contacts-list">
                {availableContacts.length === 0 ? (
                  <div className="no-contacts">
                    {searchQuery ? '没有找到匹配的联系人' : '没有可添加的联系人'}
                  </div>
                ) : (
                  availableContacts.map(contact => (
                    <div 
                      key={contact.id}
                      className={`contact-item ${selectedContacts.includes(contact.id) ? 'selected' : ''}`}
                      onClick={() => toggleContact(contact.id)}
                    >
                      <div className="checkbox">
                        {selectedContacts.includes(contact.id) && <span>✓</span>}
                      </div>
                      <img 
                        src={contact.avatar || `https://ui-avatars.com/api/?name=${encodeURIComponent(contact.name.charAt(0))}&size=40&background=random&color=fff`}
                        alt={`${contact.name} avatar`} 
                        className="contact-avatar"
                      />
                      <div className="contact-name">{contact.name}</div>
                    </div>
                  ))
                )}
              </div>
              
              {error && <div className="error-message">{error}</div>}
              
              <div className="add-members-footer">
                <button 
                  className="add-button"
                  disabled={selectedContacts.length === 0}
                  onClick={handleAddMembers}
                >
                  添加所选成员
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default GroupInfoModal; 