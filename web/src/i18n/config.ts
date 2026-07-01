import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

import en from './locales/en.json';
import fr from './locales/fr.json';
import tw from './locales/tw.json';

export const DEFAULT_LOCALE = 'en';
export const SUPPORTED_LOCALES = ['en', 'tw', 'fr'] as const;
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

export const LOCALE_NAMES: Record<SupportedLocale, string> = {
  en: 'English',
  tw: 'Twi',
  fr: 'Français',
};

export const LOCALE_STORAGE_KEY = 'caliber-locale';

i18n
  .use(initReactI18next)
  .use(LanguageDetector)
  .init({
    resources: {
      en: { translation: en },
      tw: { translation: tw },
      fr: { translation: fr },
    },
    fallbackLng: DEFAULT_LOCALE,
    supportedLngs: SUPPORTED_LOCALES,
    interpolation: { escapeValue: false },
    detection: {
      order: ['localStorage', 'navigator', 'htmlTag'],
      caches: ['localStorage'],
      lookupLocalStorage: LOCALE_STORAGE_KEY,
    },
    react: { useSuspense: false },
  });

export { i18n };
