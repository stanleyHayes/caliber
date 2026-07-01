import '@testing-library/jest-dom/vitest';

import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// Initialize the i18n instance for tests so components can call useTranslation
// without every test wrapping an I18nextProvider.
import './i18n';

const makeStorage = (): Storage => {
  const items = new Map<string, string>();
  return {
    get length() {
      return items.size;
    },
    clear: () => items.clear(),
    getItem: (key: string) => items.get(key) ?? null,
    key: (index: number) => Array.from(items.keys())[index] ?? null,
    removeItem: (key: string) => {
      items.delete(key);
    },
    setItem: (key: string, value: string) => {
      items.set(key, value);
    },
  };
};

const localStorageMock = makeStorage();

Object.defineProperty(globalThis, 'localStorage', {
  configurable: true,
  value: localStorageMock,
});
Object.defineProperty(window, 'localStorage', {
  configurable: true,
  value: localStorageMock,
});

// With globals disabled, RTL's automatic afterEach(cleanup) isn't registered, so
// unmount + reset the DOM between tests ourselves to keep them isolated.
afterEach(() => {
  cleanup();
});
