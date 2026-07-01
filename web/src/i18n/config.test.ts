import { describe, expect, it } from 'vitest';

import { DEFAULT_LOCALE, i18n, LOCALE_NAMES, SUPPORTED_LOCALES } from './config';

describe('i18n configuration', () => {
  it('defaults to English for Ghana/West Africa first', () => {
    expect(i18n.language).toBe(DEFAULT_LOCALE);
    expect(DEFAULT_LOCALE).toBe('en');
  });

  it('advertises English, Twi and French as supported locales', () => {
    expect(SUPPORTED_LOCALES).toEqual(['en', 'tw', 'fr']);
    expect(LOCALE_NAMES.en).toBe('English');
    expect(LOCALE_NAMES.tw).toBe('Twi');
    expect(LOCALE_NAMES.fr).toBe('Français');
  });

  it('falls back to English for missing translations', () => {
    const englishHeadline = i18n.t('landing.headline');
    expect(englishHeadline).toBeTruthy();
    expect(typeof englishHeadline).toBe('string');
  });

  it('can switch to Twi and return translated strings', async () => {
    await i18n.changeLanguage('tw');
    expect(i18n.t('nav.signIn')).toBe('Kɔ mu');
    expect(i18n.t('notFound.backHome')).toBe('San kɔ fie');
    // Reset to avoid leaking state into later tests.
    await i18n.changeLanguage(DEFAULT_LOCALE);
  });

  it('can switch to French and return translated strings', async () => {
    await i18n.changeLanguage('fr');
    expect(i18n.t('nav.signIn')).toBe('Se connecter');
    expect(i18n.t('notFound.backHome')).toBe("Retour à l'accueil");
    await i18n.changeLanguage(DEFAULT_LOCALE);
  });
});
