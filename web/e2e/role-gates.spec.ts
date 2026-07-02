import { expect, test } from '@playwright/test';
import { loginAs } from './helpers/auth';

test.describe('role gates', () => {
  test('candidate sees permission alerts on employer dashboard', async ({ page }) => {
    await loginAs(page, 'candidate');
    await page.goto('/radar');
    await expect(page.getByText(/insufficient permissions/i).first()).toBeVisible();
  });

  test('employer sees permission alerts on candidate agent page', async ({ page }) => {
    await loginAs(page, 'employer');
    await page.goto('/agent');
    await expect(page.getByText(/insufficient permissions/i).first()).toBeVisible();
  });
});
