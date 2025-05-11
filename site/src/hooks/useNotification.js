import { useState, useEffect, useCallback } from 'react';
import { 
  initNotificationService, 
  getNotificationPermission, 
  requestNotificationPermission, 
  sendLocalNotification,
  sendNotificationViaServiceWorker,
  isPushNotificationReady
} from '../services/notificationService';

/**
 * 通知管理Hook
 * @returns {Object} 通知相关方法和状态
 */
const useNotification = () => {
  // 状态管理
  const [permission, setPermission] = useState(getNotificationPermission());
  const [isSupported, setIsSupported] = useState(false);
  const [isReady, setIsReady] = useState(false);

  // 初始化通知服务
  useEffect(() => {
    const initialize = async () => {
      try {
        // 检查浏览器支持
        const notificationSupported = 'Notification' in window;
        const serviceWorkerSupported = 'serviceWorker' in navigator;
        const isPushSupported = 'PushManager' in window;
        
        setIsSupported(notificationSupported && serviceWorkerSupported && isPushSupported);
        
        if (notificationSupported && serviceWorkerSupported) {
          // 初始化服务
          const isReady = await initNotificationService();
          setIsReady(isReady);
          
          // 更新权限状态
          setPermission(getNotificationPermission());
        }
      } catch (error) {
        console.error('初始化通知服务失败:', error);
      }
    };
    
    initialize();
  }, []);

  // 请求用户授权
  const requestPermission = useCallback(async () => {
    try {
      const newPermission = await requestNotificationPermission();
      setPermission(newPermission);
      setIsReady(isPushNotificationReady());
      return newPermission;
    } catch (error) {
      console.error('请求通知权限失败:', error);
      return 'denied';
    }
  }, []);

  // 发送本地通知
  const showNotification = useCallback(async (title, options = {}) => {
    if (!isReady && permission !== 'granted') {
      await requestPermission();
    }
    
    if (getNotificationPermission() !== 'granted') {
      console.warn('通知权限未授予，无法显示通知');
      return false;
    }
    
    try {
      const result = await sendLocalNotification(title, options);
      return result;
    } catch (error) {
      console.error('发送通知失败:', error);
      return false;
    }
  }, [isReady, permission, requestPermission]);

  // 发送通知（即使页面已关闭）
  const sendPushNotification = useCallback(async (notificationData) => {
    if (!isReady && permission !== 'granted') {
      await requestPermission();
    }
    
    if (getNotificationPermission() !== 'granted') {
      console.warn('通知权限未授予，无法发送推送通知');
      return false;
    }
    
    try {
      return await sendNotificationViaServiceWorker(notificationData);
    } catch (error) {
      console.error('发送推送通知失败:', error);
      return false;
    }
  }, [isReady, permission, requestPermission]);

  return {
    isSupported,        // 浏览器是否支持通知
    isReady,            // 通知服务是否已准备就绪
    permission,         // 当前通知权限状态
    requestPermission,  // 请求通知权限
    showNotification,   // 显示本地通知
    sendPushNotification // 发送推送通知（即使页面已关闭）
  };
};

export default useNotification; 