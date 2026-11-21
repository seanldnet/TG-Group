# Changelog

## [0.1.8] - 2026-11-21

### 新版 Cap 登录人机验证（PR #12，待验收）

- 对接新版 Cap Widget，通过 jsDelivr 加载固定版本 `@cap.js/widget` 与浮动模式脚本；不依赖旧版 Widget files 开关。
- Owner 登录增加复选框、浮动、隐形三种验证模式；Go 后端对登录 token 调用 Cap `/{site-key}/siteverify`，失败时拒绝登录。
- 平台设置增加 Cap 域名、Site Key、Secret、启用开关和模式选择；Secret 以 AES-256-GCM 加密保存且不回显。
- Cap 加密密钥优先使用 `CAP_SECRET_ENCRYPTION_KEY`，回退复用 `BOT_TOKEN_ENCRYPTION_KEY`。
- 登录组件按 Vben V5 布局渲染，宽度限制不超过账号/密码输入框；中文状态提示。
- [x] Go test/vet/build CI 通过。
- [x] V5 spacing、TypeScript、production build CI 通过。
- [ ] 真实浏览器连接用户自建 Cap 实例的端到端验收待部署配置后完成。


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
