import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  {
    name: 'TgSaas',
    path: '/tg-saas',
    meta: {
      icon: 'lucide:bot',
      order: 1,
      title: 'TG SaaS',
    },
    children: [
      {
        name: 'TgSaasDashboard',
        path: 'dashboard',
        component: () => import('#/views/tg-saas/dashboard/index.vue'),
        meta: { title: '仪表盘', icon: 'lucide:layout-dashboard' },
      },
      {
        name: 'TgSaasOrders',
        path: 'orders',
        component: () => import('#/views/tg-saas/orders/index.vue'),
        meta: { title: '订单管理', icon: 'lucide:receipt' },
      },
      {
        name: 'TgSaasSettings',
        path: 'settings',
        component: () => import('#/views/tg-saas/settings/index.vue'),
        meta: { title: '平台设置', icon: 'lucide:settings' },
      },
      {
        name: 'TgSaasAgents',
        path: 'agents',
        component: () => import('#/views/tg-saas/agents/index.vue'),
        meta: { title: '代理管理', icon: 'lucide:users' },
      },
      {
        name: 'TgSaasPayments',
        path: 'payments',
        component: () => import('#/views/tg-saas/payments/index.vue'),
        meta: { title: '支付网关配置', icon: 'lucide:wallet' },
      },
    ],
  },
];

export default routes;
