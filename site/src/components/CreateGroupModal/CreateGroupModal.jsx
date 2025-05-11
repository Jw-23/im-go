import React, { useState, useRef } from 'react';
import './CreateGroupModal.css';
import { useTranslation } from 'react-i18next';

const CreateGroupModal = ({ isOpen, onClose, contacts = [], onCreateGroup }) => {
  const { t } = useTranslation();
  const [groupName, setGroupName] = useState('');
  const [groupDescription, setGroupDescription] = useState('');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedContacts, setSelectedContacts] = useState([]);
  const [error, setError] = useState('');
  const [isPublic, setIsPublic] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const submitTimeoutRef = useRef(null);

  if (!isOpen) return null;

  // 过滤联系人
  const filteredContacts = contacts.filter(contact => 
    (contact.name || contact.username || '').toLowerCase().includes(searchQuery.toLowerCase())
  );

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

  // 提交表单创建群聊
  const handleSubmit = (e) => {
    e.preventDefault();
    
    // 防止重复提交
    if (isSubmitting) {
      console.log('请求已在进行中，防止重复提交');
      return;
    }
    
    if (!groupName.trim()) {
      setError('请输入群聊名称');
      return;
    }
    
    if (selectedContacts.length < 1) {
      setError('请至少选择一个联系人');
      return;
    }
    
    console.log('原始selectedContacts:', selectedContacts, '类型:', selectedContacts.map(id => typeof id));
    
    // 确保成员ID是数字类型
    const memberIds = selectedContacts.map(id => {
      console.log('处理成员ID:', id, '类型:', typeof id);
      // 如果id已经是数字，直接返回；否则尝试转换
      const numericId = typeof id === 'number' ? id : parseInt(id, 10);
      return isNaN(numericId) ? null : numericId;
    }).filter(id => id !== null && id > 0); // 过滤掉转换失败的ID和无效ID
    
    console.log('创建群组，成员IDs:', memberIds, '类型:', memberIds.map(id => typeof id));
    
    if (memberIds.length === 0) {
      setError('没有有效的联系人被选择');
      return;
    }
    
    const groupData = {
      name: groupName.trim(),
      description: groupDescription.trim(),
      memberIds: memberIds,
      isPublic: isPublic
    };
    
    console.log('发送创建群组请求:', groupData);
    
    // 设置提交状态
    setIsSubmitting(true);
    setError('');
    
    // 调用创建群组函数
    onCreateGroup(groupData);
    
    // 防抖：3秒内不允许再次提交
    clearTimeout(submitTimeoutRef.current);
    submitTimeoutRef.current = setTimeout(() => {
      setIsSubmitting(false);
    }, 3000);
    
    // 重置表单
    resetForm();
  };
  
  // 重置表单
  const resetForm = () => {
    setGroupName('');
    setGroupDescription('');
    setSearchQuery('');
    setSelectedContacts([]);
    setIsPublic(false);
    setError('');
    // 不重置 isSubmitting，由超时器控制
  };
  
  // 处理关闭模态框
  const handleClose = () => {
    // 如果正在提交，不允许关闭
    if (isSubmitting) return;
    
    resetForm();
    setIsSubmitting(false);
    clearTimeout(submitTimeoutRef.current);
    onClose();
  };

  return (
    <div className="modal-overlay">
      <div className="create-group-modal">
        <div className="modal-header">
          <h2>创建群聊</h2>
          <button className="close-button" onClick={handleClose}>&times;</button>
        </div>
        
        <form onSubmit={handleSubmit}>
          <div className="form-group">
            <label htmlFor="group-name">群聊名称 <span className="required">*</span></label>
            <input
              type="text"
              id="group-name"
              value={groupName}
              onChange={(e) => setGroupName(e.target.value)}
              placeholder="请输入群聊名称"
              className="form-control"
            />
          </div>
          
          <div className="form-group">
            <label htmlFor="group-description">群聊描述</label>
            <textarea
              id="group-description"
              value={groupDescription}
              onChange={(e) => setGroupDescription(e.target.value)}
              placeholder="请输入群聊描述（可选）"
              className="form-control textarea"
              rows="3"
            />
          </div>
          
          <div className="form-group checkbox-group">
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={isPublic}
                onChange={(e) => setIsPublic(e.target.checked)}
              />
              设置为公开群组
            </label>
            <small className="help-text">公开群组允许被搜索并可以被非好友加入</small>
          </div>
          
          <div className="form-group">
            <label>选择联系人 <span className="required">*</span></label>
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="搜索联系人..."
              className="form-control search-input"
            />
            
            <div className="selected-count">
              已选择 {selectedContacts.length} 人
            </div>
            
            <div className="contacts-list">
              {filteredContacts.length === 0 ? (
                <div className="no-contacts">
                  {searchQuery ? '没有找到匹配的联系人' : '暂无联系人'}
                </div>
              ) : (
                filteredContacts.map(contact => (
                  <div 
                    key={contact.id}
                    className={`contact-item ${selectedContacts.includes(contact.id) ? 'selected' : ''}`}
                    onClick={() => toggleContact(contact.id)}
                  >
                    <div className="checkbox">
                      {selectedContacts.includes(contact.id) && <span>✓</span>}
                    </div>
                    <img 
                      src={contact.avatar || `https://ui-avatars.com/api/?name=${encodeURIComponent((contact.name || contact.username || 'User').charAt(0))}&size=40&background=random&color=fff`}
                      alt={`${contact.name || contact.username || 'User'} avatar`} 
                      className="contact-avatar"
                    />
                    <div className="contact-name">{contact.name || contact.username || `用户${contact.id}`}</div>
                  </div>
                ))
              )}
            </div>
          </div>
          
          {error && <div className="error-message">{error}</div>}
          {isSubmitting && <div className="info-message">正在创建群组，请稍候...</div>}
          
          <div className="modal-footer">
            <button type="button" className="cancel-button" onClick={handleClose} disabled={isSubmitting}>
              取消
            </button>
            <button 
              type="submit" 
              className="create-button"
              disabled={isSubmitting || selectedContacts.length < 1 || !groupName.trim()}
            >
              {isSubmitting ? '创建中...' : '创建群聊'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateGroupModal; 