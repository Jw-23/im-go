import { useState, useEffect, useCallback, useRef } from 'react';
import { useAuth } from '../contexts/AuthContext';

const CHAT_SERVER_URL = 'ws://124.71.77.122:8080'; // 从 App.jsx 迁移
const WEBSOCKET_PATH = '/ws/chat';             // 从 App.jsx 迁移

// 重连配置
const MAX_RECONNECT_ATTEMPTS = 10;  // 最大重连次数
const BASE_RECONNECT_DELAY = 1000;  // 初始重连延迟（毫秒）
const MAX_RECONNECT_DELAY = 30000;  // 最大重连延迟（毫秒）

function useWebSocket() {
  const [websocket, setWebsocket] = useState(null);
  const [lastMessage, setLastMessage] = useState(null); // 用于接收最新消息
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState(null); // 用于存储 WebSocket 错误信息
  const { token, currentUser } = useAuth();
  
  // 使用ref保存重连状态，避免闭包问题
  const reconnectAttemptsRef = useRef(0);
  const reconnectTimeoutRef = useRef(null);
  const manualCloseRef = useRef(false);

  // 创建WebSocket连接的函数
  const connectWebSocket = useCallback(() => {
    if (!token || !currentUser) return null;

    const wsUrl = `${CHAT_SERVER_URL}${WEBSOCKET_PATH}?token=${token}`;
    console.log('[useWebSocket] 连接WebSocket. URL:', wsUrl, '尝试次数:', reconnectAttemptsRef.current);
    
    const wsInstance = new WebSocket(wsUrl);

    wsInstance.onopen = () => {
      console.log('[useWebSocket] WebSocket连接已建立. ReadyState:', wsInstance.readyState);
      setWebsocket(wsInstance);
      setIsConnected(true);
      setError(null);
      reconnectAttemptsRef.current = 0; // 重置重连尝试次数
    };

    wsInstance.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data);
        console.log('[useWebSocket] 收到WebSocket消息:', message);
        setLastMessage(message); // 更新收到的最新消息
      } catch (e) {
        console.error('[useWebSocket] 解析WebSocket消息出错:', e);
        setError(e); // 解析错误也视为一种错误状态
      }
    };

    wsInstance.onerror = (err) => {
      console.error('[useWebSocket] WebSocket错误 (onerror事件): ', err);
      setIsConnected(false);
      setError(err); // 存储错误信息
    };

    wsInstance.onclose = (event) => {
      console.log(
        '[useWebSocket] WebSocket连接关闭 (onclose事件). 代码:', event.code,
        '原因:', event.reason,
        '干净关闭:', event.wasClean,
        'readyState:', wsInstance.readyState
      );
      
      setWebsocket(null);
      setIsConnected(false);
      
      // 只有在非手动关闭且用户已登录时尝试重连
      if (!manualCloseRef.current && token && currentUser) {
        // 计算重连延迟时间（指数退避算法）
        const delay = Math.min(
          MAX_RECONNECT_DELAY,
          BASE_RECONNECT_DELAY * Math.pow(2, reconnectAttemptsRef.current)
        );
        
        console.log(`[useWebSocket] 安排WebSocket重连 #${reconnectAttemptsRef.current + 1}，延迟 ${delay}ms`);
        
        // 清除之前的重连定时器
        if (reconnectTimeoutRef.current) {
          clearTimeout(reconnectTimeoutRef.current);
        }
        
        // 如果未达到最大重连次数，设置重连定时器
        if (reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttemptsRef.current += 1;
            connectWebSocket();
          }, delay);
        } else {
          console.error('[useWebSocket] 达到最大重连次数，放弃重连');
        }
      } else {
        // 重置手动关闭标志
        manualCloseRef.current = false;
      }
      
      // 设置错误状态
      if (!event.wasClean) {
        let closeError = new Error(`WebSocket关闭异常. 代码: ${event.code}, 原因: ${event.reason || '无提供原因'}`);
        if (event.code === 1006) {
          closeError = new Error('WebSocket连接失败或异常关闭 (代码 1006)。这通常表示服务器端问题或网络问题。');
        }
        setError(closeError);
      }
    };

    return wsInstance;
  }, [token, currentUser]);

  // WebSocket连接管理
  useEffect(() => {
    console.log('[useWebSocket] Hook effect触发. Token:', token ? '存在' : '不存在', 'CurrentUser:', currentUser ? '存在' : '不存在');
    
    // 重置错误状态
    setError(null);

    // 清除之前的重连定时器
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (token && currentUser) {
      // 重置重连计数
      reconnectAttemptsRef.current = 0;
      manualCloseRef.current = false;
      
      // 创建新连接
      const wsInstance = connectWebSocket();
      
      return () => {
        console.log('[useWebSocket] 清理: wsInstance.readyState:', wsInstance?.readyState);
        if (wsInstance && (wsInstance.readyState === WebSocket.OPEN || wsInstance.readyState === WebSocket.CONNECTING)) {
          console.log('[useWebSocket] 清理: 正在关闭WebSocket.');
          manualCloseRef.current = true; // 标记为手动关闭，不要重连
          wsInstance.close(1000, "Hook卸载或依赖项更改");
        }
        
        // 清除重连定时器
        if (reconnectTimeoutRef.current) {
          clearTimeout(reconnectTimeoutRef.current);
          reconnectTimeoutRef.current = null;
        }
        
        setWebsocket(null);
        setIsConnected(false);
        console.log('[useWebSocket] 清理: WebSocket状态已清除.');
      };
    } else {
      console.log('[useWebSocket] 没有token或currentUser，确保WebSocket未激活.');
      if (websocket) {
        console.log('[useWebSocket] 关闭现有WebSocket (无token/user). ReadyState:', websocket.readyState);
        manualCloseRef.current = true; // 标记为手动关闭，不要重连
        websocket.close(1000, "用户登出或token过期");
        setWebsocket(null);
        setIsConnected(false);
      }
    }
  }, [token, currentUser, connectWebSocket]);

  // 手动重连函数
  const reconnect = useCallback(() => {
    console.log('[useWebSocket] 手动重连请求');
    
    // 关闭现有连接（如果有的话）
    if (websocket && (websocket.readyState === WebSocket.OPEN || websocket.readyState === WebSocket.CONNECTING)) {
      websocket.close(1000, "手动重连请求");
    }
    
    // 清除之前的重连定时器
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    
    // 重置重连尝试计数
    reconnectAttemptsRef.current = 0;
    manualCloseRef.current = false;
    
    // 立即尝试重新连接
    connectWebSocket();
  }, [websocket, connectWebSocket]);

  const sendMessage = useCallback((messagePayload) => {
    if (websocket && websocket.readyState === WebSocket.OPEN) {
      console.log('[useWebSocket] 发送WebSocket消息:', messagePayload);
      try {
        websocket.send(JSON.stringify(messagePayload));
      } catch (e) {
        console.error("[useWebSocket] 发送消息出错:", e);
        setError(e); // 发送错误也记录
      }
    } else {
      const errMsg = '[useWebSocket] 无法发送消息。WebSocket未打开或不可用.';
      console.warn(errMsg, { readyState: websocket?.readyState });
      setError(new Error(errMsg)); // 发送时连接不可用也视为错误
      
      // 尝试重连
      if (!isConnected && token && currentUser) {
        reconnect();
      }
    }
  }, [websocket, isConnected, token, currentUser, reconnect]);

  return { 
    lastMessage, 
    isConnected, 
    sendMessage, 
    websocketError: error, 
    reconnect // 暴露手动重连函数
  };
}

export default useWebSocket; 