import { requestClient } from '#/api/request';

export interface BotRecord {
  id: number;
  name: string;
  username: string;
  status: 'active' | 'inactive' | 'error';
  groupCount: number;
  createdAt: string;
  tokenMasked?: string;
}

export interface CreateBotParams {
  name: string;
  token: string;
}

export function listBotsApi() {
  return requestClient.get<BotRecord[]>('/bots');
}

export function createBotApi(data: CreateBotParams) {
  return requestClient.post<BotRecord>('/bots', data);
}

export function renameBotApi(id: number, name: string) {
  return requestClient.request<{ id: number; name: string }>(`/bots/${id}`, { method: 'PATCH', data: { name } });
}

export function deleteBotApi(id: number) {
  return requestClient.delete<{ success: boolean }>(`/bots/${id}`);
}
