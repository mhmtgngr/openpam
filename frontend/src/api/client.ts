import axios, { AxiosError, AxiosResponse } from 'axios';
import toast from 'react-hot-toast';
import type { ApiError } from '@/types';

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

// Response interceptor
client.interceptors.response.use(
  (response: AxiosResponse) => response,
  async (error: AxiosError<ApiError>) => {
    const originalRequest = error.config as AxiosResponse & { _retry?: boolean };

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

// Generic API helpers
export const api = {
  get: <T>(url: string, params?: unknown) => client.get<T>(url, { params }),
  post: <T>(url: string, data?: unknown) => client.post<T>(url, data),
  put: <T>(url: string, data?: unknown) => client.put<T>(url, data),
  patch: <T>(url: string, data?: unknown) => client.patch<T>(url, data),
  delete: <T>(url: string) => client.delete<T>(url),
};

export default client;
