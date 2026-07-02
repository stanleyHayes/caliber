import { expect, test } from '@playwright/test';

import { loginAs } from './helpers/auth';

test.describe('Flow C — candidate agent runs overnight', () => {
  test('wake-up card and applications list render', async ({ page }) => {
    await loginAs(page, 'candidate');
    await page.goto('/agent');

    await page.getByRole('button', { name: 'Run overnight' }).click();
    await expect(page.getByRole('heading', { name: /while you were away/i })).toBeVisible({ timeout: 30000 });

    await expect(page.getByRole('heading', { name: 'Applications' })).toBeVisible();
    await expect(page.getByText(/by your agent/i).first()).toBeVisible();
  });
});
