<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';

import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  message,
  Radio,
  Switch,
} from 'ant-design-vue';
import { Page } from '@vben/common-ui';

import {
  getCapSettingsApi,
  updateCapSettingsApi,
} from '#/api/core/captcha';

const settings = reactive({
  platformName: 'TG Group SaaS',
  supportUrl: '',
  sessionTimeout: 60,
});
const captcha = reactive({
  enabled: false,
  mode: 'checkbox' as 'checkbox' | 'floating' | 'invisible',
  domain: '',
  siteKey: '',
  secret: '',
  secretConfigured: false,
});
const loading = ref(false);
const saving = ref(false);

onMounted(async () => {
  loading.value = true;
  try {
    const data = await getCapSettingsApi();
    captcha.enabled = data.enabled;
    captcha.mode = data.mode;
    captcha.domain = data.domain;
    captcha.siteKey = data.siteKey;
    captcha.secretConfigured = data.secretConfigured;
  } catch {
    message.error('读取 Cap 配置失败');
  } finally {
    loading.value = false;
  }
});

async function saveCapSettings() {
  saving.value = true;
  try {
    const data = await updateCapSettingsApi({
      enabled: captcha.enabled,
      mode: captcha.mode,
      domain: captcha.domain,
      siteKey: captcha.siteKey,
      secret: captcha.secret,
    });
    captcha.secret = '';
    captcha.secretConfigured = data.secretConfigured;
    message.success('Cap 人机验证配置已保存');
  } catch {
    message.error('保存 Cap 配置失败，请检查域名、Site Key 和 Secret');
  } finally {
    saving.value = false;
  }
}

function savePreview() {
  message.info('基础信息配置暂未接入持久化接口。');
}
</script>

<template>
  <Page title="平台设置" description="平台基本信息与安全策略">
    <section class="flex flex-col gap-4">
      <Alert
        message="新版 Cap 人机验证"
        description="仅需配置 Cap 服务域名、Site Key 和 Secret。Widget 与 WASM 由 jsDelivr 加载，不需要开启旧版 Widget files。"
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

      <Card title="Cap 人机验证">
        <div class="flex flex-col gap-4">
          <div class="flex items-center justify-between">
            <div>
              **启用登录人机验证**
              <div class="text-muted-foreground text-xs">
                开启后，Owner 登录必须通过 Cap 服务端校验
              </div>
            </div>
            <Switch
              v-model:checked="captcha.enabled"
              :loading="loading"
              :disabled="loading"
            />
          </div>

          <Form layout="vertical" :model="captcha">
            <Form.Item label="Cap 服务域名" required>
              <Input
                v-model:value="captcha.domain"
                placeholder="https://cap.example.com"
                autocomplete="url"
              />
            </Form.Item>
            <Form.Item label="Site Key" required>
              <Input
                v-model:value="captcha.siteKey"
                placeholder="请输入 Cap Site Key"
                autocomplete="off"
              />
            </Form.Item>
            <Form.Item label="Secret" required>
              <Input.Password
                v-model:value="captcha.secret"
                :placeholder="captcha.secretConfigured ? '已配置，留空表示不修改' : '请输入 Cap Secret'"
                autocomplete="new-password"
              />
              <div v-if="captcha.secretConfigured" class="text-muted-foreground text-xs">
                Secret 已加密保存，不会回显；留空保存时将保留原密钥
              </div>
            </Form.Item>
            <Form.Item label="验证显示模式">
              <Radio.Group v-model:value="captcha.mode" button-style="solid">
                <Radio.Button value="checkbox">复选框</Radio.Button>
                <Radio.Button value="floating">浮动</Radio.Button>
                <Radio.Button value="invisible">隐形</Radio.Button>
              </Radio.Group>
            </Form.Item>
          </Form>

          <Alert
            v-if="captcha.mode === 'checkbox'"
            message="复选框模式"
            description="验证组件显示在登录密码框下方，宽度与账号、密码输入框一致。"
            type="info"
            show-icon
          />
          <Alert
            v-else-if="captcha.mode === 'floating'"
            message="浮动模式"
            description="点击登录框下方的安全验证入口后，在浮动窗口中完成验证。"
            type="info"
            show-icon
          />
          <Alert
            v-else
            message="隐形模式"
            description="登录时自动在后台执行验证，不显示验证控件。"
            type="info"
            show-icon
          />

          <div class="flex justify-end">
            <Button
              type="primary"
              :loading="saving"
              :disabled="loading"
              @click="saveCapSettings"
            >
              保存 Cap 配置
            </Button>
          </div>
        </div>
      </Card>

      <Card title="其他安全策略">
        <Form layout="vertical" :model="settings">
          <Form.Item label="会话超时（分钟）">
            <InputNumber
              v-model:value="settings.sessionTimeout"
              :min="5"
              :precision="0"
              class="w-full"
            />
          </Form.Item>
        </Form>
      </Card>
    </section>
  </Page>
</template>
