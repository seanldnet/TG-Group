# TG Group Manager SaaS — Project Status

- 文档版本：0.1.2
- 更新日期：2026-11-21
- 仓库：seanldnet/TG-Group
- 主分支：main
- 阶段开发分支：feat/phase-1-vben-v5
- 阶段一 PR：#1（Draft）
- 当前阶段：阶段一（官方 Vben Admin V5 初始化与五个核心页面 UI 骨架）
- 阶段状态：代码、源码间距检查、TypeScript 与生产构建已通过；等待五页视觉验收

## 仓库审查结论

- 仓库初始时仅包含项目文档、脚本和初始化工作流，没有可继承的业务源码、后端、数据库模型、支付服务或 Bot worker。
- 旧 V2 初始化工作流的页面生成步骤失败，未创建 PR，也未通过构建；该尝试未合并并已从 V5 分支清理。
- 用户明确要求只采用 V5，不使用 V2，也不做 V2 组件/API 兼容。
- 阶段一所有开发均在独立分支进行；没有直接向 main 推送阶段一业务代码。

## 技术基线

- 管理端：Vue 3 + TypeScript + Vite
- 管理模板：官方 Vue Vben Admin V5，固定上游 commit：`e726bc92780c1023c2393042871470d413ad79dc`
- UI 变体：`apps/web-antd`（Ant Design Vue）
- V5 页面容器：`@vben/common-ui` 导出的 `Page`
- 页面表格/表单：V5 Ant Design 应用实际集成的 Ant Design Vue 组件；不引用 V2 的 `BasicTable`、`BasicForm`、`PageWrapper`、`BasicModal`、`BasicDrawer`
- 页面间距：五个目标页面使用 `flex flex-col gap-4` 或 `grid gap-4`；禁止手写 top/bottom margin
- 全站计价：业务数据库金额为整数 USDT 面值；链上边界使用最小单位整数，1 USDT = 1,000,000
- V5 上游的 monorepo `scripts`、`docs` 目录保留；TG Group 自定义脚本放在 `tg-group/scripts`

## 阶段一交付与验证

- [x] 官方 Vben Admin V5 源码初始化完成
- [x] 仪表盘、订单管理、设置、代理管理、支付网关配置五页源码生成
- [x] 五页路由注册
- [x] 自动间距与 no-V2 检查通过
- [x] V5 Ant Design 应用 TypeScript 检查通过
- [x] V5 Ant Design 应用生产构建通过
- [ ] 订单管理页视觉间距验收
- [ ] 仪表盘视觉间距验收
- [ ] 设置页视觉间距验收
- [ ] 代理管理页视觉间距验收
- [ ] 支付网关配置页视觉间距验收
- [ ] 五页视觉验收通过后将 PR 从 Draft 转为 Ready

### 最近一次 CI 证据

- Workflow：Phase 1 - Vben Admin V5 bootstrap
- Run：44798804471
- 结论：success
- 分支提交：`26ae60404ebdf535d81cbc791eda0adc87e1c825`
- 构建产物：`tg-group-admin-v5-phase1-dist`
- Artifact SHA-256：`23de6cc520637242891feb146ce00b732582e850b5cc2aa14c6060c1d38b9fb3`

**阶段一全部视觉验收通过前，不得启动阶段二。**

## 风险与说明

- 五个页面当前是 UI 骨架；表格和指标中的演示数据不代表真实运营数据。
- 设置表单的保存按钮只提示当前为预览，不写入服务端；没有伪造真实业务提交。
- 代理资格、商户套餐、bot_limit、租户隔离、支付网关和 Bot 功能均未实现，不能标记为完成。
- 当前阶段不创建业务数据库表、不接入真实支付、不实现业务 API。
