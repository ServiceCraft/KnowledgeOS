import { test, expect } from '../fixtures/auth';

test.describe('KB QA list', () => {
  test('redirects to company selection without tenant context', async ({ superadminPage }) => {
    await superadminPage.goto('/kb/qa');
    await expect(superadminPage).toHaveURL(/\/admin\/companies/);
  });
});
