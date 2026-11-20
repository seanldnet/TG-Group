<script setup lang="ts">
import { Alert, Card, Table, Tag } from 'ant-design-vue';
import { Page } from '@vben/common-ui';

const columns = [
  { title: '订单号', dataIndex: 'orderNo', key: 'orderNo' },
  { title: '购买方', dataIndex: 'buyer', key: 'buyer' },
  { title: '订单类型', dataIndex: 'kind', key: 'kind' },
  { title: '金额', dataIndex: 'amount', key: 'amount' },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
  { title: '状态', dataIndex: 'status', key: 'status' },
];
const rows = [
  { key: '1', orderNo: 'TG-DEMO-001', buyer: '演示商户 A', kind: '商户套餐', amount: '299 USDT', createdAt: '2026-11-21 09:30', status: '已支付' },
  { key: '2', orderNo: 'TG-DEMO-002', buyer: '演示代理 B', kind: '代理资格', amount: '999 USDT', createdAt: '2026-11-21 09:45', status: '待支付' },
];
</script>

<template>
  <Page title="订单管理" description="查询商户套餐、代理资格与充值订单">
    <section class="flex flex-col gap-4">
      <Alert
        message="订单列表为演示数据"
        description="当前不执行支付、退款或订单状态变更；接入真实订单服务后再开放业务操作。"
        type="info"
        show-icon
      />
      <Card title="全部订单">
        <Table :columns="columns" :data-source="rows" :pagination="false" :scroll="{ x: 900 }" size="middle">
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
