import { test, expect, loginViaUI } from '../fixtures/auth';

const superadminEmail = process.env.E2E_SUPERADMIN_EMAIL ?? 'admin@example.com';
const superadminPassword = process.env.E2E_SUPERADMIN_PASSWORD ?? 'changeme';

test.describe('Login', () => {
  test('shows error on invalid credentials', async ({ page }) => {
    await page.goto('/login');
    await page.getByLabel('Email').fill('wrong@example.com');
    await page.getByLabel('Пароль').fill('wrongpassword');
    await page.getByRole('button', { name: 'Войти' }).click();
    await expect(page.getByText('Неверный email или пароль')).toBeVisible();
  });

  test('superadmin redirects to admin companies', async ({ page }) => {
    await loginViaUI(page, superadminEmail, superadminPassword);
    await expect(page).toHaveURL(/\/admin\/companies/);
  });

  test('authenticated superadmin visiting login goes to admin companies', async ({ superadminPage }) => {
    await superadminPage.goto('/login');
    await expect(superadminPage).toHaveURL(/\/admin\/companies/);
  });
});

test.describe('Auth guards', () => {
  test('unauthenticated user is redirected to login', async ({ page }) => {
    await page.goto('/kb/qa');
    await expect(page).toHaveURL(/\/login/);
  });

  test('wildcard route redirects unauthenticated to login', async ({ page }) => {
    await page.goto('/unknown-path');
    await expect(page).toHaveURL(/\/login/);
  });
});
