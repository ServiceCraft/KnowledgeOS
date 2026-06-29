import axios from 'axios';
import {
  AUTH_STORAGE_KEY,
  buildAuthHeaders,
  getAuthSnapshot,
  tenantContextError,
} from '@/lib/tenantContext';
import { useAuthStore } from '@/stores/authStore';

const client = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
});

let isRefreshing = false;
let failedQueue: Array<{
  resolve: (token: string) => void;
  reject: (err: unknown) => void;
}> = [];

let onAuthFailure: (() => void) | null = null;

export { tenantContextError };

export function setAuthFailureHandler(handler: () => void) {
  onAuthFailure = handler;
}

function handleAuthFailure() {
  localStorage.removeItem(AUTH_STORAGE_KEY);
  if (onAuthFailure) {
    onAuthFailure();
  }
}

function processQueue(error: unknown, token: string | null) {
  failedQueue.forEach((p) => {
    if (error) {
      p.reject(error);
    } else {
      p.resolve(token!);
    }
  });
  failedQueue = [];
}

client.interceptors.request.use((config) => {
  const tenantError = tenantContextError(config.url);
  if (tenantError) {
    return Promise.reject(tenantError);
  }
  const headers = buildAuthHeaders();
  if (headers.Authorization) {
    config.headers.Authorization = headers.Authorization;
  }
  if (headers['X-Company-ID']) {
    config.headers['X-Company-ID'] = headers['X-Company-ID'];
  }
  return config;
});

client.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response?.status !== 401 || originalRequest._retry) {
      return Promise.reject(error);
    }

    if (isRefreshing) {
      return new Promise((resolve, reject) => {
        failedQueue.push({ resolve, reject });
      }).then((token) => {
        originalRequest.headers.Authorization = `Bearer ${token}`;
        return client(originalRequest);
      });
    }

    originalRequest._retry = true;
    isRefreshing = true;

    const refreshToken = getAuthSnapshot().tokens?.refresh_token ?? null;

    if (!refreshToken) {
      handleAuthFailure();
      return Promise.reject(error);
    }

    try {
      const resp = await axios.post('/api/v1/auth/refresh', {
        refresh_token: refreshToken,
      });
      const payload = resp.data.data;
      const newTokens = {
        access_token: payload.access_token,
        refresh_token: payload.refresh_token,
      };
      useAuthStore.getState().setTokens(newTokens);
      processQueue(null, payload.access_token);
      originalRequest.headers.Authorization = `Bearer ${payload.access_token}`;
      return client(originalRequest);
    } catch (refreshError) {
      processQueue(refreshError, null);
      handleAuthFailure();
      return Promise.reject(refreshError);
    } finally {
      isRefreshing = false;
    }
  }
);

export default client;
