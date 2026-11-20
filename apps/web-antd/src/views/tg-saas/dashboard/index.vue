<script setup lang="ts">
import { Alert, Card, Statistic, Table, Tag } from 'ant-design-vue';
import { Page } from '@vben/common-ui';

const columns = [
  { title: '订单号', dataIndex: 'orderNo', key: 'orderNo' },
  { title: '商户', dataIndex: 'merchant', key: 'merchant' },
  { title: '订单类型', dataIndex: 'kind', key: 'kind' },
  { title: '金额', dataIndex: 'amount', key: 'amount' },
  { title: '状态', dataIndex: 'status', key: 'status' },
];
const rows = [
  { key: '1', orderNo: 'TG-DEMO-001', merchant: '演示商户 A', kind: '商户套餐', amount: '299 USDT', status: '已支付' },
  { key: '2', orderNo: 'TG-DEMO-002', merchant: '演示商户 B', kind: '代理资格', amount: '999 USDT', status: '待支付' },
];
</script>

<template>
  <Page title="仪表盘" description="平台商户、机器人与交易概览">
    <section class="flex flex-col gap-4">
      <Alert
        message="阶段一界面预览"
        description="以下指标与订单均为演示数据，不代表真实运营数据。业务 API 将在后续阶段接入。"
        type="info"
        show-icon
      />
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card><Statistic title="商户总数" :value="128" suffix="家" /></Card>
        <Card><Statistic title="运行中 Bot" :value="96" suffix="个" /></Card>
        <Card><Statistic title="今日订单" :value="342" suffix="笔" /></Card>
        <Card><Statistic title="今日收入" :value="12860" suffix="USDT" /></Card>
      </div>
      <Card title="近期订单">
        <Table :columns="columns" :data-source="rows" :pagination="false" size="middle">
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
