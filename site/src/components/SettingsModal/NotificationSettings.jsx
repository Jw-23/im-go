import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import useNotification from '../../hooks/useNotification';

const NotificationSettings = () => {
  const { t } = useTranslation();
  const { 
    isSupported, 
    isReady, 
    permission, 
    requestPermission 
  } = useNotification();
  
  const [notificationStatus, setNotificationStatus] = useState('');

  // 初始化状态
  useEffect(() => {
    updateNotificationStatus();
  }, [isSupported, isReady, permission]);

  // 更新通知状态信息
  const updateNotificationStatus = () => {
    if (!isSupported) {
      setNotificationStatus(t('notifications.unsupported'));
      return;
    }

    switch (permission) {
      case 'granted':
        setNotificationStatus(t('notifications.enabled'));
        break;
      case 'denied':
        setNotificationStatus(t('notifications.blocked'));
        break;
      default:
        setNotificationStatus(t('notifications.notEnabled'));
        break;
    }
  };

  // 请求通知权限
  const handleEnableNotifications = async () => {
    try {
      const result = await requestPermission();
      updateNotificationStatus();
      
      if (result === 'granted') {
        // 显示测试通知
        showTestNotification();
      }
    } catch (error) {
      console.error('请求通知权限失败:', error);
    }
  };

  // 显示测试通知
  const showTestNotification = () => {
    if (!('Notification' in window)) return;
    
    if (Notification.permission === 'granted') {
      new Notification(t('notifications.testTitle'), {
        body: t('notifications.testBody'),
        icon: '/favicon.ico'
      });
    }
  };

  return (
    <div className="space-y-4">
      <h3 className="text-lg font-medium">{t('settings.notifications')}</h3>
      
      <div className="p-4 border rounded-md bg-gray-50 dark:bg-gray-800 dark:border-gray-700">
        <div className="flex justify-between items-center">
          <div>
            <p className="font-medium">{t('notifications.pushNotifications')}</p>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {notificationStatus}
            </p>
          </div>
          
          <div>
            {permission === 'granted' ? (
              <button 
                className="px-3 py-1 rounded-md bg-green-100 text-green-800 dark:bg-green-800 dark:text-green-100 cursor-default"
                disabled
              >
                {t('notifications.enabled')}
              </button>
            ) : permission === 'denied' ? (
              <button
                className="px-3 py-1 rounded-md bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200 cursor-help"
                onClick={() => alert(t('notifications.enableInstructions'))}
              >
                {t('notifications.blockedByBrowser')}
              </button>
            ) : (
              <button
                className="px-3 py-1 rounded-md bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-700 dark:hover:bg-blue-800"
                onClick={handleEnableNotifications}
                disabled={!isSupported}
              >
                {t('notifications.enable')}
              </button>
            )}
          </div>
        </div>
        
        {!isSupported && (
          <div className="mt-3 text-sm text-amber-600 dark:text-amber-400">
            {t('notifications.browserNotSupported')}
          </div>
        )}
        
        {permission === 'denied' && (
          <div className="mt-3 text-sm text-amber-600 dark:text-amber-400">
            {t('notifications.permissionDenied')}
          </div>
        )}
      </div>
      
      {permission === 'granted' && (
        <div className="flex justify-end">
          <button
            className="px-3 py-1 text-sm text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300"
            onClick={showTestNotification}
          >
            {t('notifications.sendTest')}
          </button>
        </div>
      )}
    </div>
  );
};

export default NotificationSettings; 