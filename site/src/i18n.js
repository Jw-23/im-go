import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import HttpBackend from 'i18next-http-backend'; // 我们稍后需要安装这个

i18n
  // 使用 i18next-http-backend 插件，允许从远程路径加载翻译文件
  // 例如: /locales/{{lng}}/{{ns}}.json
  .use(HttpBackend)
  // 使用 i18next-browser-languagedetector 插件，自动检测用户语言
  .use(LanguageDetector)
  // 将 i18n 实例传递给 react-i18next
  .use(initReactI18next)
  // 初始化 i18next
  .init({
    // 默认语言
    fallbackLng: 'en',
    // 调试模式，将在控制台输出信息
    debug: false,
    saveMissing: false,
    // 命名空间，用于组织翻译内容
    ns: ['translation'],
    defaultNS: 'translation',
    // interpolation: {
    //   escapeValue: false, // react already safes from xss
    // },
    backend: {
      // 翻译文件的路径
      // {{lng}} 会被替换为语言代码 (e.g., 'en', 'zh')
      // {{ns}} 会被替换为命名空间 (e.g., 'translation')
      loadPath: '/locales/{{lng}}/{{ns}}.json',
    },
    detection: {
      // 配置语言检测器
      order: ['localStorage', 'navigator', 'htmlTag', 'path', 'subdomain'],
      caches: ['localStorage'], // 将检测到的语言缓存在localStorage中
    }
  });

export default i18n; 