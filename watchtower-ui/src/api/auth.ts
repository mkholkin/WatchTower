import api from './axios';
export const mfaEnabled = process.env.REACT_APP_AUTH_MFA === 'true';
export const register = (login: string, password: string, email?: string) =>
  api.post(mfaEnabled ? '/auth/mfa/register' : '/auth/register', mfaEnabled ? { login, password, email } : { login, password });
export const login = (login: string, password: string) => api.post(mfaEnabled ? '/auth/mfa/login' : '/auth/login', { login, password });
export const verify = (challenge_id: string, code: string) => api.post('/auth/mfa/verify', { challenge_id, code });
export const recover = (login: string) => api.post('/auth/mfa/recover', { login });
export const reset = (login: string, token: string, new_password: string) => api.post('/auth/mfa/reset', { login, token, new_password });
export const changePassword = (token: string, current_password: string, new_password: string) => api.post('/auth/mfa/password', { current_password, new_password }, { headers: { Authorization: `Bearer ${token}` } });
export function authError(err: any): string {
  switch (err.response?.status) {
    case 401: return 'Incorrect credentials or verification code.';
    case 410: return 'Code expired. Start again to request a new one.';
    case 423: return 'Account locked. Recover your account to continue.';
    case 409: return 'This username is already registered.';
    case 400: return 'Check your input. Passwords must contain 1–72 UTF-8 bytes and a new password must be different.';
    case 503: return 'Email delivery is unavailable. Please try again later.';
    default: return 'Unable to complete the request. Please try again.';
  }
}
