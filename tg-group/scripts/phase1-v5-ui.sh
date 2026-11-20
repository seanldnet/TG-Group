#!/usr/bin/env bash
set -euo pipefail

APP="apps/web-antd"
mkdir -p \
  "$APP/src/router/routes/modules" \
  "$APP/src/views/tg-saas/dashboard" \
  "$APP/src/views/tg-saas/orders" \
  "$APP/src/views/tg-saas/settings" \
  "$APP/src/views/tg-saas/agents" \
  "$APP/src/views/tg-saas/payments"

cat > "$APP/src/router/routes/modules/tg-saas.ts" <<'EOF'
import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    name: 'TgSaas',
    path: '/tg-saas',
    meta: {
      icon: 'lucide:bot',
      order: 1,
      title: 'TG SaaS',
    },
    children: [
      {
        name: 'TgSaasDashboard',
        path: 'dashboard',
        component: () => import('#/views/tg-saas/dashboard/index.vue'),
        meta: { title: '仪表盘', icon: 'lucide:layout-dashboard' },
      },
      {
        name: 'TgSaasOrders',
        path: 'orders',
        component: () => import('#/views/tg-saas/orders/index.vue'),
        meta: { title: '订单管理', icon: 'lucide:receipt' },
      },
      {
        name: 'TgSaasSettings',
        path: 'settings',
        component: () => import('#/views/tg-saas/settings/index.vue'),
        meta: { title: '平台设置', icon: 'lucide:settings' },
      },
      {
        name: 'TgSaasAgents',
        path: 'agents',
        component: () => import('#/views/tg-saas/agents/index.vue'),
        meta: { title: '代理管理', icon: 'lucide:users' },
      },
      {
        name: 'TgSaasPayments',
        path: 'payments',
        component: () => import('#/views/tg-saas/payments/index.vue'),
        meta: { title: '支付网关配置', icon: 'lucide:wallet' },
      },
    ],
  },
];

export default routes;
EOF

cat > "$APP/src/views/tg-saas/dashboard/index.vue" <<'EOF'
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
EOF

cat > "$APP/src/views/tg-saas/orders/index.vue" <<'EOF'
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
EOF

cat > "$APP/src/views/tg-saas/settings/index.vue" <<'EOF'
<script setup lang="ts">
import { Alert, Button, Card, Form, Input, InputNumber, Switch, message } from 'ant-design-vue';
import { reactive } from 'vue';
import { Page } from '@vben/common-ui';

const settings = reactive({
  platformName: 'TG Group SaaS',
  supportUrl: '',
  sessionTimeout: 60,
  loginCaptcha: true,
});
function savePreview() {
  message.info('当前为界面预览，设置尚未写入服务端。');
}
</script>

<template>
  <Page title="平台设置" description="平台基本信息与安全策略">
    <section class="flex flex-col gap-4">
      <Alert
        message="设置界面预览"
        description="表单仅用于阶段一布局验收，当前没有持久化接口。"
        type="info"
        show-icon
      />
      <Card title="基础配置">
        <Form layout="vertical" :model="settings">
          <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <Form.Item label="平台名称">
              <Input v-model:value="settings.platformName" />
            </Form.Item>
            <Form.Item label="客服地址">
              <Input v-model:value="settings.supportUrl" placeholder="https://..." />
            </Form.Item>
          </div>
          <div class="flex justify-end">
            <Button type="primary" @click="savePreview">保存预览</Button>
          </div>
        </Form>
      </Card>
      <Card title="安全策略">
        <Form layout="vertical" :model="settings">
          <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <Form.Item label="会话超时（分钟）">
              <InputNumber v-model:value="settings.sessionTimeout" :min="5" :precision="0" class="w-full" />
            </Form.Item>
            <Form.Item label="登录验证码">
              <Switch v-model:checked="settings.loginCaptcha" />
            </Form.Item>
          </div>
        </Form>
      </Card>
    </section>
  </Page>
</template>
EOF

cat > "$APP/src/views/tg-saas/agents/index.vue" <<'EOF'
<script setup lang="ts">
import { Alert, Card, Table, Tag } from 'ant-design-vue';
import { Page } from '@vben/common-ui';

const columns = [
  { title: '代理名称', dataIndex: 'name', key: 'name' },
  { title: '代理等级', dataIndex: 'level', key: 'level' },
  { title: '名下商户', dataIndex: 'merchants', key: 'merchants' },
  { title: '代理资格价格', dataIndex: 'price', key: 'price' },
  { title: '状态', dataIndex: 'status', key: 'status' },
];
const rows = [
  { key: '1', name: '演示一级代理', level: '一级', merchants: 12, price: '999 USDT', status: '演示数据' },
  { key: '2', name: '演示二级代理', level: '二级', merchants: 4, price: '499 USDT', status: '演示数据' },
];
</script>

<template>
  <Page title="代理管理" description="代理资格、上下级关系与名下商户">
    <section class="flex flex-col gap-4">
      <Alert
        message="代理资格必须付费开通"
        description="代理购买、等级权限、折扣继承和商户数据隔离属于阶段二；当前页面不提供免费开通或修改真实折扣的入口。"
        type="warning"
        show-icon
      />
      <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card title="代理规则">
          <ul class="list-disc space-y-2 pl-5">
            <li>支付成功回调后才创建代理资格。</li>
            <li>代理后台仅能访问自己名下商户。</li>
            <li>自身进价折扣由上级代理或平台确定。</li>
            <li>下级商户折扣与套餐加价均使用整数 USDT。</li>
          </ul>
        </Card>
        <Card title="资格与额度原则">
          <ul class="list-disc space-y-2 pl-5">
            <li>代理资格按付费套餐购买。</li>
            <li>商户套餐支付成功后分配 bot_limit。</li>
            <li>添加 Bot 前校验剩余额度。</li>
            <li>超额时提示购买更高套餐。</li>
          </ul>
        </Card>
      </div>
      <Card title="代理列表">
        <Table :columns="columns" :data-source="rows" :pagination="false" size="middle">
          <template #bodyCell="{ column, record }">
            <Tag v-if="column.key === 'status'" color="default">{{ record.status }}</Tag>
          </template>
        </Table>
      </Card>
    </section>
  </Page>
</template>
EOF

cat > "$APP/src/views/tg-saas/payments/index.vue" <<'EOF'
<script setup lang="ts">
import { Alert, Card, Form, Input, InputNumber, Select, Switch } from 'ant-design-vue';
import { reactive } from 'vue';
import { Page } from '@vben/common-ui';

const gmpay = reactive({ enabled: false, apiToken: '', feeRate: 0, minimumAmount: 1 });
const edusdt = reactive({ enabled: false, merchantId: '', apiKey: '', minimumAmount: 1 });
const nativeUsdt = reactive({ enabled: false, network: 'TRON', walletAddress: '', minimumAmount: 1 });
</script>

<template>
  <Page title="支付网关配置" description="GMPay、EDUSDT 与内置 USDT 收款">
    <section class="flex flex-col gap-4">
      <Alert
        message="金额精度规则"
        description="业务数据库中的套餐、代理折扣、订单和佣金金额均为整数 USDT 面值。仅在链上支付边界转换为最小单位整数：1 USDT = 1,000,000。"
        type="warning"
        show-icon
      />
      <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <Card title="GMPay（EPUSDT）">
          <Form layout="vertical" :model="gmpay">
            <Form.Item label="启用网关"><Switch v-model:checked="gmpay.enabled" /></Form.Item>
            <Form.Item label="API Token"><Input.Password v-model:value="gmpay.apiToken" /></Form.Item>
            <Form.Item label="费率（整数）">
              <InputNumber v-model:value="gmpay.feeRate" :min="0" :precision="0" class="w-full" />
            </Form.Item>
            <Form.Item label="最低收款金额">
              <InputNumber v-model:value="gmpay.minimumAmount" :min="1" :precision="0" addon-after="USDT" class="w-full" />
            </Form.Item>
          </Form>
        </Card>
        <Card title="EDUSDT">
          <Form layout="vertical" :model="edusdt">
            <Form.Item label="启用网关"><Switch v-model:checked="edusdt.enabled" /></Form.Item>
            <Form.Item label="商户标识"><Input v-model:value="edusdt.merchantId" /></Form.Item>
            <Form.Item label="API Key"><Input.Password v-model:value="edusdt.apiKey" /></Form.Item>
            <Form.Item label="最低收款金额">
              <InputNumber v-model:value="edusdt.minimumAmount" :min="1" :precision="0" addon-after="USDT" class="w-full" />
            </Form.Item>
          </Form>
        </Card>
        <Card title="内置 USDT 收款">
          <Form layout="vertical" :model="nativeUsdt">
            <Form.Item label="启用网关"><Switch v-model:checked="nativeUsdt.enabled" /></Form.Item>
            <Form.Item label="收款网络">
              <Select v-model:value="nativeUsdt.network" :options="[{ label: 'TRON（TRC20）', value: 'TRON' }]" />
            </Form.Item>
            <Form.Item label="收款地址"><Input v-model:value="nativeUsdt.walletAddress" /></Form.Item>
            <Form.Item label="最低收款金额">
              <InputNumber v-model:value="nativeUsdt.minimumAmount" :min="1" :precision="0" addon-after="USDT" class="w-full" />
            </Form.Item>
          </Form>
        </Card>
        <Card title="统一金额模型">
          <div class="flex flex-col gap-3">
            <Alert message="数据库面值：整数 USDT" description="例如套餐价格 99 表示 99 USDT，不保存小数金额。" type="info" />
            <Alert message="链上单位：最小单位整数" description="网关适配层按 1 USDT = 1,000,000 最小单位换算；禁止把链上最小单位直接写入业务金额字段。" type="info" />
          </div>
        </Card>
      </div>
    </section>
  </Page>
</template>
EOF

echo "Vben Admin v5 SaaS page scaffold generated."
