<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { Alert, Card, Empty, Statistic, Table, Tag } from 'ant-design-vue';
import { Page } from '@vben/common-ui';

import {
  getDashboardSummaryApi,
  type DashboardOrder,
  type DashboardSummary,
} from '#/api/core/dashboard';

const summary = ref<DashboardSummary | null>(null);
const loading = ref(false);
const errorMessage = ref('');

const columns = [
  { title: '机器人', dataIndex: 'botName', key: 'botName' },
  { title: '订单号', dataIndex: 'orderNo', key: 'orderNo' },
  { title: 'Telegram 用户 ID', dataIndex: 'telegramUserId', key: 'telegramUserId' },
  { title: '金额（最小单位）', dataIndex: 'amountMinor', key: 'amountMinor' },
  { title: '币种', dataIndex: 'currency', key: 'currency' },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
  { title: '状态', dataIndex: 'status', key: 'status' },
];

const statusLabels: Record<DashboardOrder['status'], string> = {
  pending: '待支付',
  paid: '已支付',
  failed: '失败',
  cancelled: '已取消',
  refunded: '已退款',
};

const statusColors: Record<DashboardOrder['status'], string> = {
  pending: 'processing',
  paid: 'success',
  failed: 'error',
  cancelled: 'default',
  refunded: 'warning',
};

function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

async function loadDashboard() {
  loading.value = true;
  errorMessage.value = '';
  try {
    summary.value = await getDashboardSummaryApi();
  } catch {
    errorMessage.value = '运营数据加载失败，请检查服务连接后重试。';
  } finally {
    loading.value = false;
  }
}

onMounted(loadDashboard);
</script>

<template>
  <Page title="运营总览" description="单一平台、多机器人运行状态与全局运营数据">
    <section class="flex flex-col gap-4">
      <Alert
        v-if="errorMessage"
        :message="errorMessage"
        type="error"
        show-icon
        closable
        @close="errorMessage = ''"
      />
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card>
          <Statistic title="机器人总数" :value="summary?.totalBots ?? 0" suffix="个" />
        </Card>
        <Card>
          <Statistic title="运行中 Bot" :value="summary?.activeBots ?? 0" suffix="个" />
        </Card>
        <Card>
          <Statistic title="管理群组" :value="summary?.totalGroups ?? 0" suffix="个" />
        </Card>
        <Card>
          <Statistic title="群成员记录" :value="summary?.totalMembers ?? 0" suffix="条" />
        </Card>
      </div>
      <Card title="近期订单">
        <Table
          :columns="columns"
          :data-source="summary?.recentOrders ?? []"
          :loading="loading"
          :pagination="false"
          :scroll="{ x: 1100 }"
          row-key="orderNo"
          size="middle"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'createdAt'">
              {{ formatDate(record.createdAt) }}
            </template>
            <Tag v-else-if="column.key === 'status'" :color="statusColors[record.status as DashboardOrder['status']]">
              {{ statusLabels[record.status as DashboardOrder['status']] ?? record.status }}
            </Tag>
          </template>
          <template #emptyText>
            <Empty description="暂无订单记录" />
          </template>
        </Table>
      </Card>
    </section>
  </Page>
</template>
