import { expect, test } from '@playwright/test';
import { loginAs } from './helpers/auth';

test.describe('Flow A — employer creates a role and shortlist', () => {
  test('generates a spec, rubric, and explainable shortlist', async ({ page }) => {
    await loginAs(page, 'employer');
    await page.goto('/roles/new');

    await page.getByPlaceholder(/senior Go backend engineer/i).fill(
      'We need a senior backend engineer in Accra. Must know Go, Postgres, and gRPC. GHS 20k-28k, start within one month.'
    );
    await page.getByRole('button', { name: 'Generate spec & rubric' }).click();

    await expect(page.getByRole('heading', { name: 'Scoring rubric' })).toBeVisible({ timeout: 15000 });
    await expect(page.getByRole('heading', { name: 'Explainable shortlist' })).toBeVisible();

    await page.getByRole('button', { name: 'Generate shortlist' }).click();
    await expect(page.getByText(/in pool/i)).toBeVisible({ timeout: 15000 });
  });
});
