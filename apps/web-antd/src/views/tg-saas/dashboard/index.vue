<script setup lang="ts">
import { Alert, Card, Statistic, Table, Tag } from 'ant-design-vue';
import { Page } from '@vben/common-ui';

const columns = [
  { title: '机器人', dataIndex: 'botName', key: 'botName' },
  { title: '订单号', dataIndex: 'orderNo', key: 'orderNo' },
  { title: '订单类型', dataIndex: 'kind', key: 'kind' },
  { title: '金额', dataIndex: 'amount', key: 'amount' },
  { title: '状态', dataIndex: 'status', key: 'status' },
];
const rows = [
  { key: '1', botName: '社区管理机器人', orderNo: 'TG-DEMO-001', kind: '平台服务', amount: '29 USDT', status: '已支付' },
  { key: '2', botName: '活动运营机器人', orderNo: 'TG-DEMO-002', kind: '功能服务', amount: '9 USDT', status: '待支付' },
];
</script>

<template>
  <Page title="运营总览" description="单一平台、多机器人运行状态与全局运营数据">
    <section class="flex flex-col gap-4">
      <Alert
        message="阶段一界面预览"
        description="以下指标与订单均为演示数据，不代表真实运营数据。当前尚未接入 Bot、订单或统计 API。"
        type="info"
        show-icon
      />
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <Card><Statistic title="机器人总数" :value="2" suffix="个" /></Card>
        <Card><Statistic title="运行中 Bot" :value="1" suffix="个" /></Card>
        <Card><Statistic title="管理群组" :value="11" suffix="个" /></Card>
        <Card><Statistic title="今日消息" :value="1688" suffix="条" /></Card>
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
