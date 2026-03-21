import type { Role } from './models';

export interface AuthUser {
  id: string;
  email: string;
  role: Role;
  company_id?: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  user: AuthUser;
  access_token: string;
  refresh_token: string;
}

export type { Role };
