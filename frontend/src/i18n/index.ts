import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import {initReactI18next} from 'react-i18next';

import en from './en.json';
import ja from './ja.json';

const i18nData = {
  en: {
    translation: en
  },
  ja: {
    translation: ja
  }
};

export default i18n
  .use(initReactI18next)
  .use(LanguageDetector)
  .init({
    resources: i18nData,
    debug: false,
    interpolation: {
      escapeValue: false
    }
  });
