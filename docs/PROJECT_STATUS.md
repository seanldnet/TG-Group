# TG Group Manager — Project Status

- 文档版本：0.1.8
- 更新日期：2026-11-21
- 仓库：seanldnet/TG-Group
- 前端阶段分支：feat/phase-1-vben-v5（PR #1，Draft）
- 后端基础分支：feat/backend-foundation（PR #2，Draft，基于 PR #1）
- Owner 认证分支：feat/auth-session-api（PR #3，Draft，基于 PR #2）
- 当前状态：PR #2 的 Go test/vet/build CI 已通过；PR #3 Owner 认证 API 已提交，CI 验证中

## 产品架构基线

- 单一平台所有者统一管理多个 Telegram Bot。
- 不提供多商户注册、多租户隔离、商户套餐或代理销售体系。
- Telegram 群成员仍是正常业务管理对象。
- 每个 Bot 可独立关联群组并配置群管规则；平台支付和安全配置统一管理。

## 当前已提交实现

### 前端（PR #1）

- 官方 Vue Vben Admin V5，固定上游 commit：`e726bc92780c1023c2393042871470d413ad79dc`。
- UI 变体：`apps/web-antd`；不使用 V2 API 或兼容层。
- 五个 UI 页面：运营总览、机器人管理、订单（按 Bot 筛选）、平台设置、统一支付配置。
- V5 专项 CI 在提交 `0fc7335` 的运行 `44830241707` 结果为 success。
- 页面仍是演示 UI，尚未连接真实 API。

### 后端基础（PR #2）

- 新增 Go HTTP 服务入口、配置加载、PostgreSQL 连接池、嵌入式 SQL migration 和迁移记录表。
- 初始 schema 包含唯一平台 Owner、Bot、群组、群成员、订单（保留 Bot 归属）及平台设置；不包含 Tenant/Agent/代理销售表。
- 全新数据库仅在不存在 Owner 时创建 bcrypt 管理员记录，默认用户名/密码为 `admin / 123456`；启动逻辑不更新已有账号密码。
- 新增 Redis 连接、存活/就绪探针及优雅关闭。
- 新增 image-only Compose：应用、PostgreSQL、Redis，含健康检查和持久化 volume；Compose 不含 `build:`。
- PR #2 的 Go CI（test、vet、build）已通过 GitHub Actions。
- 默认管理员记录已初始化；Owner 登录/session API 已在 PR #3 实现，包含 12 小时会话、退出撤销、密码修改后撤销全部会话，以及 Redis 登录尝试限流；PR #3 CI/审查待确认。
- V5 登录页面和 API 客户端尚未接入真实后端，因此完整 UI 登录尚不可用。
- GHCR 镜像发布、Docker 首次安装/升级验收尚未完成；当前 Compose 引用的是待发布镜像。

## 最终交付与部署基线（强制）

用户明确要求两种互相独立的免编译部署方式，均为正式交付验收条件。

### A. Docker Compose 一键部署（内置数据库）

- Compose 必须同时编排 TG Group 应用、PostgreSQL 和 Redis；数据库及 Redis 使用持久化 volumes。
- GHCR 提供云端预构建应用镜像；Compose 正式文件只允许 `image:`，禁止 `build:`。
- 服务器只需 Docker Engine 与 Compose 插件；不得下载源码或安装编译工具链。
- 全新数据库首次启动自动迁移并初始化默认管理员：用户名 `admin`，密码 `123456`。
- 默认管理员只在无管理员记录时创建；升级、重启不得覆盖已有管理员密码。
- Owner 登录/session API 已在 PR #3 提交；仍需完成 V5 前端接入、GHCR 构建发布、完整安装/升级/回滚流程并验收。

### B. 宝塔面板 + Garble Go 二进制（网页安装向导）

- Release 提供对应 Linux OS/ARCH 的 Garble 混淆 Go 二进制；服务器不编译。
- 启动后通过浏览器访问安装向导。
- 向导填写 PostgreSQL 主机、端口、数据库名、用户名、密码及必要连接参数，并验证连接。
- 用户自行设置管理员用户名和密码；向导执行迁移、初始化基础配置并锁定重复安装入口。
- 若应用依赖 Redis，需允许配置已有 Redis 连接信息。

### Garble 构建约束

- 正式 Go 二进制必须由固定版本 Garble 构建；普通 `go build` 产物不能作为正式 Release。
- CI 记录 Go/Garble 版本、参数、commit、目标 OS/ARCH，发布 SHA-256 并运行 smoke test。
- Garble 是混淆，不是加密，不得宣传为绝对防逆向。

## 发布验收门槛

- [ ] V5 五页视觉复核与阶段一验收。
- [ ] 后端 Go test、vet、build CI 全绿。
- [ ] 登录/session API 与前端认证接通。
- [ ] Docker 首次部署自动启动应用、PostgreSQL、Redis。
- [ ] Docker 首次登录 `admin / 123456` 成功；升级/重启不重置密码。
- [ ] Docker Compose 无 `build:`，镜像 digest 可核验。
- [ ] PostgreSQL/Redis 数据持久化、健康检查、迁移和升级验证通过。
- [ ] 宝塔安装向导成功配置数据库、创建用户指定管理员并锁定安装入口。
- [ ] Garble 多平台构建、校验和与 smoke test 通过。

详细部署要求见 `docs/DEPLOYMENT.md`。文档规格和已提交的基础代码不代表正式部署交付已经完成；不得提前宣称完成。


## Cap 新版人机验证（PR #12，开发中）

- 对接新版自建 Cap：Widget 与 WASM 从固定版本 jsDelivr 加载，后端只向 Cap 的 `/{site-key}/siteverify` 发送验证请求，不依赖旧版 Widget files。
- 登录验证提供复选框、浮动、隐形三种可配置模式；Owner 登录时后端强制验证 token，验证失败/超时/Cap 服务不可用均拒绝登录。
- 平台设置仅需 Cap 服务域名、Site Key、Secret，另可启停验证并选择显示模式；Secret 使用 AES-256-GCM 加密入库，读取配置时不返回明文。
- 加密密钥优先读取 `CAP_SECRET_ENCRYPTION_KEY`，未设置时复用 `BOT_TOKEN_ENCRYPTION_KEY`；必须保持该密钥稳定并纳入安全备份。
- 登录页验证组件适配 Vben V5，复选框与浮动触发入口均限制在账号/密码输入框宽度内；隐形模式在提交时执行。
- [x] Go test/vet/build CI 通过（run 44840730682，commit ad50df4）。
- [x] V5 spacing、TypeScript、production build CI 通过（run 44840730584，commit ad50df4）。
- [ ] 真实浏览器连接用户自建 Cap 实例的端到端验收待部署配置后完成。
