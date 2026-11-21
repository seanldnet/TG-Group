<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Alert, Button, Card, Input, Select, Table } from 'ant-design-vue';
import { Page } from '@vben/common-ui';

import { getBotsApi, type BotSummary } from '#/api/core/bots';
import { getGroupsApi, type GroupListResult } from '#/api/core/groups';

const bots = ref<BotSummary[]>([]);
const result = ref<GroupListResult>({ items: [], total: 0, page: 1, pageSize: 20 });
const botId = ref('');
const keyword = ref('');
const loading = ref(false);
const error = ref('');

const botOptions = computed(() => [
  { label: '全部机器人', value: '' },
  ...bots.value.map((bot) => ({ label: bot.name, value: String(bot.id) })),
]);
const columns = [
  { title: '群组名称', dataIndex: 'title', key: 'title', width: 220 },
  { title: 'Telegram Chat ID', dataIndex: 'telegramChatId', key: 'telegramChatId', width: 180 },
  { title: '所属机器人', dataIndex: 'botName', key: 'botName', width: 180 },
  { title: '成员记录数', dataIndex: 'memberCount', key: 'memberCount', width: 120 },
  { title: '接入时间', dataIndex: 'createdAt', key: 'createdAt', width: 190 },
];

function formatDate(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

async function load() {
  loading.value = true;
  error.value = '';
  try {
    result.value = await getGroupsApi({
      ...(botId.value ? { botId: Number(botId.value) } : {}),
      ...(keyword.value.trim() ? { q: keyword.value.trim() } : {}),
      page: result.value.page,
      pageSize: result.value.pageSize,
    });
  } catch {
    error.value = '群组数据加载失败，请检查服务连接后重试。';
  } finally {
    loading.value = false;
  }
}

function search() {
  result.value.page = 1;
  void load();
}

function reset() {
  botId.value = '';
  keyword.value = '';
  result.value.page = 1;
  void load();
}

function changePage(pagination: { current?: number; pageSize?: number }) {
  result.value.page = pagination.current ?? 1;
  result.value.pageSize = pagination.pageSize ?? 20;
  void load();
}

onMounted(async () => {
  try { bots.value = await getBotsApi(); } catch { /* group list remains usable */ }
  await load();
});
</script>

<template>
  <Page title="群组管理" description="查看各 Telegram Bot 已同步的群组与成员记录">
    <section class="flex flex-col gap-4">
      <Alert v-if="error" :message="error" type="error" show-icon closable @close="error = ''" />
      <Card title="群组列表">
        <div class="flex flex-col gap-4">
          <div class="flex flex-wrap items-center gap-3">
            <Select v-model:value="botId" :options="botOptions" class="min-w-52" placeholder="所属机器人" />
            <Input
              v-model:value="keyword"
              allow-clear
              class="w-72"
              placeholder="搜索群组名称或 Telegram Chat ID"
              @press-enter="search"
            />
            <Button type="primary" @click="search">查询</Button>
            <Button @click="reset">重置</Button>
          </div>
          <Table
            :columns="columns"
            :data-source="result.items"
            :loading="loading"
            :pagination="{ current: result.page, pageSize: result.pageSize, total: result.total, showSizeChanger: true }"
            :scroll="{ x: 900 }"
            row-key="id"
            size="middle"
            @change="changePage"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'createdAt'">{{ formatDate(record.createdAt) }}</template>
            </template>
          </Table>
        </div>
      </Card>
    </section>
  </Page>
</template>
