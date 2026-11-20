<script setup lang="ts">
import type { VbenFormSchema } from '#/adapter/form';
import type { Recordable } from '@vben/types';

import { computed } from 'vue';

import { ProfilePasswordSetting, z } from '@vben/common-ui';

import { message } from 'ant-design-vue';

import { changePasswordApi } from '#/api';
import { useAuthStore } from '#/store';

const authStore = useAuthStore();

const formSchema = computed((): VbenFormSchema[] => [
  {
    fieldName: 'oldPassword',
    label: '旧密码',
    component: 'VbenInputPassword',
    componentProps: {
      placeholder: '请输入当前密码',
      autocomplete: 'current-password',
    },
    rules: z.string().min(1, { message: '请输入当前密码' }),
  },
  {
    fieldName: 'newPassword',
    label: '新密码',
    component: 'VbenInputPassword',
    componentProps: {
      passwordStrength: true,
      autocomplete: 'new-password',
      placeholder: '请输入至少 8 位新密码',
    },
    rules: z.string().min(8, { message: '新密码至少需要 8 位' }),
  },
  {
    fieldName: 'confirmPassword',
    label: '确认密码',
    component: 'VbenInputPassword',
    componentProps: {
      passwordStrength: true,
      autocomplete: 'new-password',
      placeholder: '请再次输入新密码',
    },
    dependencies: {
      rules(values) {
        const { newPassword } = values;
        return z
          .string({ error: '请再次输入新密码' })
          .min(1, { message: '请再次输入新密码' })
          .refine((value) => value === newPassword, {
            message: '两次输入的密码不一致',
          });
      },
      triggerFields: ['newPassword'],
    },
  },
]);

async function handleSubmit(values: Recordable<any>) {
  await changePasswordApi({
    currentPassword: values.oldPassword,
    newPassword: values.newPassword,
  });
  message.success('密码修改成功，请使用新密码重新登录');
  await authStore.logout();
}
</script>

<template>
  <ProfilePasswordSetting
    class="w-1/3"
    :form-schema="formSchema"
    @submit="handleSubmit"
  />
</template>
