<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';

import {
  Alert,
  Button,
  Card,
  Input,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tag,
  message,
} from 'ant-design-vue';
import { Page } from '@vben/common-ui';

import type { BotRecord } from '#/api';
import {
  createBotApi,
  deleteBotApi,
  listBotsApi,
  renameBotApi,
} from '#/api';

const bots = ref<BotRecord[]>([]);
const loading = ref(false);
const createOpen = ref(false);
const createLoading = ref(false);
const renameOpen = ref(false);
const renameLoading = ref(false);

const createForm = reactive({ name: '', token: '' });
const renameForm = reactive({ id: 0, name: '' });

const totalBots = computed(() => bots.value.length);
const activeBots = computed(
  () => bots.value.filter((bot) => bot.status === 'active').length,
);
const totalGroups = computed(() =>
  bots.value.reduce((sum, bot) => sum + bot.groupCount, 0),
);

const columns = [
  { title: '机器人名称', dataIndex: 'name', key: 'name' },
  { title: 'Telegram 用户名', dataIndex: 'username', key: 'username' },
  { title: '已接入群组', dataIndex: 'groupCount', key: 'groupCount' },
  { title: '运行状态', dataIndex: 'status', key: 'status' },
  { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt' },
  { title: '操作', key: 'actions', fixed: 'right' as const, width: 150 },
];

function statusLabel(status: BotRecord['status']) {
  if (status === 'active') return '运行中';
  if (status === 'error') return '异常';
  return '已停用';
}

function statusColor(status: BotRecord['status']) {
  if (status === 'active') return 'success';
  if (status === 'error') return 'error';
  return 'default';
}

async function loadBots() {
  loading.value = true;
  try {
    bots.value = await listBotsApi();
  } finally {
    loading.value = false;
  }
}

function openCreate() {
  createForm.name = '';
  createForm.token = '';
  createOpen.value = true;
}

async function submitCreate() {
  const name = createForm.name.trim();
  const token = createForm.token.trim();
  if (!name || !token) {
    message.warning('请填写机器人名称和 Bot Token');
    return;
  }
  createLoading.value = true;
  try {
    const created = await createBotApi({ name, token });
    createOpen.value = false;
    createForm.token = '';
    message.success(`机器人 @${created.username} 已验证并添加`);
    await loadBots();
  } finally {
    createLoading.value = false;
  }
}

function openRename(bot: BotRecord) {
  renameForm.id = bot.id;
  renameForm.name = bot.name;
  renameOpen.value = true;
}

async function submitRename() {
  const name = renameForm.name.trim();
  if (!name) {
    message.warning('机器人名称不能为空');
    return;
  }
  renameLoading.value = true;
  try {
    await renameBotApi(renameForm.id, name);
    renameOpen.value = false;
    message.success('机器人名称已更新');
    await loadBots();
  } finally {
    renameLoading.value = false;
  }
}

async function removeBot(bot: BotRecord) {
  await deleteBotApi(bot.id);
  message.success(`机器人 @${bot.username} 已删除`);
  await loadBots();
}

onMounted(() => {
  void loadBots();
});
</script>

<template>
  <Page title="机器人管理" description="由一个平台所有者统一管理多个 Telegram Bot">
    <section class="flex flex-col gap-4">
      <Alert
        message="单用户、多机器人模式"
        description="添加时会向 Telegram 验证 Bot Token，验证通过后加密保存。机器人与其群组数据按 Bot 归属管理。"
        type="info"
        show-icon
      />

      <div class="flex justify-end">
        <Button type="primary" @click="openCreate">添加机器人</Button>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
        <Card>
          <div class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground">已登记机器人</span>
            <strong class="text-2xl">{{ totalBots }}</strong>
          </div>
        </Card>
        <Card>
          <div class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground">运行中</span>
            <strong class="text-2xl">{{ activeBots }}</strong>
          </div>
        </Card>
        <Card>
          <div class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground">接入群组</span>
            <strong class="text-2xl">{{ totalGroups }}</strong>
          </div>
        </Card>
      </div>

      <Card title="Bot 列表">
        <Table
          :columns="columns"
          :data-source="bots"
          :loading="loading"
          :pagination="{ pageSize: 10, showSizeChanger: true }"
          :scroll="{ x: 900 }"
          row-key="id"
          size="middle"
        >
          <template #bodyCell="{ column, record }">
            <Tag v-if="column.key === 'status'" :color="statusColor(record.status)">
              {{ statusLabel(record.status) }}
            </Tag>
            <Space v-else-if="column.key === 'actions'">
              <Button type="link" size="small" @click="openRename(record as BotRecord)">
                重命名
              </Button>
              <Popconfirm
                title="确定删除这个机器人？"
                description="该 Bot 关联的群组与成员记录也会按数据库级联规则删除。"
                ok-text="删除"
                cancel-text="取消"
                @confirm="removeBot(record as BotRecord)"
              >
                <Button danger type="link" size="small">删除</Button>
              </Popconfirm>
            </Space>
          </template>
        </Table>
      </Card>
    </section>

    <Modal
      v-model:open="createOpen"
      title="添加 Telegram 机器人"
      :confirm-loading="createLoading"
      ok-text="验证并添加"
      cancel-text="取消"
      @ok="submitCreate"
    >
      <div class="flex flex-col gap-3 py-2">
        <label class="text-sm font-medium">机器人名称</label>
        <Input v-model:value="createForm.name" placeholder="例如：社区管理机器人" />
        <label class="text-sm font-medium">Bot Token</label>
        <Input.Password
          v-model:value="createForm.token"
          autocomplete="off"
          placeholder="粘贴从 BotFather 获取的 Token"
        />
        <span class="text-xs text-muted-foreground">
          Token 仅用于 Telegram 验证和加密保存，不会在列表中再次显示。
        </span>
      </div>
    </Modal>

    <Modal
      v-model:open="renameOpen"
      title="重命名机器人"
      :confirm-loading="renameLoading"
      ok-text="保存"
      cancel-text="取消"
      @ok="submitRename"
    >
      <div class="py-2">
        <Input v-model:value="renameForm.name"  :maxlength="120" placeholder="机器人名称" />
      </div>
    </Modal>
  </Page>
</template>
