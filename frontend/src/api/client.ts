import axios from 'axios';

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

export function setAuthFailureHandler(handler: () => void) {
  onAuthFailure = handler;
}

function handleAuthFailure() {
  localStorage.removeItem('auth-tokens');
  localStorage.removeItem('auth-user');
  localStorage.removeItem('auth-storage');
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
  try {
    const raw = localStorage.getItem('auth-storage');
    if (raw) {
      const parsed = JSON.parse(raw);
      const token = parsed?.state?.tokens?.access_token;
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
    }
  } catch {
    // ignore parse errors
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

    let refreshToken: string | null = null;
    try {
      const raw = localStorage.getItem('auth-storage');
      if (raw) {
        refreshToken = JSON.parse(raw)?.state?.tokens?.refresh_token ?? null;
      }
    } catch {
      // ignore
    }

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
      // Update zustand persisted store
      try {
        const raw = localStorage.getItem('auth-storage');
        if (raw) {
          const stored = JSON.parse(raw);
          stored.state.tokens = newTokens;
          localStorage.setItem('auth-storage', JSON.stringify(stored));
        }
      } catch {
        // ignore
      }
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
