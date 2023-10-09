import {test as setup, expect} from '@playwright/test';
import {TARGET_HOST} from '../playwright.config';

const authFile = 'playwright/.auth/user.json';

setup('login', async ({page}) => {
  await page.goto(TARGET_HOST);
  await page.getByLabel('User').fill('user1');
  await page.getByLabel('Password').fill('password');
  await page.getByRole('button', {name: 'Login'}).click();

  await expect(page.getByText('Logout')).toBeVisible();

  await page.context().storageState({path: authFile});
});
