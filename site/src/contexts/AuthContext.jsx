import React, { createContext, useState, useEffect, useContext } from 'react';
import { registerUser, loginUser, getCurrentUserProfile } from '../services/api';

const AuthContext = createContext();

export const useAuth = () => useContext(AuthContext);

export const AuthProvider = ({ children }) => {
  const [currentUser, setCurrentUser] = useState(null);
  const [token, setToken] = useState(localStorage.getItem('jwtToken'));
  const [isLoading, setIsLoading] = useState(true); // Start with loading true to check initial auth status
  const [authError, setAuthError] = useState(null);

  useEffect(() => {
    const initializeAuth = async () => {
      if (token) {
        localStorage.setItem('jwtToken', token); // Ensure it's set if passed from initial state
        const response = await getCurrentUserProfile();
        if (response.success && response.data) {
          setCurrentUser(response.data);
        } else {
          // Token might be invalid or expired
          localStorage.removeItem('jwtToken');
          setToken(null);
          setCurrentUser(null);
          if(response.status === 401) {
            console.log("Session expired or token invalid.");
          } else if (response.error){
            setAuthError(response.error); // Store other errors
          }
        }
      } else {
        localStorage.removeItem('jwtToken'); // Ensure no token if not in state
      }
      setIsLoading(false);
    };
    initializeAuth();
  }, [token]); // Rerun if token changes (e.g. after login/logout)

  const handleLogin = async (credentials) => {
    setIsLoading(true);
    setAuthError(null);
    const response = await loginUser(credentials);
    if (response.success && response.data.token) {
      setToken(response.data.token);
      // User profile will be fetched by the useEffect due to token change
      return { success: true };
    } else {
      setCurrentUser(null);
      setAuthError(response.error || 'Login failed.');
      setIsLoading(false);
      return { success: false, error: response.error || 'Login failed.' };
    }
  };

  const handleRegister = async (userData) => {
    setIsLoading(true);
    setAuthError(null);
    const response = await registerUser(userData);
    if (response.success && response.data) {
      // Optionally log in the user directly after registration
      // For now, let's just inform success and they can log in.
      setIsLoading(false);
      return { success: true, data: response.data };
    } else {
      setAuthError(response.error || 'Registration failed.');
      setIsLoading(false);
      return { success: false, error: response.error || 'Registration failed.' };
    }
  };

  const handleLogout = () => {
    setIsLoading(true);
    setAuthError(null);
    setToken(null);
    setCurrentUser(null);
    localStorage.removeItem('jwtToken');
    setIsLoading(false);
    // Here you might want to also clear other user-related state in your app
  };

  const value = {
    currentUser,
    token,
    isLoading,
    authError,
    login: handleLogin,
    register: handleRegister,
    logout: handleLogout,
    clearAuthError: () => setAuthError(null),
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}; 