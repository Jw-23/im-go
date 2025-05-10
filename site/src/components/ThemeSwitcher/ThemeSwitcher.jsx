import React from 'react';
import { useTheme } from '../../contexts/ThemeContext';
import './ThemeSwitcher.css';
import { useTranslation } from 'react-i18next';

const ThemeSwitcher = () => {
  const { themePreference, setThemePreference } = useTheme();
  const { t } = useTranslation();

  return (
    <div className="theme-switcher">
      <label htmlFor="theme-select" className="theme-switcher__label">{t('theme')}:</label>
      <select 
        id="theme-select" 
        value={themePreference} 
        onChange={(e) => setThemePreference(e.target.value)}
        className="theme-switcher__select"
      >
        <option value="auto">{t('theme_auto')}</option>
        <option value="light">{t('theme_light')}</option>
        <option value="dark">{t('theme_dark')}</option>
      </select>
    </div>
  );
};

export default ThemeSwitcher; 