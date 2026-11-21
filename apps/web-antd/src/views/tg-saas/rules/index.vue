<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { Alert, Button, Card, Checkbox, Input, Select, Space, message } from 'ant-design-vue';
import { Page } from '@vben/common-ui';

import { getGroupsApi, type GroupListItem } from '#/api/core/groups';
import { getGroupRulesApi, saveGroupRulesApi } from '#/api/core/group-rules';

const groups = ref<GroupListItem[]>([]);
const selectedGroupId = ref<number>();
const blockLinks = ref(false);
const keywordsText = ref('');
const loading = ref(false);
const saving = ref(false);
const error = ref('');

const groupOptions = computed(() => groups.value.map((g) => ({
  label: `${g.title} · ${g.botName}`,
  value: g.id,
})));

async function loadRules(groupId: number) {
  loading.value = true;
  error.value = '';
  try {
    const rules = await getGroupRulesApi(groupId);
    blockLinks.value = rules.blockLinks;
    keywordsText.value = rules.blockedKeywords.join('\n');
  } catch {
    error.value = '规则读取失败，请稍后重试。';
  } finally {
    loading.value = false;
  }
}

async function save() {
  if (!selectedGroupId.value) return;
  saving.value = true;
  error.value = '';
  try {
    const blockedKeywords = keywordsText.value.split('\n').map((s) => s.trim()).filter(Boolean);
    await saveGroupRulesApi(selectedGroupId.value, { blockLinks: blockLinks.value, blockedKeywords });
    message.success('群管规则已保存');
  } catch {
    error.value = '保存失败，请检查服务连接后重试。';
  } finally {
    saving.value = false;
  }
}

watch(selectedGroupId, (id) => { if (id) void loadRules(id); });

onMounted(async () => {
  try {
    const result = await getGroupsApi({ page: 1, pageSize: 100 });
    groups.value = result.items;
    if (groups.value[0]) selectedGroupId.value = groups.value[0].id;
  } catch {
    error.value = '群组列表加载失败，请先确认机器人已同步群组。';
  }
});
</script>

<template>
  <Page title="群管规则" description="按群组配置链接拦截与关键词过滤策略">
    <section class="flex flex-col gap-4">
      <Alert v-if="error" :message="error" type="error" show-icon />
      <Alert
        message="规则说明"
        description="当前版本提供群组级策略保存接口。已保存的规则会由 Bot 轮询器检查普通文本消息；命中链接或关键词时尝试删除消息。Bot 需要具备删除消息权限。"
        type="info"
        show-icon
      />
      <Card title="规则配置">
        <div class="flex flex-col gap-5">
          <div class="flex flex-col gap-2">
            <span class="font-medium">目标群组</span>
            <Select
              v-model:value="selectedGroupId"
              :options="groupOptions"
              :loading="loading"
              class="w-full max-w-xl"
              placeholder="选择需要配置规则的群组"
            />
          </div>
          <Checkbox v-model:checked="blockLinks">拦截普通文本消息中的链接</Checkbox>
          <div class="flex flex-col gap-2">
            <span class="font-medium">屏蔽关键词</span>
            <Input.TextArea
              v-model:value="keywordsText"
              :rows="8"
              placeholder="每行一个关键词，消息包含关键词时尝试删除"
            />
            <span class="text-xs text-muted-foreground">最多 100 个关键词，每个最多 64 个字符。</span>
          </div>
          <Space>
            <Button type="primary" :loading="saving" :disabled="!selectedGroupId" @click="save">保存规则</Button>
          </Space>
        </div>
      </Card>
    </section>
  </Page>
</template>
