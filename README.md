# TG Group Manager

单用户、多机器人的 Telegram 群组管理系统。一个平台所有者通过统一管理后台管理多个 Telegram Bot。

## 产品架构

- 单一平台所有者，不提供商户注册、商户独立后台或多租户隔离。
- 多个 Telegram Bot 由同一个平台统一添加和管理；每个 Bot 可独立关联群组并配置群管规则。
- 不包含代理销售、代理等级、上下级代理、代理充值或代理佣金体系。
- Telegram 群成员仍是被管理的业务对象，保留身份、积分、等级、处罚、邀请和活动数据。
- 支付网关与安全策略由平台统一配置；订单记录 Bot 归属并支持按 Bot 查询。

## 管理端技术基线

- Frontend: Vue 3 + TypeScript + Vite
- Admin template: official Vue Vben Admin V5
- UI variant: `apps/web-antd` (Ant Design Vue)
- Vben V5 only; no V2 compatibility wrappers or APIs.
- Business prices use integer USDT face value (1 = 1 USDT).
- On-chain USDT settlement uses integer smallest units (1 USDT = 1,000,000 smallest units), converted only at the payment boundary.

## Phase 1 gate

The five administration pages must pass source spacing checks, TypeScript validation, production build, and visual review before Phase 1 is accepted:

1. Operations dashboard
2. Bot management
3. Order management (filterable by Bot)
4. Platform settings
5. Platform payment gateway configuration

Project page content uses flex/grid gap utilities. Manual top/bottom margin utilities are prohibited in these five pages.

## Project status

See `docs/PROJECT_STATUS.md` and `CHANGELOG.md`.
