<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Alert, Button, Card, Input, Select, Table, Tag, message } from 'ant-design-vue';
import { Page } from '@vben/common-ui';

import { getBotsApi, type BotSummary } from '#/api/core/bots';
import {
  getOrdersApi,
  type OrderListResult,
  type OrderStatus,
} from '#/api/core/orders';

const bots = ref<BotSummary[]>([]);
const orders = ref<OrderListResult>({ items: [], total: 0, page: 1, pageSize: 20 });
const selectedBot = ref('');
const selectedStatus = ref<OrderStatus | ''>('');
const keyword = ref('');
const loading = ref(false);
const errorMessage = ref('');

const botOptions = computed(() => [
  { label: '全部机器人', value: '' },
  ...bots.value.map((bot) => ({ label: bot.name, value: String(bot.id) })),
]);

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '待支付', value: 'pending' },
  { label: '已支付', value: 'paid' },
  { label: '失败', value: 'failed' },
  { label: '已取消', value: 'cancelled' },
  { label: '已退款', value: 'refunded' },
];

const statusLabels: Record<OrderStatus, string> = {
  pending: '待支付',
  paid: '已支付',
  failed: '失败',
  cancelled: '已取消',
  refunded: '已退款',
};

const statusColors: Record<OrderStatus, string> = {
  pending: 'processing',
  paid: 'success',
  failed: 'error',
  cancelled: 'default',
  refunded: 'warning',
};

const columns = [
  { title: '订单号', dataIndex: 'orderNo', key: 'orderNo', width: 190 },
  { title: '所属机器人', dataIndex: 'botName', key: 'botName', width: 170 },
  { title: 'Telegram 用户 ID', dataIndex: 'telegramUserId', key: 'telegramUserId', width: 160 },
  { title: '金额（数据库最小单位）', dataIndex: 'amountMinor', key: 'amountMinor', width: 190 },
  { title: '币种', dataIndex: 'currency', key: 'currency', width: 90 },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 190 },
  { title: '订单状态', dataIndex: 'status', key: 'status', width: 110 },
];

function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

async function loadBots() {
  try {
    bots.value = await getBotsApi();
  } catch {
    message.error('机器人筛选项加载失败，请刷新页面重试。');
  }
}

async function loadOrders() {
  loading.value = true;
  errorMessage.value = '';
  try {
    orders.value = await getOrdersApi({
      ...(selectedBot.value ? { botId: Number(selectedBot.value) } : {}),
      ...(selectedStatus.value ? { status: selectedStatus.value } : {}),
      ...(keyword.value.trim() ? { q: keyword.value.trim() } : {}),
      page: orders.value.page,
      pageSize: orders.value.pageSize,
    });
  } catch {
    errorMessage.value = '订单数据加载失败，请检查服务连接后重试。';
  } finally {
    loading.value = false;
  }
}

function searchOrders() {
  orders.value.page = 1;
  void loadOrders();
}

function resetFilters() {
  selectedBot.value = '';
  selectedStatus.value = '';
  keyword.value = '';
  orders.value.page = 1;
  void loadOrders();
}

function handleTableChange(pagination: { current?: number; pageSize?: number }) {
  orders.value.page = pagination.current ?? 1;
  orders.value.pageSize = pagination.pageSize ?? 20;
  void loadOrders();
}

onMounted(async () => {
  await Promise.all([loadBots(), loadOrders()]);
});
</script>

<template>
  <Page title="订单管理" description="统一查看真实订单数据，支持按 Telegram Bot、订单状态和订单号筛选">
    <section class="flex flex-col gap-4">
      <Alert
        v-if="errorMessage"
        :message="errorMessage"
        type="error"
        show-icon
        closable
        @close="errorMessage = ''"
      />
      <Card title="订单列表">
        <div class="flex flex-col gap-4">
          <div class="flex flex-wrap items-center gap-3">
          <Select
            v-model:value="selectedBot"
            :options="botOptions"
            class="min-w-52"
            placeholder="所属机器人"
          />
          <Select
            v-model:value="selectedStatus"
            :options="statusOptions"
            class="min-w-36"
            placeholder="订单状态"
          />
          <Input
            v-model:value="keyword"
            allow-clear
            class="w-64"
            placeholder="搜索订单号或 Telegram 用户 ID"
            @press-enter="searchOrders"
          />
          <Button type="primary" @click="searchOrders">查询</Button>
          <Button @click="resetFilters">重置</Button>
          </div>
          <Table
          :columns="columns"
          :data-source="orders.items"
          :loading="loading"
          :pagination="{
            current: orders.page,
            pageSize: orders.pageSize,
            total: orders.total,
            showSizeChanger: true,
          }"
          :scroll="{ x: 1100 }"
          row-key="orderNo"
          size="middle"
          @change="handleTableChange"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'createdAt'">
              {{ formatDate(record.createdAt) }}
            </template>
            <template v-else-if="column.key === 'amountMinor'">
              {{ record.amountMinor.toLocaleString() }}
            </template>
            <Tag v-else-if="column.key === 'status'" :color="statusColors[record.status]">
              {{ statusLabels[record.status] }}
            </Tag>
          </template>
          </Table>
        </div>
      </Card>
    </section>
  </Page>
</template>
