import axios, { AxiosError, AxiosResponse } from 'axios';
import toast from 'react-hot-toast';
import type { ApiError, UsageLimitError } from '@/types';

const API_BASE = (import.meta as unknown as { env: { VITE_API_URL?: string } }).env.VITE_API_URL || 'http://localhost:8500/api/v1';

const client = axios.create({
  baseURL: API_BASE,
  headers: { 'Content-Type': 'application/json' },
});

// Request ID for correlation
let requestId = 0;
const getRequestId = () => {
  return `req_${Date.now()}_${++requestId}`;
};

// Request interceptor
client.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('access_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    config.headers['X-Request-ID'] = getRequestId();
    return config;
  },
  (error) => Promise.reject(error)
);

// Check if error is a usage limit error
const isUsageLimitError = (error: AxiosError<ApiError | UsageLimitError>): boolean => {
  const code = error.response?.data?.error?.code;
  return code === '1308' ||
    code === 'USAGE_LIMIT_REACHED' ||
    code === 'RATE_LIMIT_EXCEEDED' ||
    error.response?.status === 429;
};

// Handle usage limit error
const handleUsageLimitError = (error: AxiosError<ApiError | UsageLimitError>) => {
  const data = error.response?.data as UsageLimitError;
  const resetAt = data?.error?.details?.reset_at || extractResetTime(data?.error?.message) || '';

  // Store error data for usage limit page
  localStorage.setItem('usage_limit_code', data?.error?.code || '1308');
  localStorage.setItem('usage_limit_message', data?.error?.message || 'Usage limit reached');
  if (resetAt) {
    localStorage.setItem('usage_limit_reset_at', resetAt);
  }
  if (data?.request_id) {
    localStorage.setItem('usage_limit_request_id', data.request_id);
  }

  // Redirect to usage limit page
  window.location.href = '/usage-limit';
};

// Extract reset time from error message (fallback)
const extractResetTime = (message?: string): string | null => {
  if (!message) return null;
  const match = message.match(/(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})/);
  return match ? match[1] : null;
};

// Response interceptor
client.interceptors.response.use(
  (response: AxiosResponse) => response,
  async (error: AxiosError<ApiError | UsageLimitError>) => {
    const originalRequest = error.config as AxiosResponse & { _retry?: boolean };

    // Handle usage limit errors
    if (isUsageLimitError(error)) {
      handleUsageLimitError(error);
      return Promise.reject(error);
    }

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      // Try to refresh token
      const refreshToken = localStorage.getItem('refresh_token');
      if (refreshToken) {
        try {
          const response = await axios.post<{ access_token: string }>(
            `${API_BASE}/auth/refresh`,
            { refresh_token: refreshToken }
          );
          localStorage.setItem('access_token', response.data.access_token);
          originalRequest.headers.Authorization = `Bearer ${response.data.access_token}`;
          return client(originalRequest);
        } catch {
          localStorage.removeItem('access_token');
          localStorage.removeItem('refresh_token');
          window.location.href = '/login';
          return Promise.reject(error);
        }
      } else {
        window.location.href = '/login';
      }
    }

    // Handle error messages
    const errorMessage = error.response?.data?.error?.message || 'An error occurred';
    const errorCode = error.response?.data?.error?.code;

    if (errorCode !== 'NETWORK_ERROR') {
      toast.error(errorMessage);
    }

    return Promise.reject(error);
  }
);

// Generic API helpers - unwrap response.data for cleaner API
export const api = {
  get: <T>(url: string, params?: unknown) =>
    client.get<T>(url, { params }).then((res) => res.data),
  post: <T>(url: string, data?: unknown) =>
    client.post<T>(url, data).then((res) => res.data),
  put: <T>(url: string, data?: unknown) =>
    client.put<T>(url, data).then((res) => res.data),
  patch: <T>(url: string, data?: unknown) =>
    client.patch<T>(url, data).then((res) => res.data),
  delete: <T>(url: string) =>
    client.delete<T>(url).then((res) => res.data),
};

export default client;
