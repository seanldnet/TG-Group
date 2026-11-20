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
