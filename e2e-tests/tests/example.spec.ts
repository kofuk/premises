import { test, expect } from '@playwright/test';
import { TARGET_HOST } from '../playwright.config';

test('has title', async ({ page }) => {
  await page.goto(TARGET_HOST);

  // Expect a title "to contain" a substring.
  await expect(page).toHaveTitle(/Login/);
});
