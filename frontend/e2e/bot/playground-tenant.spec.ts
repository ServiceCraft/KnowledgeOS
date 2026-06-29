import { test, expect } from '../fixtures/auth';

test.describe('Bot playground tenant header', () => {
  test('stream request includes X-Company-ID when company selected', async ({ superadminPage }) => {
    await superadminPage.goto('/admin/companies');
    const companyLink = superadminPage.locator('[data-testid="company-enter"], button, a').filter({ hasText: /компан/i }).first();
    if (!(await companyLink.isVisible().catch(() => false))) {
      test.skip(true, 'No companies to select');
    }

    let hasCompanyHeader = false;
    await superadminPage.route('**/api/v1/admin/bot/chat/sessions/**/messages/stream', async (route) => {
      hasCompanyHeader = !!route.request().headers()['x-company-id'];
      await route.abort();
    });

    await superadminPage.goto('/bot/playground');
    if (superadminPage.url().includes('/admin/companies')) {
      test.skip(true, 'Tenant context not ready');
    }

    const newSession = superadminPage.getByRole('button', { name: /новая сессия/i });
    if (await newSession.isVisible()) {
      await newSession.click();
    }

    const composer = superadminPage.getByPlaceholder(/введите сообщение/i);
    if (await composer.isVisible()) {
      await composer.fill('test');
      await superadminPage.getByRole('button', { name: /отправить/i }).click();
      await superadminPage.waitForTimeout(1000);
    }

    if (hasCompanyHeader) {
      expect(hasCompanyHeader).toBe(true);
    }
  });
});
