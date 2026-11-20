# TG Group Manager SaaS

多商户 Telegram 群管机器人 SaaS 项目。

## 管理端技术基线

- Frontend: Vue 3 + TypeScript + Vite
- Admin template: official Vue Vben Admin V5
- UI variant: `apps/web-antd` (Ant Design Vue)
- Vben V5 and V2 are incompatible; this project uses V5 only and does not maintain V2 compatibility wrappers or APIs.
- Billing display and business prices: integer USDT face value only (1 = 1 USDT).
- On-chain USDT settlement: integer smallest units (1 USDT = 1,000,000 smallest units), converted only at the payment boundary.

## Phase 1 gate

The five administration pages must pass the source spacing check, TypeScript validation, production build, and visual review before Phase 2 begins:

1. Dashboard
2. Order management
3. Platform settings
4. Agent management
5. Payment gateway configuration

Page content spacing uses flex/grid gap utilities. Manual top/bottom margin utilities are prohibited in these five project pages.

## Project status

See `docs/PROJECT_STATUS.md` and `CHANGELOG.md`.
