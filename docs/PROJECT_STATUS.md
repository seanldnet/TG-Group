# TG Group Manager — Project Status

- 文档版本：0.1.3
- 更新日期：2026-11-21
- 仓库：seanldnet/TG-Group
- 主分支：main
- 阶段开发分支：feat/phase-1-vben-v5
- 阶段一 PR：#1（Draft）
- 当前阶段：阶段一（官方 Vben Admin V5 初始化与单用户多机器人管理 UI）
- 阶段状态：V5 页面骨架与基础 CI 已通过；架构调整后的页面及筛选功能等待新一轮 CI 和视觉验收

## 产品架构基线

- 产品采用单用户、多机器人模式：一个平台所有者使用一个管理后台统一管理多个 Telegram Bot。
- 不提供多商户注册、商户独立后台、租户隔离或商户套餐。
- 不提供代理销售体系：不设计代理等级、上下级代理、代理充值、代理折扣或代理佣金。
- Telegram 群成员仍是业务管理对象，保留群成员身份、积分、等级、处罚和活动数据；群成员不是商户或代理。
- 每个 Bot 可独立关联群组，并配置验证、反垃圾、欢迎语、处罚和自动化规则。
- 支付网关和平台安全配置由唯一平台所有者统一管理；订单保留 Bot 归属，便于按 Bot 查询。
- 后续业务模型以 Bot、Group、TelegramUser、Order、PlatformSetting 等为核心；不得引入 Tenant、Merchant、Agent 等多租户/代理销售核心模型。

## 仓库审查结论

- 仓库初始时仅包含项目文档、脚本和初始化工作流，没有可继承的业务源码、后端、数据库模型、支付服务或 Bot worker。
- 旧 V2 初始化工作流的页面生成步骤失败；未创建 PR、未通过构建。该尝试未合并并已从 V5 分支清理。
- 用户明确要求只采用官方 Vben Admin V5，不使用 V2，也不做 V2 组件/API 兼容。
- 阶段一开发均在独立分支进行；没有直接向 main 推送阶段一业务代码。

## 技术基线

- 管理端：Vue 3 + TypeScript + Vite
- 管理模板：官方 Vue Vben Admin V5，固定上游 commit：`e726bc92780c1023c2393042871470d413ad79dc`
- UI 变体：`apps/web-antd`（Ant Design Vue）
- V5 页面容器：`@vben/common-ui` 导出的 `Page`
- 页面表格/表单：V5 Ant Design 应用实际集成的 Ant Design Vue 组件；不引用 V2 的 `BasicTable`、`BasicForm`、`PageWrapper`、`BasicModal`、`BasicDrawer`
- 页面间距：所有目标业务页面使用 `flex flex-col gap-4` 或 `grid gap-4`；禁止手写 top/bottom margin
- 金额：业务金额使用整数 USDT 面值；链上边界使用最小单位整数，1 USDT = 1,000,000
- V5 上游 monorepo 的 `scripts`、`docs` 目录保留；TG Group 自定义脚本放在 `tg-group/scripts`

## 阶段一交付与验证

- [x] 官方 Vben Admin V5 源码初始化完成
- [x] 单用户多机器人产品架构已写入项目基线
- [x] 运营总览替换商户/代理指标
- [x] 新增机器人管理页面，移除代理管理路由
- [x] 订单页面加入按 Bot 筛选的前端演示
- [x] 支付配置明确为平台统一配置
- [x] no-V2 组件引用检查（原阶段通过；本轮改动后待重跑）
- [x] V5 页面源码间距检查（原阶段通过；本轮改动后待重跑）
- [ ] 架构调整后 TypeScript 检查与生产构建
- [ ] 五个业务页面视觉验收
- [ ] 视觉验收通过后将 PR 从 Draft 转为 Ready

### 最近一次已知 CI 证据（架构调整前）

- Workflow：Phase 1 - Vben Admin V5 bootstrap
- Run：44798804471
- 结论：success
- 分支提交：`26ae60404ebdf535d81cbc791eda0adc87e1c825`
- 构建产物：`tg-group-admin-v5-phase1-dist`
- Artifact SHA-256：`23de6cc520637242891feb146ce00b732582e850b5cc2aa14c6060c1d38b9fb3`

## 风险与说明

- 当前五个业务页面仍为 UI 骨架；列表、统计和订单数据均为演示数据。
- 添加 Bot、Token 验证、Bot 启停、群组关联、支付配置保存尚未接入后端 API，不得宣称已可真实运营。
- 当前阶段不创建业务数据库表、不接入真实支付、不实现业务 API。
- 不得恢复商户/代理页面、租户上下文或 V2 兼容层。
