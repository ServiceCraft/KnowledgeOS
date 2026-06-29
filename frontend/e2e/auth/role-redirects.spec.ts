import { test, expect } from '../fixtures/auth';

test.describe('Role redirects', () => {
  test('superadmin cannot access export without company context from kb', async ({ superadminPage }) => {
    await superadminPage.goto('/settings/export');
    await expect(superadminPage).toHaveURL(/\/admin\/companies/);
  });
});
