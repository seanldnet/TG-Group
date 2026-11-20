# TG Group Manager SaaS — Project Status

- 文档版本：0.1.1
- 更新日期：2026-11-21
- 仓库：seanldnet/TG-Group
- 主分支：main
- 阶段开发分支：feat/phase-1-vben-v5
- 当前阶段：阶段一（官方 Vben Admin V5 初始化与五个核心页面 UI 骨架）
- 阶段状态：进行中；等待 V5 导入、自动检查、TypeScript、生产构建和视觉验收

## 仓库审查结论

- 仓库此前仅包含项目文档、脚本和初始化工作流，没有可继承的业务源码、后端、数据库模型、支付服务或 Bot worker。
- 旧 V2 初始化工作流在页面生成步骤失败，未创建 PR，也未通过 TypeScript 或生产构建；该尝试不作为交付成果。
- 用户明确要求只采用 V5，不使用 V2，也不做 V2 组件/API 兼容。
- 当前阶段在独立分支执行，主分支不承载阶段一业务开发。

## 技术基线

- 管理端：Vue 3 + TypeScript + Vite
- 管理模板：官方 Vue Vben Admin V5，固定上游 commit：`e726bc92780c1023c2393042871470d413ad79dc`
- UI 变体：`apps/web-antd`（Ant Design Vue）
- V5 页面容器：`@vben/common-ui` 导出的 `Page`
- 页面表格/表单：使用 V5 应用实际集成的 Ant Design Vue 组件；不引用 V2 的 `BasicTable`、`BasicForm`、`PageWrapper`、`BasicModal`、`BasicDrawer`
- 页面间距：业务页面通过 `flex flex-col gap-4` 或 `grid gap-4` 组织；五个目标页面禁止手写 top/bottom margin
- 全站计价：业务数据库金额为整数 USDT 面值；链上边界使用最小单位整数，1 USDT = 1,000,000
- 当前阶段不创建业务数据库表、不接入真实支付、不实现业务 API

## 阶段门禁

### 阶段一
- [ ] 官方 Vben Admin V5 源码初始化完成
- [ ] 订单管理页间距验收
- [ ] 仪表盘间距验收
- [ ] 设置页间距验收
- [ ] 代理管理页间距验收
- [ ] 支付网关配置页间距验收
- [ ] 自动间距与 no-V2 检查通过
- [ ] V5 Ant Design 应用 TypeScript 检查通过
- [ ] V5 Ant Design 应用生产构建通过
- [ ] 五页视觉验收通过
- [ ] CHANGELOG 记录阶段一验收证据

**阶段一全部通过前，不得启动阶段二。**

## 风险与说明

- 当前五个页面是明确标注的 UI 骨架，表格中的演示数据不代表真实运营数据。
- V5 使用官方 monorepo 结构，Ant Design 应用位于 `apps/web-antd`；不能套用 V2 的目录、路由或组件 API。
- 后端、租户/代理数据模型、支付网关与 Bot worker 尚未实现。
