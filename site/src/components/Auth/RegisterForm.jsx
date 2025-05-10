import React, { useState } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import './AuthForm.css'; // Shared CSS for auth forms
import { useTranslation } from 'react-i18next';

const RegisterForm = ({ onSwitchToLogin, onSuccess }) => {
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [nickname, setNickname] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const { register } = useAuth();
  const { t } = useTranslation();

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);
    const result = await register({ username, email, password, nickname });
    setIsLoading(false);
    if (result.success) {
      alert(t('registration_successful_message')); // Or auto-login
      if(onSuccess) onSuccess(); // e.g. switch to login form
    } else {
      setError(result.error || t('registration_failed_message'));
    }
  };

  return (
    <form onSubmit={handleSubmit} className="auth-form">
      <h2>{t('register')}</h2>
      {error && <p className="auth-form__error">{error}</p>}
      <div className="auth-form__field">
        <label htmlFor="register-username">{t('username')}</label>
        <input type="text" id="register-username" value={username} onChange={(e) => setUsername(e.target.value)} required />
      </div>
      <div className="auth-form__field">
        <label htmlFor="register-email">{t('email')}</label>
        <input type="email" id="register-email" value={email} onChange={(e) => setEmail(e.target.value)} required />
      </div>
      <div className="auth-form__field">
        <label htmlFor="register-password">{t('password')}</label>
        <input type="password" id="register-password" value={password} onChange={(e) => setPassword(e.target.value)} required />
      </div>
      <div className="auth-form__field">
        <label htmlFor="register-nickname">{t('nickname')}</label>
        <input type="text" id="register-nickname" value={nickname} onChange={(e) => setNickname(e.target.value)} />
      </div>
      <button type="submit" className="auth-form__button" disabled={isLoading}>
        {isLoading ? t('registering') : t('register_button')}
      </button>
      <p className="auth-form__switch">
        {t('already_have_account')} <button type="button" onClick={onSwitchToLogin} className="auth-form__switch-button">{t('login')}</button>
      </p>
    </form>
  );
};

export default RegisterForm; 