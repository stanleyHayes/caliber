import type { Page } from '@playwright/test';

const password = 'Demo-Caliber-2026';

const accounts = {
  employer: 'talent@mtn.com.gh',
  candidate: 'ama.mensah@example.com',
};

export async function loginAs(page: Page, role: 'employer' | 'candidate') {
  const email = accounts[role];
  await page.goto('/login');
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await page.waitForURL('/app');
  await page.waitForSelector('[data-testid="app-shell"]', { timeout: 10000 });
}

export async function logout(page: Page) {
  await page.getByRole('button', { name: /sign out/i }).click();
  await page.waitForURL('/login');
}
