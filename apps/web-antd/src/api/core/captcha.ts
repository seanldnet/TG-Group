import { requestClient } from '#/api/request';

export interface CapSettings {
  enabled: boolean;
  mode: 'checkbox' | 'floating' | 'invisible';
  domain: string;
  siteKey: string;
  secretConfigured: boolean;
}

export interface CapSettingsInput {
  enabled: boolean;
  mode: 'checkbox' | 'floating' | 'invisible';
  domain: string;
  siteKey: string;
  secret?: string;
}

export function getCapSettingsApi() {
  return requestClient.get<CapSettings>('/settings/captcha');
}

export function updateCapSettingsApi(data: CapSettingsInput) {
  return requestClient.put<CapSettings>('/settings/captcha', data);
}
