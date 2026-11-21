import { requestClient } from '#/api/request';

export interface GroupListItem {
  id: number;
  botId: number;
  botName: string;
  telegramChatId: number;
  title: string;
  memberCount: number;
  createdAt: string;
}

export interface GroupListResult {
  items: GroupListItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface GroupListParams {
  botId?: number;
  q?: string;
  page: number;
  pageSize: number;
}

export function getGroupsApi(params: GroupListParams) {
  return requestClient.get<GroupListResult>('/groups', { params });
}
