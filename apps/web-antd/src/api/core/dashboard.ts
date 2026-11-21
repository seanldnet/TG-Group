import { requestClient } from '#/api/request';

export interface DashboardOrder {
  orderNo: string;
  botName: string;
  telegramUserId: number | null;
  amountMinor: number;
  currency: string;
  status: 'pending' | 'paid' | 'failed' | 'cancelled' | 'refunded';
  createdAt: string;
}

export interface DashboardSummary {
  totalBots: number;
  activeBots: number;
  totalGroups: number;
  totalMembers: number;
  recentOrders: DashboardOrder[];
}

export function getDashboardSummaryApi() {
  return requestClient.get<DashboardSummary>('/dashboard/summary');
}
