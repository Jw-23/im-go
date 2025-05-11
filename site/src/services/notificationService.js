/**
 * 通知服务，用于管理推送通知功能
 */

// 检查浏览器是否支持Service Worker
const isServiceWorkerSupported = 'serviceWorker' in navigator;
// 检查浏览器是否支持通知功能
const isNotificationSupported = 'Notification' in window;
// 检查浏览器是否支持Push API
const isPushSupported = 'PushManager' in window;

// 存储serviceWorkerRegistration
let swRegistration = null;

/**
 * 注册Service Worker
 * @returns {Promise<ServiceWorkerRegistration|null>}
 */
export const registerServiceWorker = async () => {
  if (!isServiceWorkerSupported) {
    console.warn('此浏览器不支持Service Worker，无法使用推送通知功能');
    return null;
  }

  try {
    swRegistration = await navigator.serviceWorker.register('/service-worker.js');
    console.log('Service Worker注册成功:', swRegistration);
    return swRegistration;
  } catch (error) {
    console.error('Service Worker注册失败:', error);
    return null;
  }
};

/**
 * 请求通知权限
 * @returns {Promise<string>} 权限状态: 'granted', 'denied', 'default'
 */
export const requestNotificationPermission = async () => {
  if (!isNotificationSupported) {
    console.warn('此浏览器不支持通知功能');
    return 'denied';
  }

  try {
    const permission = await Notification.requestPermission();
    console.log('通知权限状态:', permission);
    return permission;
  } catch (error) {
    console.error('请求通知权限失败:', error);
    return 'denied';
  }
};

/**
 * 获取当前的通知权限状态
 * @returns {string} 权限状态: 'granted', 'denied', 'default'
 */
export const getNotificationPermission = () => {
  if (!isNotificationSupported) return 'denied';
  return Notification.permission;
};

/**
 * 检查是否已经具备推送通知的所有条件
 * @returns {boolean}
 */
export const isPushNotificationReady = () => {
  return isServiceWorkerSupported && 
         isNotificationSupported && 
         isPushSupported &&
         getNotificationPermission() === 'granted' &&
         swRegistration !== null;
};

/**
 * 发送本地通知（当页面在后台但浏览器仍在运行时）
 * @param {string} title 通知标题
 * @param {object} options 通知选项
 * @returns {Promise<boolean>} 是否成功发送
 */
export const sendLocalNotification = async (title, options = {}) => {
  if (!swRegistration) {
    const reg = await registerServiceWorker();
    if (!reg) return false;
  }

  try {
    await swRegistration.showNotification(title, {
      body: options.body || '新消息',
      icon: options.icon || '/favicon.ico',
      badge: options.badge || '/favicon.ico',
      data: options.data || {},
      requireInteraction: options.requireInteraction !== false,
      tag: options.tag || 'default'
    });
    return true;
  } catch (error) {
    console.error('发送本地通知失败:', error);
    return false;
  }
};

/**
 * 通过Service Worker发送通知
 * 即使页面关闭，Service Worker也可以显示通知
 * @param {object} notificationData 通知数据
 * @returns {Promise<boolean>} 是否成功发送
 */
export const sendNotificationViaServiceWorker = async (notificationData) => {
  if (!isServiceWorkerSupported) return false;
  
  // 尝试获取serviceWorker
  if (!swRegistration) {
    await registerServiceWorker();
  }

  // 确保Service Worker已激活
  if (!navigator.serviceWorker.controller) {
    console.warn('等待Service Worker激活...');
    await new Promise(resolve => {
      navigator.serviceWorker.addEventListener('controllerchange', () => {
        resolve();
      });
    });
  }

  try {
    navigator.serviceWorker.controller.postMessage({
      type: 'MANUAL_NOTIFICATION',
      title: notificationData.title || '新消息',
      body: notificationData.body || '您收到一条新消息',
      data: notificationData.data || {}
    });
    return true;
  } catch (error) {
    console.error('发送通知到Service Worker失败:', error);
    return false;
  }
};

/**
 * 初始化通知服务
 * 在应用启动时调用
 */
export const initNotificationService = async () => {
  // 注册Service Worker
  const reg = await registerServiceWorker();
  
  // 如果注册成功，请求通知权限
  if (reg) {
    await requestNotificationPermission();
  }
  
  return isPushNotificationReady();
}; 