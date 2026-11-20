<script lang="ts" setup>
import type { VbenFormSchema } from '@vben/common-ui';

import { computed, onMounted, ref } from 'vue';

import { AuthenticationLogin, z } from '@vben/common-ui';
import { message } from 'ant-design-vue';
import { $t } from '@vben/locales';

import { getPublicCaptchaConfigApi } from '#/api/core/auth';
import CapLoginVerification from '#/components/cap/CapLoginVerification.vue';
import { useAuthStore } from '#/store';

defineOptions({ name: 'Login' });

const authStore = useAuthStore();
const captchaConfig = ref<{
  enabled: boolean;
  mode: 'checkbox' | 'floating' | 'invisible';
  domain: string;
  siteKey: string;
} | null>(null);
const captchaWidget = ref<{ getToken: () => Promise<string> } | null>(null);
const captchaRenderKey = ref(0);
const captchaConfigLoaded = ref(false);

const formSchema = computed((): VbenFormSchema[] => [
  {
    component: 'VbenInput',
    componentProps: {
      autocomplete: 'username',
      placeholder: '请输入平台管理员账号',
    },
    fieldName: 'username',
    label: '管理员账号',
    rules: z.string().min(1, { message: '请输入平台管理员账号' }),
  },
  {
    component: 'VbenInputPassword',
    componentProps: {
      autocomplete: 'current-password',
      placeholder: $t('authentication.password'),
    },
    fieldName: 'password',
    label: $t('authentication.password'),
    rules: z.string().min(1, { message: $t('authentication.passwordTip') }),
  },
]);

const captchaEndpoint = computed(() => {
  const config = captchaConfig.value;
  if (!config) return '';
  return `${config.domain.replace(/\/+$/, '')}/${encodeURIComponent(config.siteKey)}/`;
});

onMounted(async () => {
  try {
    const config = await getPublicCaptchaConfigApi();
    captchaConfig.value = config;
  } catch {
    message.error('无法读取人机验证配置，请稍后重试');
  } finally {
    captchaConfigLoaded.value = true;
  }
});

async function handleLogin(params: Record<string, unknown>) {
  if (!captchaConfigLoaded.value || !captchaConfig.value) {
    message.error('人机验证配置尚未加载，请稍后重试');
    return;
  }
  let token = '';
  if (captchaConfig.value?.enabled) {
    try {
      token = await captchaWidget.value?.getToken() ?? '';
      if (!token) throw new Error('请先完成安全验证');
    } catch (error) {
      message.error(error instanceof Error ? error.message : '请先完成安全验证');
      return;
    }
  }
  try {
    await authStore.authLogin({
      ...params,
      captchaToken: token,
    });
  } finally {
    // Cap tokens are single-use; require a fresh challenge after every login attempt.
    captchaRenderKey.value += 1;
  }
}
</script>

<template>
  <AuthenticationLogin
    :form-schema="formSchema"
    :loading="authStore.loginLoading"
    @submit="handleLogin"
  >
    <template #captcha>
      <CapLoginVerification
        v-if="captchaConfig?.enabled"
        :key="`${captchaConfig.mode}-${captchaEndpoint}-${captchaRenderKey}`"
        ref="captchaWidget"
        :endpoint="captchaEndpoint"
        :mode="captchaConfig.mode"
        class="mb-4"
        @error="message.error($event)"
      />
    </template>
  </AuthenticationLogin>
</template>
