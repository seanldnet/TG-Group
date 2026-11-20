<script setup lang="ts">
import { Alert, Button, Card, Table, Tag, message } from 'ant-design-vue';
import { Page } from '@vben/common-ui';

const columns = [
  { title: '机器人名称', dataIndex: 'name', key: 'name' },
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '已接入群组', dataIndex: 'groups', key: 'groups' },
  { title: '今日消息', dataIndex: 'messages', key: 'messages' },
  { title: '运行状态', dataIndex: 'status', key: 'status' },
];
const rows = [
  { key: '1', name: '社区管理机器人', username: '@community_guard_bot', groups: 8, messages: 1260, status: '运行中' },
  { key: '2', name: '活动运营机器人', username: '@community_event_bot', groups: 3, messages: 428, status: '已暂停' },
];
function addBotPreview() {
  message.info('当前为界面预览，Bot Token 校验和保存接口将在后续阶段接入。');
}
</script>

<template>
  <Page title="机器人管理" description="由一个平台所有者统一管理多个 Telegram Bot">
    <section class="flex flex-col gap-4">
      <Alert
        message="单用户、多机器人模式"
        description="本系统只有一个平台所有者。每个 Bot 可独立配置群组、验证、反垃圾、欢迎语和自动化规则；当前列表为演示数据。"
        type="info"
        show-icon
      />
      <div class="flex justify-end">
        <Button type="primary" @click="addBotPreview">添加机器人</Button>
      </div>
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        <Card>
          <div class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground">已登记机器人</span>
            <strong class="text-2xl">2</strong>
          </div>
        </Card>
        <Card>
          <div class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground">运行中</span>
            <strong class="text-2xl">1</strong>
          </div>
        </Card>
        <Card>
          <div class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground">接入群组</span>
            <strong class="text-2xl">11</strong>
          </div>
        </Card>
      </div>
      <Card title="Bot 列表">
        <Table :columns="columns" :data-source="rows" :pagination="false" :scroll="{ x: 760 }" size="middle">
          <template #bodyCell="{ column, record }">
            <Tag v-if="column.key === 'status'" :color="record.status === '运行中' ? 'success' : 'default'">
              {{ record.status }}
            </Tag>
          </template>
        </Table>
      </Card>
    </section>
  </Page>
</template>
