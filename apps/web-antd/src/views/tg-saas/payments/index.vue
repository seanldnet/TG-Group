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
