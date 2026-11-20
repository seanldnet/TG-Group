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

## 最终交付：两种免编译部署方式

1. **Docker 一键部署（默认推荐）**：GHCR 提供预构建应用镜像，Compose 同时启动应用、PostgreSQL 和 Redis。服务器只需 Docker / Compose，执行 `docker compose pull` 和 `docker compose up -d`。首次全新安装默认管理员为 `admin / 123456)，打开后台即可登录；升级和重启不得重置用户修改后的密码。正式 Compose 只用 `image:`，严禁 `build:`。
2. **宝塔面板 + Garble Go 二进制**：用户把对应平台的 Garble 混淆 Go 二进制上传到宝塔并启动，浏览器打开内置安装页面，填写 PostgreSQL 数据库连接信息，再自行设置管理员账号和密码；向导自动迁移数据库并初始化，完成后进入后台。若应用依赖 Redis，安装向导/配置也须支持填写 Redis 连接信息。

两种方式都不需要在服务器编译。Docker 镜像和 Garble 二进制由 CI 云端构建、测试和发布。普通 `go build` 产物不得冒充正式 Garble 二进制。详细要求见 [交付与部署标准](docs/DEPLOYMENT.md)。

## Phase 1 gate

The five administration pages must pass source spacing checks, TypeScript validation, production build, and visual review before Phase 1 is accepted:

1. Operations dashboard
2. Bot management
3. Order management (filterable by Bot)
4. Platform settings
5. Platform payment gateway configuration

## Project status

See `docs/PROJECT_STATUS.md` and `CHANGELOG.md`.
