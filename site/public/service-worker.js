// 缓存版本，方便更新Service Worker
const CACHE_VERSION = 'v1';
const CACHE_NAME = `im-go-cache-${CACHE_VERSION}`;

// 安装Service Worker
self.addEventListener('install', (event) => {
  console.log('Service Worker 安装中...');
  
  // 预缓存核心资源
  event.waitUntil(
    fetch('/sw-assets.json')
      .then(response => response.json())
      .then(data => {
        const cachePaths = data.assets || [];
        
        return caches.open(CACHE_NAME)
          .then(cache => {
            console.log('Service Worker: 缓存核心资源');
            return cache.addAll(cachePaths);
          });
      })
      .catch(error => {
        console.error('Service Worker: 预缓存资源失败', error);
      })
  );
  
  self.skipWaiting();
});

// 激活Service Worker
self.addEventListener('activate', (event) => {
  console.log('Service Worker 已激活');
  // 清理旧版本缓存
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME) {
            console.log('Service Worker: 删除旧缓存', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    })
  );
  return self.clients.claim();
});

// 监听fetch事件，实现离线缓存
self.addEventListener('fetch', (event) => {
  // 只处理GET请求
  if (event.request.method !== 'GET') return;
  
  // 忽略Chrome扩展请求
  if (event.request.url.startsWith('chrome-extension://')) return;
  
  // 优先使用网络请求，失败时使用缓存
  event.respondWith(
    fetch(event.request)
      .then(response => {
        // 如果获取到了有效响应，更新缓存
        if (response && response.status === 200) {
          const responseClone = response.clone();
          caches.open(CACHE_NAME).then(cache => {
            cache.put(event.request, responseClone);
          });
        }
        return response;
      })
      .catch(() => {
        // 网络请求失败，尝试从缓存获取
        return caches.match(event.request);
      })
  );
});

// 监听push事件
self.addEventListener('push', (event) => {
  console.log('收到推送消息', event);

  let notificationData = {};
  try {
    notificationData = event.data.json();
  } catch (e) {
    notificationData = {
      title: '新消息',
      body: event.data ? event.data.text() : '收到一条新消息',
      data: { url: '/' }
    };
  }

  const title = notificationData.title || '新消息';
  const options = {
    body: notificationData.body || '您收到一条新消息',
    icon: '/favicon.ico',
    badge: '/favicon.ico',
    data: notificationData.data || { url: '/' },
    tag: notificationData.tag || 'default', // 使用tag避免多条通知
    requireInteraction: true // 需要用户交互才会消失
  };

  event.waitUntil(
    self.registration.showNotification(title, options)
  );
});

// 处理通知点击事件
self.addEventListener('notificationclick', (event) => {
  console.log('点击了通知', event);
  event.notification.close();

  // 打开对应的页面或对话
  const urlToOpen = event.notification.data?.url || '/';

  event.waitUntil(
    clients.matchAll({ type: 'window' }).then((clientsList) => {
      // 查找是否有已打开的窗口
      const hadWindowToFocus = clientsList.some((client) => {
        if (client.url === urlToOpen && 'focus' in client) {
          client.focus();
          return true;
        }
        return false;
      });

      // 如果没有已打开的窗口，则打开新窗口
      if (!hadWindowToFocus && clients.openWindow) {
        return clients.openWindow(urlToOpen);
      }
    })
  );
});

// 接收消息事件处理
self.addEventListener('message', (event) => {
  console.log('Service Worker 收到消息:', event.data);
  
  // 处理从主页面发送的消息
  if (event.data && event.data.type === 'MANUAL_NOTIFICATION') {
    const { title, body, data } = event.data;
    
    self.registration.showNotification(title, {
      body,
      icon: '/favicon.ico',
      badge: '/favicon.ico',
      data: data || { url: '/' },
      requireInteraction: true
    });
  }
}); 