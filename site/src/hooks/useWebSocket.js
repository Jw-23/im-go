import { useState, useEffect, useCallback } from 'react';
import { useAuth } from '../contexts/AuthContext';

const CHAT_SERVER_URL = 'ws://localhost:8080'; // 从 App.jsx 迁移
const WEBSOCKET_PATH = '/ws/chat';             // 从 App.jsx 迁移

function useWebSocket() {
  const [websocket, setWebsocket] = useState(null);
  const [lastMessage, setLastMessage] = useState(null); // 用于接收最新消息
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState(null); // 用于存储 WebSocket 错误信息
  const { token, currentUser } = useAuth();

  useEffect(() => {
    console.log('[useWebSocket] Hook effect triggered. Token:', token, 'CurrentUser:', currentUser);
    // 重置上一个连接可能产生的错误状态
    setError(null);

    if (token && currentUser) {
      const wsUrl = `${CHAT_SERVER_URL}${WEBSOCKET_PATH}?token=${token}`;
      console.log('[useWebSocket] Preparing to connect WebSocket. URL:', wsUrl);
      const wsInstance = new WebSocket(wsUrl);

      wsInstance.onopen = () => {
        console.log('[useWebSocket] WebSocket connection established (onopen event). ReadyState:', wsInstance.readyState);
        setWebsocket(wsInstance);
        setIsConnected(true);
        setError(null); // 连接成功，清除错误
      };

      wsInstance.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          console.log('[useWebSocket] WebSocket message received:', message);
          setLastMessage(message); // 更新收到的最新消息
        } catch (e) {
          console.error('[useWebSocket] Error parsing WebSocket message:', e);
          setError(e); // 解析错误也视为一种错误状态
        }
      };

      wsInstance.onerror = (err) => {
        console.error('[useWebSocket] WebSocket error (onerror event): ', err);
        setIsConnected(false);
        setError(err); // 存储错误信息
        // setWebsocket(null); // 通常 onerror 后会紧跟 onclose，onclose 中会清理 websocket
      };

      wsInstance.onclose = (event) => {
        console.log(
          '[useWebSocket] WebSocket connection closed (onclose event). Code:', event.code,
          'Reason:', event.reason,
          'wasClean:', event.wasClean,
          'readyState at close:', wsInstance.readyState
        );
        setWebsocket(null);
        setIsConnected(false);
        // 根据关闭代码判断是否是错误关闭
        if (!event.wasClean) {
            let closeError = new Error(`WebSocket closed uncleanly. Code: ${event.code}, Reason: ${event.reason || 'No reason provided'}`);
            if (event.code === 1006){
                closeError = new Error('WebSocket connection failed or was abnormally closed (Code 1006). This often indicates server-side issues or network problems.');
            }
            setError(closeError);
        }
      };

      return () => {
        console.log('[useWebSocket] Cleanup: wsInstance.readyState at start:', wsInstance?.readyState);
        if (wsInstance && (wsInstance.readyState === WebSocket.OPEN || wsInstance.readyState === WebSocket.CONNECTING)) {
          console.log('[useWebSocket] Cleanup: Closing WebSocket.');
          wsInstance.close(1000, "Hook unmounting or dependencies changed");
        }
        setWebsocket(null);
        setIsConnected(false);
        // setError(null); // 清理时不一定清除错误，App 组件可能还需要展示最后一次的错误
        console.log('[useWebSocket] Cleanup: WebSocket state cleared.');
      };
    } else {
      console.log('[useWebSocket] No token or currentUser, ensuring WebSocket is not active.');
      if (websocket) {
        console.log('[useWebSocket] Closing existing WebSocket (no token/user). ReadyState:', websocket.readyState);
        websocket.close(1000, "User logged out or token expired");
        setWebsocket(null);
        setIsConnected(false);
      }
      // 如果因为缺少 token/currentUser 而没有尝试连接，则不应设置错误
      // setError(null);
    }
  }, [token, currentUser]);

  const sendMessage = useCallback((messagePayload) => {
    if (websocket && websocket.readyState === WebSocket.OPEN) {
      console.log('[useWebSocket] Sending WebSocket message:', messagePayload);
      try {
      websocket.send(JSON.stringify(messagePayload));
      } catch (e) {
        console.error("[useWebSocket] Error sending message:", e);
        setError(e); // 发送错误也记录
      }
    } else {
      const errMsg = '[useWebSocket] Cannot send message. WebSocket not open or not available.';
      console.warn(errMsg, { readyState: websocket?.readyState });
      setError(new Error(errMsg)); // 发送时连接不可用也视为错误
    }
  }, [websocket]);

  return { lastMessage, isConnected, sendMessage, websocketError: error };
}

export default useWebSocket; 