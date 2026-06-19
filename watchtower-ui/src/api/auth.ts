import api from './axios';

interface JWTTokenResponse {
  access_token: string;
}

interface UserCreatedResponse {
  login: string;
}

export const register = (login: string, password: string) =>
  api.post<UserCreatedResponse>('/auth/register', { login, password });

export const login = (login: string, password: string) =>
  api.post<JWTTokenResponse>('/auth/login', { login, password });
