import {test, expect} from '@playwright/test';
import {TARGET_HOST} from '../playwright.config';

test('Should setup server', async ({page}) => {
    await page.goto(TARGET_HOST);
    await page.getByText(/^2 GB/).click();
    await page.getByText('Next').click();
    await page.getByText('Next').click();
    await page.getByLabel('Generate a New World').click();
    await page.getByText('Next').click();
    await page.getByLabel('World Name').fill(`e2e-test-${Date.now()}`);
    await page.getByText('Next').click();
    await page.getByText('Next').click();
    await page.getByText('Start').click();

    await expect(page.getByText('Stop')).toBeVisible();
});
