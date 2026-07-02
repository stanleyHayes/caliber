import { expect, test } from '@playwright/test';

import { loginAs } from './helpers/auth';

test.describe('Talent Radar dashboard', () => {
  test('renders all four panels for an employer', async ({ page }) => {
    await loginAs(page, 'employer');
    await page.goto('/radar');

    await expect(page.getByRole('heading', { name: 'Talent Radar' })).toBeVisible();
    await expect(page.getByText(/faster/i)).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Supply & demand' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Live talent pool' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Match alerts' })).toBeVisible();
  });
});
