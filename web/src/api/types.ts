// DTOs mirror the grpc-gateway JSON (camelCase) of caliber.v1 IdentityService.

export type UserRole =
  | 'USER_ROLE_UNSPECIFIED'
  | 'USER_ROLE_EMPLOYER'
  | 'USER_ROLE_RECRUITER'
  | 'USER_ROLE_CANDIDATE';

export interface User {
  id: string;
  email: string;
  role: UserRole;
  name: string;
  createdAt: string;
}

export interface TokenPair {
  accessToken: string;
  refreshToken: string;
  accessExpiresIn: number;
}

export interface AuthResponse {
  user: User;
  tokens: TokenPair;
}

export interface RefreshResponse {
  tokens: TokenPair;
}

export interface MeResponse {
  user: User;
}

export interface LoginInput {
  email: string;
  password: string;
}

export interface RegisterInput {
  name: string;
  email: string;
  password: string;
  role: UserRole;
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}
