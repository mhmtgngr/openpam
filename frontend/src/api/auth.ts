import { api } from './client';
import type {
  AuthResponse,
  User,
  MFADevice,
  TOTPSetup,
} from '@/types';

export const authApi = {
  // Login
  login: (email: string, password: string, mfaCode?: string) =>
    api.post<AuthResponse>('/auth/login', { email, password, mfa_code: mfaCode }),

  // Logout
  logout: () => api.post<void>('/auth/logout'),

  // Refresh token
  refreshToken: (refreshToken: string) =>
    api.post<{ access_token: string }>('/auth/refresh', { refresh_token: refreshToken }),

  // Get current user
  me: () => api.get<User>('/auth/me'),

  // MFA - Setup TOTP
  setupTOTP: () => api.post<TOTPSetup>('/auth/mfa/totp/setup'),

  // MFA - Verify and enable TOTP
  verifyTOTP: (secret: string, code: string) =>
    api.post<{ devices: MFADevice[] }>('/auth/mfa/totp/verify', { secret, code }),

  // MFA - Disable TOTP
  disableTOTP: (code: string) => api.post<void>('/auth/mfa/totp/disable', { code }),

  // MFA - List devices
  listMFADevices: () => api.get<{ devices: MFADevice[] }>('/auth/mfa/devices'),

  // MFA - Delete device
  deleteMFADevice: (deviceId: string) => api.delete<void>(`/auth/mfa/devices/${deviceId}`),

  // MFA - Begin WebAuthn registration
  beginWebAuthnRegistration: () =>
    api.post<{ credential_creation_options: unknown }>('/auth/mfa/webauthn/register/begin'),

  // MFA - Finish WebAuthn registration
  finishWebAuthnRegistration: (credential: unknown) =>
    api.post<{ devices: MFADevice[] }>('/auth/mfa/webauthn/register/finish', { credential }),

  // Password change
  changePassword: (oldPassword: string, newPassword: string) =>
    api.post<void>('/auth/password/change', { old_password: oldPassword, new_password: newPassword }),
};
