import { test, expect } from '../fixtures/auth';

test.describe('Superadmin sidebar', () => {
  test('without selected company KB links are hidden', async ({ superadminPage }) => {
    await superadminPage.goto('/admin/companies');
    await expect(superadminPage.getByRole('link', { name: 'Вопросы и ответы' })).not.toBeVisible();
    await expect(superadminPage.locator('[data-sidebar="menu-button"]').filter({ hasText: 'Компании' })).toBeVisible();
  });
});

test.describe('Superadmin company context', () => {
  test('tenant-scoped route redirects to admin companies without selection', async ({ superadminPage }) => {
    await superadminPage.goto('/kb/qa');
    await expect(superadminPage).toHaveURL(/\/admin\/companies/);
  });
});
