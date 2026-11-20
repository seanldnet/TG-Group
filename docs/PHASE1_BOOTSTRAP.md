# Phase 1 — Vben Admin V5 bootstrap

本项目仅采用官方 Vue Vben Admin V5，固定上游 commit `e726bc92780c1023c2393042871470d413ad79dc`。V5 与 V2 不兼容，本仓库不保留 V2 API、目录结构或组件兼容层。

GitHub Actions 将导入官方 monorepo，并在 `apps/web-antd` 中生成以下页面：

- 仪表盘
- 订单管理
- 平台设置
- 代理管理
- 支付网关配置

页面容器使用 V5 的 `Page`；页面表单和表格采用 V5 Ant Design 应用的 Ant Design Vue 组件。五页通过 `gap-4` 间距检查、no-V2 引用检查、TypeScript 检查、生产构建和人工视觉复核之前，阶段一不得验收，也不得启动阶段二。

当前页面为 UI 骨架；演示数据不代表真实业务已接通。
