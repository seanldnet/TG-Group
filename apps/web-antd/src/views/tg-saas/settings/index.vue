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
