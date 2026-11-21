import { requestClient } from '#/api/request';

export interface GroupModerationRules {
  groupId: number;
  blockLinks: boolean;
  blockedKeywords: string[];
  updatedAt?: string;
}

export function getGroupRulesApi(groupId: number) {
  return requestClient.get<GroupModerationRules>(`/groups/${groupId}/rules`);
}

export function saveGroupRulesApi(
  groupId: number,
  data: Pick<GroupModerationRules, 'blockLinks' | 'blockedKeywords'>,
) {
  return requestClient.put<GroupModerationRules>(`/groups/${groupId}/rules`, data);
}
