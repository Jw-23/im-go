import React from 'react';
import './SettingsButton.css';
import { useTranslation } from 'react-i18next';

const SettingsButton = ({ onClick }) => {
  const { t } = useTranslation();
  return (
    <button className="settings-button" onClick={onClick} aria-label={t('settings_button_label')}>
      {/* Use an icon here in the future, e.g., SVG or an icon font */}
      <span role="img" aria-hidden="true">⚙️</span> {/* Basic gear icon */}
      {/* Settings */}
    </button>
  );
};

export default SettingsButton; 