import React, { useState } from 'react';
import './SettingsModal.css';
import ThemeSwitcher from '../ThemeSwitcher/ThemeSwitcher';
import LanguageSwitcher from '../LanguageSwitcher/LanguageSwitcher';
import LoginForm from '../Auth/LoginForm';
import RegisterForm from '../Auth/RegisterForm';
import { useAuth } from '../../contexts/AuthContext';
import { useTranslation } from 'react-i18next';

const SettingsModal = ({ isOpen, onClose }) => {
  const { currentUser, logout, authError, clearAuthError, isLoading: authIsLoading } = useAuth();
  const [showLogin, setShowLogin] = useState(true);
  const { t } = useTranslation();
  const [activeSection, setActiveSection] = useState('language'); // Default active section

  if (!isOpen) {
    return null;
  }

  const handleClose = () => {
    clearAuthError();
    onClose();
    setActiveSection('language'); // Reset to default section on close
  };

  const handleAuthSuccess = () => {
    // Potentially close modal on successful login/registration or show user info
    // For now, AuthContext handles user state, we can keep modal open or close it.
    // Let's close it for simplicity after login/register action from form itself.
    handleClose();
  }

  const navItems = [
    { key: 'language', label: t('language') },
    { key: 'theme', label: t('theme') },
    { key: 'account', label: t('account') },
    // Add more sections here if needed
    // { key: 'more', label: t('more_settings_placeholder_nav') } // Example for a new nav item
  ];

  return (
    <div className="settings-modal__overlay" onClick={handleClose}>
      <div className="settings-modal__content" onClick={(e) => e.stopPropagation()}>
        <div className="settings-modal__header">
          <h2>{t('settings')}</h2>
          <button onClick={handleClose} className="settings-modal__close-button" aria-label={t('close_settings_button_label')}>
            &times;
          </button>
        </div>
        <div className="settings-modal__body-container"> 
          <nav className="settings-modal__nav">
            {navItems.map(item => (
              <button
                key={item.key}
                className={`settings-modal__nav-item ${activeSection === item.key ? 'settings-modal__nav-item--active' : ''}`}
                onClick={() => setActiveSection(item.key)}
              >
                {item.label}
              </button>
            ))}
          </nav>
          <main className="settings-modal__main-content">
            {activeSection === 'language' && (
              <div className="settings-modal__section language-settings-section">
                <LanguageSwitcher />
              </div>
            )}

            {activeSection === 'theme' && (
              <div className="settings-modal__section theme-settings-section">
                <ThemeSwitcher />
              </div>
            )}

            {activeSection === 'account' && (
              <div className="settings-modal__section auth-settings-section">
                {authIsLoading && <p>{t('loading_account')}</p>}
                {!authIsLoading && currentUser ? (
                  <div className="user-info">
                    <p>{t('logged_in_as')} <strong>{currentUser.nickname || currentUser.username}</strong> ({currentUser.email})</p>
                    <button onClick={logout} className="auth-form__button auth-form__button--logout">{t('logout')}</button>
                  </div>
                ) : (
                  <>
                    {showLogin ? (
                      <LoginForm 
                        onSwitchToRegister={() => setShowLogin(false)} 
                        onSuccess={handleAuthSuccess} 
                      />
                    ) : (
                      <RegisterForm 
                        onSwitchToLogin={() => setShowLogin(true)} 
                        onSuccess={() => { 
                            alert(t('registration_successful_message'));
                            setShowLogin(true); // Switch to login form after successful registration
                        }}
                      />
                    )}
                  </>
                )}
                {authError && !currentUser && <p className="auth-form__error" style={{marginTop: '10px'}}>{authError}</p>}
              </div>
            )}
          </main>
        </div>
      </div>
    </div>
  );
};

export default SettingsModal; 