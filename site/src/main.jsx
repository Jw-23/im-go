import React, { Suspense } from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import './index.css';
import './i18n'; // Import i18n configuration
import { ThemeProvider } from './contexts/ThemeContext';
import { AuthProvider } from './contexts/AuthContext';
import { initNotificationService } from './services/notificationService';

// 初始化通知服务
initNotificationService().then(isReady => {
  console.log('通知服务初始化完成，状态:', isReady ? '已就绪' : '未就绪');
}).catch(error => {
  console.error('通知服务初始化失败:', error);
});

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <Suspense fallback="Loading...">
      <AuthProvider>
        <ThemeProvider>
          <App />
        </ThemeProvider>
      </AuthProvider>
    </Suspense>
  </React.StrictMode>
); 