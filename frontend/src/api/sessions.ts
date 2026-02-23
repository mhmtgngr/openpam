import { api } from './client';
import type { Session, SessionEvent, PaginatedResponse } from '@/types';

export interface SessionListParams {
  limit?: number;
  offset?: number;
  search?: string;
  status?: string;
  type?: string;
  user_id?: string;
  target_id?: string;
}

export const sessionsApi = {
  // List sessions
  list: (params?: SessionListParams) =>
    api.get<PaginatedResponse<Session>>('/sessions', params),

  // Get session by ID
  get: (id: string) => api.get<Session>(`/sessions/${id}`),

  // Terminate session
  terminate: (id: string, reason?: string) =>
    api.post<Session>(`/sessions/${id}/terminate`, { reason }),

  // Get session events
  getEvents: (id: string, params?: { limit?: number; offset?: number }) =>
    api.get<PaginatedResponse<SessionEvent>>(`/sessions/${id}/events`, params),

  // Get recording URL
  getRecordingUrl: (id: string) =>
    api.get<{ url: string; expires_at: string }>(`/sessions/${id}/recording`),

  // Get active sessions
  getActive: () => api.get<Session[]>('/sessions/active'),

  // Start session
  start: (targetId: string, credentialId?: string) =>
    api.post<{ session_id: string; websocket_url: string }>('/sessions/start', {
      target_id: targetId,
      credential_id: credentialId,
    }),
};
