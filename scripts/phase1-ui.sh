#!/usr/bin/env bash
set -euo pipefail

mkdir -p src/components/PageContainer src/views/tg-saas/dashboard src/views/tg-saas/orders src/views/tg-saas/settings src/views/tg-saas/agents src/views/tg-saas/payments

cat > src/components/PageContainer/index.vue <<'EOF'
<template>
  <PageWrapper :title="title" dense>
    <section class="flex flex-col gap-4">
      <slot />
    </section>
  </PageWrapper>
</template>
<script lang="ts" setup>
  import { PageWrapper } from '@/components/Page';
  defineProps<{ title: string }>();
</script>
EOF

cat > src/views/tg-saas/dashboard/index.vue <<'EOF'
<template>
  <PageContainer title="仪表盘">
    <Alert message="平台运营总览" description="当前展示为阶段一 UI 骨架数据，真实商户、订单与 Bot 指标将在后续阶段接入。" type="info" show-icon />
    <section class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
      <Card title="商户总数"><Statistic :value="128" suffix="家" /></Card>
      <Card title="运行中 Bot"><Statistic :value="96" suffix="个" /></Card>
      <Card title="今日订单"><Statistic :value="342" suffix="笔" /></Card>
      <Card title="今日收入"><Statistic :value="12860" suffix="USDT" /></Card>
    </section>
    <Card title="近期订单"><BasicTable @register="registerTable" /></Card>
  </PageContainer>
</template>
<script lang="ts" setup>
  import { Alert, Card, Statistic } from 'ant-design-vue';
  import { BasicTable, useTable } from '@/components/Table';
  import PageContainer from '@/components/PageContainer/index.vue';
  const [registerTable] = useTable({
    columns: [
      { title: '订单号', dataIndex: 'orderNo', width: 190 },
      { title: '商户', dataIndex: 'merchant' },
      { title: '套餐', dataIndex: 'plan' },
      { title: '金额', dataIndex: 'amount', customRender: ({ text }) => String(text) + ' USDT' },
      { title: '状态', dataIndex: 'status' },
    ],
    dataSource: [
      { orderNo: 'TG202611210001', merchant: 'Demo Shop A', plan: '专业版', amount: 299, status: '已支付' },
      { orderNo: 'TG202611210002', merchant: 'Demo Shop B', plan: '基础版', amount: 99, status: '待支付' },
    ],
    pagination: false,
    showIndexColumn: false,
  });
</script>
EOF

cat > src/views/tg-saas/orders/index.vue <<'EOF'
<template>
  <PageContainer title="订单管理">
    <Alert message="订单管理 UI 骨架" description="当前为演示记录；订单状态、支付回调与退款操作将在后续业务阶段接入。" type="info" show-icon />
    <Card title="订单列表"><BasicTable @register="registerTable" /></Card>
  </PageContainer>
</template>
<script lang="ts" setup>
  import { Alert, Card } from 'ant-design-vue';
  import { BasicTable, useTable } from '@/components/Table';
  import PageContainer from '@/components/PageContainer/index.vue';
  const [registerTable] = useTable({
    columns: [
      { title: '订单号', dataIndex: 'orderNo', width: 190 },
      { title: '购买方', dataIndex: 'buyer' },
      { title: '订单类型', dataIndex: 'kind' },
      { title: '金额', dataIndex: 'amount', customRender: ({ text }) => String(text) + ' USDT' },
      { title: '创建时间', dataIndex: 'createdAt' },
      { title: '状态', dataIndex: 'status' },
    ],
    dataSource: [
      { orderNo: 'TG202611210001', buyer: 'Demo Shop A', kind: '商户套餐', amount: 299, createdAt: '2026-11-21 09:30', status: '已支付' },
      { orderNo: 'TG202611210002', buyer: 'Demo Agent B', kind: '代理资格', amount: 999, createdAt: '2026-11-21 09:45', status: '待支付' },
    ],
    pagination: false,
    showIndexColumn: false,
  });
</script>
EOF

cat > src/views/tg-saas/settings/index.vue <<'EOF'
<template>
  <PageContainer title="平台设置">
    <Alert message="平台基础设置" description="本页仅建立 Vben BasicForm 的布局和字段规范，保存接口将在后续阶段实现。" type="info" show-icon />
    <Card title="基础配置"><BasicForm @register="registerForm" /></Card>
    <Card title="安全策略"><BasicForm @register="registerSecurityForm" /></Card>
  </PageContainer>
</template>
<script lang="ts" setup>
  import { Alert, Card } from 'ant-design-vue';
  import { BasicForm, useForm } from '@/components/Form';
  import PageContainer from '@/components/PageContainer/index.vue';
  const [registerForm] = useForm({
    labelWidth: 140,
    showActionButtonGroup: false,
    schemas: [
      { field: 'platformName', label: '平台名称', component: 'Input', defaultValue: 'TG Group SaaS', colProps: { span: 12 } },
      { field: 'supportUrl', label: '客服地址', component: 'Input', componentProps: { placeholder: 'https://...' }, colProps: { span: 12 } },
      { field: 'defaultCurrency', label: '默认计价单位', component: 'Input', defaultValue: 'USDT', componentProps: { disabled: true }, colProps: { span: 12 } },
    ],
  });
  const [registerSecurityForm] = useForm({
    labelWidth: 140,
    showActionButtonGroup: false,
    schemas: [
      { field: 'sessionTimeout', label: '会话超时（分钟）', component: 'InputNumber', defaultValue: 60, componentProps: { min: 5, precision: 0 }, colProps: { span: 12 } },
      { field: 'loginCaptcha', label: '登录验证码', component: 'Switch', defaultValue: true, colProps: { span: 12 } },
    ],
  });
</script>
EOF

cat > src/views/tg-saas/agents/index.vue <<'EOF'
<template>
  <PageContainer title="代理管理">
    <Alert message="代理资格采用付费开通" description="代理购买、上下级关系、折扣权限和商户数据隔离将在阶段二实现；本页不提供免费申请入口。" type="info" show-icon />
    <Card title="代理列表">
      <BasicTable @register="registerTable" />
      <div class="flex flex-wrap gap-4">
        <Button type="primary" @click="modalOpen = true">新建代理套餐（UI）</Button>
        <Button @click="drawerOpen = true">查看代理规则</Button>
      </div>
    </Card>
    <BasicModal v-model:open="modalOpen" title="代理资格套餐（界面预览）">
      <p>正式创建代理资格必须通过整数 USDT 支付成功回调，不允许后台直接免费开通。</p>
    </BasicModal>
    <BasicDrawer v-model:open="drawerOpen" title="代理折扣与加价规则">
      <p>代理只能配置下级商户价格；自身折扣由上级代理或平台确定。</p>
    </BasicDrawer>
  </PageContainer>
</template>
<script lang="ts" setup>
  import { ref } from 'vue';
  import { Alert, Button, Card } from 'ant-design-vue';
  import { BasicTable, useTable } from '@/components/Table';
  import { BasicModal } from '@/components/Modal';
  import { BasicDrawer } from '@/components/Drawer';
  import PageContainer from '@/components/PageContainer/index.vue';
  const modalOpen = ref(false);
  const drawerOpen = ref(false);
  const [registerTable] = useTable({
    columns: [
      { title: '代理名称', dataIndex: 'name' },
      { title: '代理等级', dataIndex: 'level' },
      { title: '名下商户', dataIndex: 'merchants' },
      { title: '资格价格', dataIndex: 'price', customRender: ({ text }) => String(text) + ' USDT' },
      { title: '状态', dataIndex: 'status' },
    ],
    dataSource: [
      { name: 'Demo 一级代理', level: '一级', merchants: 12, price: 999, status: '演示数据' },
      { name: 'Demo 二级代理', level: '二级', merchants: 4, price: 499, status: '演示数据' },
    ],
    pagination: false,
    showIndexColumn: false,
  });
</script>
EOF

cat > src/views/tg-saas/payments/index.vue <<'EOF'
<template>
  <PageContainer title="支付网关配置">
    <Alert message="全站统一使用整数 USDT 面值" description="数据库价格与订单金额保存整数 USDT；链上监听与验账使用 6 位精度最小单位整数。当前仅为配置页面骨架，不会发起真实收款。" type="warning" show-icon />
    <section class="grid grid-cols-1 gap-4 xl:grid-cols-2">
      <Card title="GMPay（EPUSDT）"><BasicForm @register="registerGmpay" /></Card>
      <Card title="EDUSDT"><BasicForm @register="registerEdusdt" /></Card>
      <Card title="内置 USDT 支付"><BasicForm @register="registerNative" /></Card>
      <Card title="金额精度规则">
        <Alert message="1 USDT = 1,000,000 链上最小单位" description="套餐、代理折扣、订单与佣金金额均使用整数 USDT 面值；链上最小单位只用于支付网关边界换算。" type="info" show-icon />
      </Card>
    </section>
  </PageContainer>
</template>
<script lang="ts" setup>
  import { Alert, Card } from 'ant-design-vue';
  import { BasicForm, useForm } from '@/components/Form';
  import PageContainer from '@/components/PageContainer/index.vue';
  const commonSchemas = [
    { field: 'enabled', label: '启用网关', component: 'Switch', defaultValue: false, colProps: { span: 24 } },
    { field: 'minimumAmount', label: '最低收款金额', component: 'InputNumber', defaultValue: 1, componentProps: { min: 1, precision: 0, addonAfter: 'USDT' }, colProps: { span: 24 } },
  ];
  const [registerGmpay] = useForm({
    labelWidth: 130,
    showActionButtonGroup: false,
    schemas: [
      ...commonSchemas,
      { field: 'apiToken', label: 'API Token', component: 'InputPassword', componentProps: { placeholder: '配置后由服务端安全保存' }, colProps: { span: 24 } },
      { field: 'feeRate', label: '费率', component: 'InputNumber', defaultValue: 0, componentProps: { min: 0, precision: 0, addonAfter: '整数' }, colProps: { span: 24 } },
    ],
  });
  const [registerEdusdt] = useForm({
    labelWidth: 130,
    showActionButtonGroup: false,
    schemas: [
      ...commonSchemas,
      { field: 'merchantId', label: '商户标识', component: 'Input', colProps: { span: 24 } },
      { field: 'apiKey', label: 'API Key', component: 'InputPassword', colProps: { span: 24 } },
    ],
  });
  const [registerNative] = useForm({
    labelWidth: 130,
    showActionButtonGroup: false,
    schemas: [
      ...commonSchemas,
      { field: 'network', label: '收款网络', component: 'Select', componentProps: { options: [{ label: 'TRON (TRC20)', value: 'TRON' }] }, colProps: { span: 24 } },
      { field: 'walletAddress', label: '收款地址', component: 'Input', colProps: { span: 24 } },
    ],
  });
</script>
EOF

cat > src/router/routes/modules/tg-saas.ts <<'EOF'
import type { AppRouteModule } from '@/router/types';
import { LAYOUT } from '@/router/constant';

const tgSaas: AppRouteModule = {
  path: '/tg-saas',
  name: 'TgSaas',
  component: LAYOUT,
  redirect: '/tg-saas/dashboard',
  meta: { orderNo: 1, icon: 'ion:server-outline', title: 'TG SaaS' },
  children: [
    { path: 'dashboard', name: 'TgSaasDashboard', component: () => import('@/views/tg-saas/dashboard/index.vue'), meta: { title: '仪表盘' } },
    { path: 'orders', name: 'TgSaasOrders', component: () => import('@/views/tg-saas/orders/index.vue'), meta: { title: '订单管理' } },
    { path: 'settings', name: 'TgSaasSettings', component: () => import('@/views/tg-saas/settings/index.vue'), meta: { title: '设置' } },
    { path: 'agents', name: 'TgSaasAgents', component: () => import('@/views/tg-saas/agents/index.vue'), meta: { title: '代理管理' } },
    { path: 'payments', name: 'TgSaasPayments', component: () => import('@/views/tg-saas/payments/index.vue'), meta: { title: '支付网关配置' } },
  ],
};

export default tgSaas;
EOF
