import { expect, test } from '@playwright/test';
import { loginAs, logout } from './helpers/auth';

test.describe('authentication', () => {
  test('employer can sign in through the UI', async ({ page }) => {
    await page.goto('/login');
    await page.getByLabel('Email').fill('talent@mtn.com.gh');
    await page.getByLabel('Password').fill('Demo-Caliber-2026');
    await page.getByRole('button', { name: 'Sign in' }).click();
    await page.waitForURL('/app');
    await expect(page.getByRole('heading', { name: /welcome/i })).toBeVisible();
  });

  test('shows an error for invalid credentials', async ({ page }) => {
    await page.goto('/login');
    const email = `bad-${Date.now()}@example.com`;
    await page.getByLabel('Email').fill(email);
    await page.getByLabel('Password').fill('wrong');
    await page.getByRole('button', { name: 'Sign in' }).click();
    await expect(page.getByRole('alert')).toContainText(/invalid/i);
  });

  test('restores a session from localStorage refresh token', async ({ page }) => {
    await loginAs(page, 'employer');
    await page.reload();
    await page.waitForSelector('[data-testid="app-shell"]');
    await page.goto('/radar');
    await expect(page.getByRole('heading', { name: 'Talent Radar' })).toBeVisible();
  });

  test('sign out redirects to login', async ({ page }) => {
    await loginAs(page, 'employer');
    await logout(page);
    await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible();
  });
});
