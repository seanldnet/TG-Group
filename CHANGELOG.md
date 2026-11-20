# Changelog

## [0.1.8] - 2026-11-21

### Telegram Bot 管理 API（PR #6，CI 待复验）

- 新增 Owner 鉴权保护的 Bot 列表、创建、重命名、删除 API。
- 创建 Bot 前调用 Telegram `getMe` 验证 Token；区分 Token 无效与 Telegram 网络/服务异常。
- Bot Token 使用 AES-256-GCM 加密后保存，接口仅返回 Token 掩码；新增 `BOT_TOKEN_ENCRYPTION_KEY` 配置及密钥持久化说明。
- 新增 Bot Telegram username 唯一索引、Bot 列表群组计数及加密相关测试。
- 新注册 Bot 保持 inactive；Telegram polling/update worker 仍待开发。
- [ ] Go test/vet/build CI 通过。
- [ ] V5 Bot 管理页面接入真实 API。

### Owner 认证与 V5 接入

- PR #3 Owner 登录、持久化 Bearer 会话、退出撤销、密码修改和 Redis 登录限流的 Go CI 已通过。
- PR #5 已提交 V5 登录页、用户信息、退出登录、密码修改真实 API 接入；TypeScript 与生产构建验证待通过。
- [ ] 浏览器端完整登录、退出和密码修改验收。

## [0.1.7] - 2026-11-21

### Owner 登录与会话 API（PR #3，CI/审查待确认）

- 新增持久化随机 Bearer 会话，数据库仅保存 SHA-256 token hash，默认 12 小时过期。
- 新增 Owner 登录、当前用户信息、权限码、退出登录和修改密码 API。
- 修改密码验证当前 bcrypt 密码，更新后撤销该 Owner 的全部会话。
- 新增 Redis 登录尝试限流：每个来源 IP 15 分钟最多 5 次尝试。
- 新增会话表 migration 和认证基础单元测试。
- [ ] PR #3 Go CI、数据库集成验证和安全审查通过。
- [ ] V5 登录页、用户信息、退出登录与修改密码接通真实 API。

## [0.1.6] - 2026-11-21

### 后端基础设施第一批实现（PR #2，待 CI/审查）

- 新增 Go HTTP 服务入口、环境配置、优雅关闭。
- 新增 PostgreSQL 连接池、嵌入式 SQL migration 和迁移 ledger。
- 新增单平台 Owner、多 Bot、群组、群成员、订单和平台设置初始 schema；不引入商户/租户/代理模型。
- 新增首次 Owner bcrypt 初始化逻辑：仅在不存在 Owner 时创建默认 `admin / 123456`，不覆盖已存在账号密码。
- 新增 Redis 连接和健康探针。
- 新增包含应用、PostgreSQL、Redis 的 image-only Compose，带持久化 volumes 和健康检查。
- 新增 Go test/vet/build CI 工作流。
- [ ] Go CI 实际运行通过并完成数据库集成验收。
- [ ] 登录/session API 与 V5 前端接通。
- [ ] GHCR 云端镜像发布和 Docker 首装/升级验收。
- [ ] 宝塔安装向导、Garble 多平台 Release 与 SHA-256 验收。

本阶段不代表登录、Docker 镜像发布、宝塔安装器或 Garble Release 已交付。

## [0.1.5] - 2026-11-21

### 明确双部署模式及首次安装体验

- [x] Docker Compose 方案明确包含应用、PostgreSQL、Redis 三个服务。
- [x] Docker 全新数据库首次启动默认管理员为 `admin / 123456`，仅首次初始化创建，不覆盖后续修改的密码。
- [x] 宝塔方案明确为上传 Garble Go 二进制、启动后访问网页安装向导。
- [x] 宝塔安装向导要求填写 PostgreSQL 连接参数，并由用户自行设置管理员账号密码。
- [x] 明确安装向导自动迁移、初始化并锁定重复安装入口。
- [x] 明确宝塔模式的 Redis 配置要求（若运行时依赖 Redis）。
- [ ] Docker Compose 全新部署、升级及默认登录验收。
- [ ] 宝塔上传二进制、网页安装向导及数据库迁移验收。
- [ ] Garble 多平台构建、校验和与 smoke test。

## [0.1.4] - 2026-11-21

- 固化 Docker GHCR 预构建镜像、服务器免编译、Compose 禁止 `build:` 的要求。
- 固化正式 Go 二进制必须使用 Garble 混淆构建、附 SHA-256 与构建元数据的要求。

## [0.1.3] - 2026-11-21

- 产品架构调整为单用户、多机器人；取消多商户、多租户和代理销售体系。
- 管理端改为机器人管理、Bot 订单筛选和平台统一设置。

## [0.1.2] - 2026-11-21

- 阶段一 V5 页面 UI 骨架与初始 CI 验证记录。

## [0.1.1] - 2026-11-21

- 明确官方 Vben Admin V5 技术基线，拒绝 V2 和兼容层；记录 V2 bootstrap 失败尝试。
