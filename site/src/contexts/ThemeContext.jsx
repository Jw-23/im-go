import React, { createContext, useState, useEffect, useContext } from 'react';

const ThemeContext = createContext();

export const useTheme = () => useContext(ThemeContext);

export const ThemeProvider = ({ children }) => {
  // themePreference stores the user's explicit choice: 'light', 'dark', or 'auto'
  const [themePreference, setThemePreference] = useState(() => {
    const storedPreference = localStorage.getItem('themePreference');
    return storedPreference || 'auto';
  });

  // appliedTheme stores the actual theme being applied: 'light' or 'dark'
  const [appliedTheme, setAppliedTheme] = useState('light');

  useEffect(() => {
    localStorage.setItem('themePreference', themePreference);

    const root = window.document.documentElement;

    const applyTheme = (themeToApply) => {
      root.classList.remove('light', 'dark');
      root.classList.add(themeToApply);
      root.setAttribute('data-theme', themeToApply);
      setAppliedTheme(themeToApply);
    };

    const handleSystemThemeChange = (e) => {
      if (themePreference === 'auto') {
        applyTheme(e.matches ? 'dark' : 'light');
      }
    };

    if (themePreference === 'auto') {
      const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
      applyTheme(mediaQuery.matches ? 'dark' : 'light');
      mediaQuery.addEventListener('change', handleSystemThemeChange);
      return () => mediaQuery.removeEventListener('change', handleSystemThemeChange);
    } else {
      applyTheme(themePreference);
    }
  }, [themePreference]);

  return (
    <ThemeContext.Provider value={{ themePreference, setThemePreference, appliedTheme }}>
      {children}
    </ThemeContext.Provider>
  );
}; 