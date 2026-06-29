import type { AxiosError } from 'axios';

export function apiError(err: unknown, fallback: string): string {
  const ax = err as AxiosError<{ error?: string }>;
  return ax?.response?.data?.error ?? fallback;
}
