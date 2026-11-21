import { requestClient } from '#/api/request';

export type OrderStatus = 'pending' | 'paid' | 'failed' | 'cancelled' | 'refunded';

export interface OrderListItem {
  orderNo: string;
  botName: string;
  telegramUserId: number | null;
  amountMinor: number;
  currency: string;
  status: OrderStatus;
  createdAt: string;
}

export interface OrderListResult {
  items: OrderListItem[];
  total: number;
  page: number;
  pageSize: number;
}

export interface OrderListParams {
  botId?: number;
  status?: OrderStatus;
  q?: string;
  page: number;
  pageSize: number;
}

export function getOrdersApi(params: OrderListParams) {
  return requestClient.get<OrderListResult>('/orders', { params });
}
