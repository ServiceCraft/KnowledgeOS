import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const authDir = path.join(__dirname, '.auth');

const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:8080';
const superadminEmail = process.env.E2E_SUPERADMIN_EMAIL ?? 'admin@example.com';
const superadminPassword = process.env.E2E_SUPERADMIN_PASSWORD ?? 'changeme';

interface LoginResponse {
  data: {
    access_token: string;
    refresh_token: string;
    user: { id: string; email: string; role: string; company_ids?: string[] };
  };
}

async function apiLogin(email: string, password: string) {
  const resp = await fetch(`${baseURL}/api/v1/auth/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });
  if (!resp.ok) {
    throw new Error(`Login failed for ${email}: HTTP ${resp.status}. Is docker compose running at ${baseURL}?`);
  }
  return (await resp.json()) as LoginResponse;
}

async function globalSetup() {
  fs.mkdirSync(authDir, { recursive: true });

  const health = await fetch(baseURL).catch(() => null);
  if (!health?.ok) {
    throw new Error(
      `App not reachable at ${baseURL}. Start the stack: docker compose up -d --build`
    );
  }

  const superadmin = await apiLogin(superadminEmail, superadminPassword);
  fs.writeFileSync(
    path.join(authDir, 'superadmin.json'),
    JSON.stringify(
      {
        cookies: [],
        origins: [
          {
            origin: baseURL,
            localStorage: [
              {
                name: 'auth-storage',
                value: JSON.stringify({
                  state: {
                    user: superadmin.data.user,
                    tokens: {
                      access_token: superadmin.data.access_token,
                      refresh_token: superadmin.data.refresh_token,
                    },
                    isAuthenticated: true,
                    selectedCompanyId: null,
                    selectedCompanyName: null,
                  },
                  version: 0,
                }),
              },
            ],
          },
        ],
      },
      null,
      2
    )
  );
}

export default globalSetup;
