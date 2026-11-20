<script setup lang="ts">
import { computed, ref } from 'vue';
import { Alert, Card, Select, Table, Tag } from 'ant-design-vue';
import { Page } from '@vben/common-ui';

const selectedBot = ref('all');
const botOptions = [
  { label: '全部机器人', value: 'all' },
  { label: '社区管理机器人', value: 'community' },
  { label: '活动运营机器人', value: 'events' },
];
const rows = [
  { key: '1', orderNo: 'TG-DEMO-001', botId: 'community', botName: '社区管理机器人', buyer: 'Telegram 用户 10001', kind: '平台服务', amount: '29 USDT', createdAt: '2026-11-21 09:30', status: '已支付' },
  { key: '2', orderNo: 'TG-DEMO-002', botId: 'events', botName: '活动运营机器人', buyer: 'Telegram 用户 10002', kind: '功能服务', amount: '9 USDT', createdAt: '2026-11-21 09:45', status: '待支付' },
];
const filteredRows = computed(() => selectedBot.value === 'all' ? rows : rows.filter((row) => row.botId === selectedBot.value));
const columns = [
  { title: '订单号', dataIndex: 'orderNo', key: 'orderNo' },
  { title: '所属机器人', dataIndex: 'botName', key: 'botName' },
  { title: 'Telegram 用户', dataIndex: 'buyer', key: 'buyer' },
  { title: '订单类型', dataIndex: 'kind', key: 'kind' },
  { title: '金额', dataIndex: 'amount', key: 'amount' },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
  { title: '状态', dataIndex: 'status', key: 'status' },
];
</script>

<template>
  <Page title="订单管理" description="统一查看平台订单，并按所属 Telegram Bot 筛选">
    <section class="flex flex-col gap-4">
      <Alert
        message="订单列表为演示数据"
        description="当前不执行支付、退款或订单状态变更；订单归属机器人筛选为前端演示，真实数据将在后续接入。"
        type="info"
        show-icon
      />
      <Card title="全部订单">
        <div class="flex flex-wrap items-center gap-3">
          <span class="text-sm">所属机器人</span>
          <Select v-model:value="selectedBot" :options="botOptions" class="min-w-52" />
        </div>
        <Table :columns="columns" :data-source="filteredRows" :pagination="false" :scroll="{ x: 1100 }" size="middle">
          <template #bodyCell="{ column, record }">
            <Tag v-if="column.key === 'status'" :color="record.status === '已支付' ? 'success' : 'processing'">
              {{ record.status }}
            </Tag>
          </template>
        </Table>
      </Card>
    </section>
  </Page>
</template>
