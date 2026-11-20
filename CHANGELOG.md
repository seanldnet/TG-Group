# Changelog

本文件按阶段记录可验证的交付结果。未通过验收的模块不得标记为完成。

## [0.1.2] - 2026-11-21

### 阶段一：官方 Vben Admin V5 初始化与核心页面 UI 骨架

- [x] 官方 Vben Admin V5 导入完成，固定上游 commit `e726bc92780c1023c2393042871470d413ad79dc`。
- [x] 五个管理页面源码及路由生成：仪表盘、订单管理、平台设置、代理管理、支付网关配置。
- [x] 页面容器使用 V5 `Page`，组件使用 V5 Ant Design 应用中的 Ant Design Vue。
- [x] 清理 V2 初始化工作流和 V2 页面脚本；没有 V2 API 或组件兼容层。
- [x] 保留 V5 上游 monorepo 的 `docs` 与 `scripts`；项目自定义脚本独立放在 `tg-group/scripts`。
- [x] 五页 gap-4 与禁止手动 top/bottom margin 的源码检查通过。
- [x] 五页 no-V2 组件引用检查通过。
- [x] pnpm workspace 依赖安装通过。
- [x] V5 Ant Design 应用 TypeScript 检查通过。
- [x] V5 Ant Design 应用生产构建通过。
- [x] GitHub Actions 构建产物上传成功。
- [ ] 仪表盘视觉间距验收。
- [ ] 订单管理页视觉间距验收。
- [ ] 设置页视觉间距验收。
- [ ] 代理管理页视觉间距验收。
- [ ] 支付网关配置页视觉间距验收。

**阶段一状态：代码与 CI 检查通过，视觉验收尚未完成；PR #1 保持 Draft，不进入阶段二。**

### CI 记录

- Workflow Run：44798804471（success）
- Branch：`feat/phase-1-vben-v5`
- Commit：`26ae60404ebdf535d81cbc791eda0adc87e1c825`
- Artifact：`tg-group-admin-v5-phase1-dist`
- SHA-256：`23de6cc520637242891feb146ce00b732582e850b5cc2aa14c6060c1d38b9fb3`

## [0.1.1] - 2026-11-21

### V5 技术基线调整

- [x] 明确技术基线为官方 Vben Admin V5，拒绝 V2 与 V2 兼容层。
- [x] 固定官方 V5 上游 commit。
- [x] 编写 V5 monorepo 页面生成脚本和五页面间距检查脚本。

### 失败尝试记录

- V2 bootstrap workflow 的页面生成步骤失败；未创建 PR、未通过构建。用户已明确要求放弃 V2，该失败尝试不会合并。
