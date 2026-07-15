import type { AxiosError } from 'axios';

export function apiError(err: unknown, fallback: string): string {
  const ax = err as AxiosError<{ error?: string }>;
  return ax?.response?.data?.error ?? fallback;
}

// Known backend tenant/auth errors translated for the user. Surfacing the real
// cause («выберите компанию») instead of a generic failure makes access issues
// self-diagnosing.
const API_ERROR_LABELS: Record<string, string> = {
  'company selection required': 'Выберите компанию, чтобы продолжить.',
  'no company associated with this user': 'У вашего пользователя нет привязанной компании — обратитесь к администратору.',
  'company access denied': 'Нет доступа к выбранной компании — обратитесь к администратору.',
  'insufficient permissions': 'Недостаточно прав для этого раздела.',
};

export function apiErrorLabel(err: unknown, fallback: string): string {
  const raw = apiError(err, '');
  if (!raw) return fallback;
  return API_ERROR_LABELS[raw] ?? `${fallback} (${raw})`;
}
