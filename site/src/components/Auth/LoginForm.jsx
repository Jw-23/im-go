import React, { useState } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import './AuthForm.css'; // Shared CSS for auth forms
import { useTranslation } from 'react-i18next';

const LoginForm = ({ onSwitchToRegister, onSuccess }) => {
  const [usernameOrEmail, setUsernameOrEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const { login } = useAuth();
  const { t } = useTranslation();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);
    const result = await login({ username: usernameOrEmail, password });
    setIsLoading(false);
    if (result.success) {
      if(onSuccess) onSuccess();
    } else {
      setError(result.error || t('login_failed_message'));
    }
  };

  return (
    <form onSubmit={handleSubmit} className="auth-form">
      <h2>{t('login')}</h2>
      {error && <p className="auth-form__error">{error}</p>}
      <div className="auth-form__field">
        <label htmlFor="login-username">{t('username_or_email')}</label>
        <input 
          type="text" 
          id="login-username" 
          value={usernameOrEmail} 
          onChange={(e) => setUsernameOrEmail(e.target.value)} 
          required 
          placeholder={t('enter_username_or_email')}
        />
      </div>
      <div className="auth-form__field">
        <label htmlFor="login-password">{t('password')}</label>
        <input 
          type="password" 
          id="login-password" 
          value={password} 
          onChange={(e) => setPassword(e.target.value)} 
          required 
        />
      </div>
      <button type="submit" className="auth-form__button" disabled={isLoading}>
        {isLoading ? t('logging_in') : t('login_button')}
      </button>
      <p className="auth-form__switch">
        {t('dont_have_account')} <button type="button" onClick={onSwitchToRegister} className="auth-form__switch-button">{t('register')}</button>
      </p>
    </form>
  );
};

export default LoginForm; 