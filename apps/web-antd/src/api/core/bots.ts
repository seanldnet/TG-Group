import { requestClient } from '#/api/request';

export interface BotSummary {
  id: number;
  name: string;
  username: string;
  status: 'active' | 'inactive' | 'error';
  groupCount: number;
  createdAt: string;
}

export function getBotsApi() {
  return requestClient.get<BotSummary[]>('/bots');
}
