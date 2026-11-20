import { baseRequestClient, requestClient } from '#/api/request';

export namespace AuthApi {
  export interface LoginParams {
    password: string;
    username: string;
  }

  export interface LoginResult {
    accessToken: string;
    expiresAt: string;
  }

  export interface PasswordChangeParams {
    currentPassword: string;
    newPassword: string;
  }
}

export async function loginApi(data: AuthApi.LoginParams) {
  return requestClient.post<AuthApi.LoginResult>('/auth/login', data);
}

export async function refreshTokenApi() {
  return baseRequestClient.post('/auth/refresh', undefined, {
    withCredentials: true,
  });
}

export async function logoutApi() {
  // Use requestClient so the active Bearer token is attached to the revoke request.
  return requestClient.post('/auth/logout');
}

export async function changePasswordApi(data: AuthApi.PasswordChangeParams) {
  return requestClient.put('/auth/password', data);
}

export async function getAccessCodesApi() {
  return requestClient.get<string[]>('/auth/codes');
}
