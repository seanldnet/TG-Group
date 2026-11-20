# TG Group Manager SaaS — Project Status

- 文档版本：0.1.0
- 更新日期：2026-11-21
- 仓库：seanldnet/TG-Group
- 主分支：main
- 当前阶段：阶段一（Vben Admin 初始化与五个核心页面 UI 间距骨架）
- 阶段状态：进行中，尚未通过构建和逐页验收

## 仓库审查结论

仓库初始审查确认：空仓库、无历史提交、无其他分支、无 PR、无 Issues、无业务源码。本项目按新项目初始化，不把其他仓库（包括 TG-Card 或旧 TG SaaS 仓库）的实现视为本仓库现状。

## 技术基线

- 管理端：Vue 3 + TypeScript + Vite
- UI 框架：Vue Vben Admin v2.11.5（Ant Design Vue 4）
- Vben 业务组件：PageWrapper（由项目 PageContainer 适配器封装）、BasicTable、BasicForm、BasicModal、BasicDrawer
- 页面间距：页面内容唯一使用 flex/grid gap；基础间距 gap-4 = 16px
- 全站计价：整数 USDT 面值；链上结算边界使用 1 USDT = 1,000,000 最小单位
- 当前阶段不创建业务数据库表、不接入支付、不实现业务 API

## 阶段门禁

### 阶段一
- [ ] Vben Admin 源码初始化完成
- [ ] 订单管理页间距验收
- [ ] 仪表盘间距验收
- [ ] 设置页间距验收
- [ ] 代理管理页间距验收
- [ ] 支付网关配置页间距验收
- [ ] 自动间距检查通过
- [ ] TypeScript 检查通过
- [ ] 前端生产构建通过
- [ ] CHANGELOG 记录阶段一验收证据

阶段一全部通过前，不得启动阶段二。

## 风险与说明

- Vben Admin v2 是为满足指令中的 BasicTable / BasicForm 组件形态而选择的兼容主线；上游固定到 commit 038936ff7d6178b1f3da2f5e6a36dc5a1155c74b，避免跟随上游分支漂移。
- UI 骨架不代表真实业务功能完成；所有表格数据均为演示占位数据，后续阶段接入真实 API。
- 当前仓库没有后端、数据库、支付网关或 Bot worker，均属于后续阶段。
