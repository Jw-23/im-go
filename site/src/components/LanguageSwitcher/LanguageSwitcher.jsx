import React from 'react';
import { useTranslation } from 'react-i18next';
import './LanguageSwitcher.css'; // We'll create this CSS file next

const LanguageSwitcher = () => {
  const { i18n, t } = useTranslation();

  const changeLanguage = (lng) => {
    i18n.changeLanguage(lng);
  };

  return (
    <div className="language-switcher">
      <label htmlFor="language-select" className="language-switcher__label">{t('language')}:</label>
      <select 
        id="language-select" 
        value={i18n.resolvedLanguage}
        onChange={(e) => changeLanguage(e.target.value)}
        className="language-switcher__select"
      >
        <option value="zh">{t('language_chinese')}</option>
        <option value="en">{t('language_english')}</option>
      </select>
    </div>
  );
};

export default LanguageSwitcher; 