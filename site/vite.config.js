import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // 代理所有以 /api/v1 开头的请求
      '/api/v1': {
        target: 'http://127.0.0.1:8081', // 目标是您的后端 API 服务器地址
        changeOrigin: true, // 需要改变源，通常设置为 true
        // secure: false,
        // rewrite: (path) => path.replace(/^\/api\/v1/, ''), 
      },
      // 代理所有以 /auth 开头的请求 (用于登录/注册)
      '/auth': {
         target: 'http://127.0.0.1:8081',
         changeOrigin: true,
      },
       // 如果有 /uploads 路径需要代理（例如头像显示）
       '/uploads': {
          target: 'http://127.0.0.1:8081', // API 服务器现在也处理静态文件
          changeOrigin: true,
       }
    }
  }
}); 