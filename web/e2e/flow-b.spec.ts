import { expect, test } from '@playwright/test';
import { createRole, loginAs } from './helpers/auth';

test.describe('Flow B — candidate completes a screening interview', () => {
  test('answers adaptive questions and receives a report card', async ({ page }) => {
    // Create a role as an employer so we have a deterministic role id.
    await loginAs(page, 'employer');
    await createRole(page, 'We need a backend engineer in Accra with Go, Postgres, and gRPC experience.');

    const interviewLink = page.getByRole('link', { name: 'Run a screening interview' });
    await expect(interviewLink).toBeVisible({ timeout: 15000 });
    const href = await interviewLink.getAttribute('href');
    const roleId = new URLSearchParams(href!.split('?')[1]).get('roleId')!;

    // Run the interview as the candidate.
    await loginAs(page, 'candidate');
    await page.goto(`/interview?roleId=${roleId}`);
    await page.getByRole('button', { name: 'Start interview' }).click();

    // Answer up to a bounded number of questions.
    for (let i = 0; i < 6; i++) {
      const field = page.getByRole('textbox', { name: 'Answer with a concrete example…' });
      // Wait (bounded) for the next question's field to appear; once it stops
      // appearing the interview is complete, so exit the loop. Branch on the wait
      // rather than swallowing an assertion that could never fail.
      const appeared = await field
        .waitFor({ state: 'visible', timeout: 5000 })
        .then(() => true)
        .catch(() => false);
      if (!appeared) break;
      await field.fill(
        'I built a Go service that handled 10k RPS using Postgres and gRPC, with structured logging and unit tests.'
      );
      await page.getByRole('button', { name: 'Submit answer' }).click();
    }

    await expect(page.getByRole('heading', { name: /report card/i })).toBeVisible({ timeout: 30000 });
    await expect(page.getByText(/Medium confidence/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /Dispute this report card/i })).toBeVisible();
  });
});
