# Changelog

## [0.1.24] - 2026-11-22

### 群管规则管理员豁免

- 群管策略执行前通过 Telegram `getChatMember` 查询消息发送者身份；`creator` 与 `administrator` 不受自动删除规则影响。
- 若 Telegram 无法确认发送者身份，则 fail-open 保留消息并记录警告，避免因权限/API 故障误删管理员消息。
- 新增 getChatMember 成功响应与 API 错误单元测试；Go/Vben CI 待本次提交后执行。
- 规则目前仍只检查普通文本，不覆盖媒体 caption 和 message entities；真实 Telegram 群验收仍待执行。


## [0.1.23] - 2026-11-22

### PR #22 Vben V5 TypeScript 修复与 CI 状态

- 修复群管规则页在群组列表为空时访问 `groups[0].id` 导致的 TS2532；改为先检查首个群组对象。
- Go test/vet/build 与 PostgreSQL 17、Redis 7 服务容器 CI 通过（run 44941649944）。
- Vben V5 spacing/架构检查通过；首次 TypeScript 检查失败，修复已提交，修复后 TypeScript、生产构建和 artifact 上传待复验（run 44941649890）。
- PR #22 仍为 Draft；Telegram 实群验收未执行，不标记为通过。

## [0.1.22] - 2026-11-22

### 群管规则配置与执行（PR #22）

- 新增群组级链接拦截、关键词规则持久化及 Owner 保护的读取/保存 API。
- Bot 运行时对普通文本消息执行策略匹配，命中后调用 Telegram `deleteMessage`。
- 规则默认关闭；当前不覆盖媒体 caption、message entities，也没有管理员豁免。
- 本次状态由 0.1.23 记录最新 CI 结果和 TypeScript 修复。


## [0.1.21] - 2026-11-22

### 订单管理 CI 修复

- 后端 Go test/vet/build 与 PostgreSQL 17、Redis 7 服务 CI 通过。
- 修复订单状态表格插槽的 `any` 类型索引问题。
- 后端 Go test/vet/build 与 PostgreSQL 17、Redis 7 服务 CI 通过（run 44940473483）。
- Vben V5 spacing/架构检查、TypeScript、production build 与 artifact 上传全部通过（run 44940473464）。
- CI artifact：`tg-group-admin-v5-phase1-dist`，SHA-256：`a23792a11955e2a5b0e42b7cc9cdf1341b524c3b0140155a34daedff8058f461`。
- Owner 登录后的订单页面浏览器验收仍待执行；PR #20 仍为 Draft，未合并 main。

## [0.1.19] - 2026-11-21

### 订单管理真实数据接入

- 新增 Owner 会话保护的订单列表 API，支持 Bot、状态、订单号/Telegram 用户 ID 筛选与服务端分页。
- V5 订单页移除演示数据，接入真实 Bot 列表与 PostgreSQL 订单查询。
- 金额保留数据库最小单位展示，不虚构订单类型或金额换算。
- Go/V5 CI 与浏览器验收待执行，不提前标记通过。


## [0.1.18] - 2026-11-21

### 运营总览真实数据接入（PR #19 开发中）

- 新增 Owner 会话保护的 `GET /dashboard/summary`，汇总真实 Bot、群组、群成员和最近订单数据。
- Dashboard 移除硬编码演示指标与虚构订单，接入真实 API，并提供加载、错误和空状态。
- 订单金额展示数据库 `amount_minor` 原始最小单位，避免在未明确单位换算前错误显示 USDT 金额。
- [x] Go test/vet/build 与 PostgreSQL 17/Redis 7 服务容器 CI 通过（Actions run 44882643375）。
- [x] Vben V5 spacing、TypeScript、production build 和 artifact 上传通过（Actions run 44882643424）。
- [ ] Owner 登录后的浏览器端真实数据验收待完成。

## [0.1.16] - 2026-11-21

### PR #17 集成分支 CI 状态校准

- 修正 PR #17 状态文档：Go/PostgreSQL/Redis、Vben V5、Docker 多架构 build-only、Garble 双架构构建及 QEMU smoke test 均已通过。
- 明确 PR #17 仍为 Draft，未合并 main；PR 构建不推送 GHCR，正式 Tag Release 和生产 digest 验收仍待执行。
- 记录用户已确认 Cap 当前可用，但不将未分别验证的 checkbox、floating、invisible 三模式 E2E 标记为全部通过。
- [x] Backend CI（Actions run 44845979907）。
- [x] Vben V5 validation（Actions run 44845979927）。
- [x] Docker multi-arch build-only（Actions run 44845979949）。
- [x] Garble + V5 assets + QEMU smoke tests（Actions run 44845980018）。

## [0.1.15] - 2026-11-21

### Vben V5 Owner 登录与新版 Cap 验证集成（PR #17，CI 待运行）

- 集成 V5 Owner 真实登录、登出、改密和新版 Cap 登录验证；支持 checkbox、floating、invisible 三种模式。
- Cap Widget 固定使用 jsDelivr `@cap.js/widget@0.1.63`，不使用旧版 Widget files，不引入 V2 兼容。
- 后端登录前强制调用 Cap `siteverify`；验证失败、超时、非 2xx 和无效响应均拒绝登录。
- Cap Secret 以 AES-256-GCM 加密入库，优先使用 `CAP_SECRET_ENCRYPTION_KEY`，回退到安装配置中持久化的 Bot Token AES-256-GCM 密钥。
- Cap 设置页支持域名、Site Key、Secret、启停和模式选择；移除设置页不需要的说明提示框。
- [x] Go test/vet/build 与 Cap 单元测试通过（Actions run 44845396660）。
- [x] Vben V5 spacing、TypeScript、production build 与 artifact 上传通过（Actions run 44845396655）。
- [x] Garble linux/amd64、linux/arm64 混淆构建、QEMU smoke test 与 V5 静态资源 PR artifacts 通过（Actions run 44845607727）；正式 Tag Release 未执行。
- [ ] 用户自建 Cap 实例三种模式浏览器端 E2E 通过。

## [0.1.14] - 2026-11-21

### 宝塔正式交付包：V5 静态资源与 systemd 服务（PR #16，CI 运行中）

- Garble Release 流水线新增独立 Vben V5 前端构建，正式资产包含 linux/amd64、linux/arm64 混淆二进制及 `tg-group-web-assets.tar.gz`。
- 静态资源构建固定 Node 22.18.0 / pnpm 11.16.0，并在构建后校验 `index.html` 存在。
- Release SHA256SUMS 覆盖两个架构二进制和前端静态资源包，BUILD-METADATA 记录 Go/Garble/Node/pnpm 版本。
- 新增带基础 systemd 安全限制的 `tg-group.service`，默认仅监听 loopback，便于宝塔 Nginx HTTPS 反向代理。
- [x] Vben V5 前端生产构建、静态资源打包、Garble linux/amd64/arm64 构建、QEMU smoke test 与 PR artifacts 通过（Actions run 44843856015；rebase 前等价 staging branch）。
- [ ] 正式 Tag 发布时的聚合 SHA256SUMS 与 GitHub Release 上传。
- [ ] 宝塔真实服务器安装、反向代理、首次初始化和升级验收。

## [0.1.13] - 2026-11-21

### 宝塔网页安装向导（PR #14，CI 已通过）

- 无数据库配置时启动安装专用页面，避免应用 API 和 Bot worker 在初始化前运行。
- 新增 PostgreSQL/Redis 连接测试、SSL mode 选择、数据库迁移和用户自设 Owner 初始化。
- Owner 密码只以 bcrypt hash 写入数据库；检测已有 Owner 时要求匹配原账号凭据，拒绝覆盖。
- 新数据库自动生成并持久化独立 32-byte Bot Token AES-256-GCM 加密密钥；数据库已有 Bot 记录时要求通过服务环境提供原密钥，避免破坏已有 Token 解密能力。
- 安装配置原子写入默认 `/var/lib/tg-group/config.json`，权限 0600；安装锁创建后拒绝再次访问安装入口。
- 首次启动生成 256-bit 随机安装令牌并以 0600 权限保存，安装 API 必须通过 `X-Install-Token` 校验；令牌在宝塔服务日志中仅作为首次引导凭据提供。
- 安装成功后需要通过宝塔面板重启服务，加载配置进入正常管理模式。
- [x] Go test/vet/build CI 通过；PostgreSQL 17 + Redis 7 安装集成测试通过（Actions run 44843294641）。
- [x] PostgreSQL 17/Redis 7 服务容器集成测试通过，覆盖首次安装、Owner bcrypt、配置文件/密钥持久化、安装锁及拒绝覆盖现有 Owner。
- [ ] 宝塔真实服务器守护进程重启、反向代理与浏览器登录验收。

## [0.1.12] - 2026-11-21

### Garble 多架构二进制构建与 Release 流水线（独立开发分支，待 CI）

- 固定 Go 1.23.12 与 Garble v0.14.2，针对 linux/amd64、linux/arm64 执行真实 Garble 混淆构建。
- 新增构建脚本，嵌入版本、commit、构建时间和目标架构信息，并为每个架构生成 SHA-256。
- 新增 `--version` 输出，CI 通过 QEMU 在对应目标架构容器中执行二进制 smoke test。
- Pull Request 仅构建并保存 CI artifacts；只有推送 `v*` tag 才创建 GitHub Release 并上传二进制、SHA256SUMS 和构建元数据。
- [x] PR #13 的 linux/amd64、linux/arm64 Garble build、QEMU smoke test、CI artifacts 和 Go test/vet/build 均通过；正式 Tag Release 尚未执行。
- [ ] 宝塔网页安装向导及安装锁仍待后续独立阶段实现。

## [0.1.11] - 2026-11-21

### Docker 一键安装、升级与 digest 回滚（PR #11，待验收）

- Compose 应用镜像改为 APP_IMAGE 环境变量，默认 ghcr.io/seanldnet/tg-group:latest，继续保持 image-only，不含 build:。
- 新增 scripts/deploy/install-docker.sh：检查 Docker daemon/Compose v2，下载并验证 Compose 配置，首次生成随机 PostgreSQL、Redis 和 Bot Token 加密密钥。
- 已存在 .env 时不覆盖，升级时保留管理员凭据、数据库密码、Redis 密码、Token 加密密钥和镜像 pin。
- 支持将 APP_IMAGE 固定为 GHCR 不可变 digest 后执行镜像回滚。
- 更新部署说明，明确 GHCR 私有包登录、安装/升级命令、digest 回滚与数据库迁移注意事项。
- [x] PR #10 Docker multi-arch build-only CI（linux/amd64、linux/arm64）通过。
- [ ] PR #11 shell 静态检查与 Compose contract CI。
- [ ] GHCR 正式镜像发布、全新安装、升级和 digest 回滚验收。


## [0.1.10] - 2026-11-21

### Docker 云端镜像构建与 V5 静态资源集成（PR #10，CI 待确认）

- 新增根目录多阶段 Dockerfile：Node/pnpm 云端构建 Vben V5 前端，Go 构建后端，最终运行镜像内置静态前端。
- Go 服务新增 `WEB_DIR` 静态目录配置；同一端口提供前端页面、`/api/*` API 和健康检查。
- V5 开发代理和生产 API 地址统一为 `/api`，关闭 Nitro mock。
- 移除 backend-only Dockerfile，统一使用根目录 Dockerfile 构建正式镜像。
- 新增 GHCR 多架构发布流水线：PR 只构建不推送；main/tag 发布版本标签、latest 与 SHA 标签，并输出 digest。
- Compose 仍只引用预构建镜像，不包含 `build:`。
- [ ] Docker multi-arch build CI 通过。
- [ ] 全新 Compose 部署、登录、静态资源/API 路由验收。
- [ ] GHCR 正式镜像发布与 digest 生产验收。

## [0.1.9] - 2026-11-21

### Telegram Bot 长轮询与群组/成员同步（PR #7，Go CI 全绿）

- 新增 Bot runtime manager，服务启动后每 10 秒对账 active Bot worker，并随服务停止取消 worker。
- 使用 Telegram `getUpdates` 长轮询接收 message、my_chat_member、chat_member 更新。
- 新增持久化 update offset；群组和成员使用 upsert 幂等同步。
- Telegram 返回无效 Token 时将 Bot 标记为 error 并停止对应 worker；临时故障退避重试。
- 新注册 Bot 在 Token 验证成功后进入 active，由 runtime 自动接管。
- [x] Go test/vet/build CI 全部通过。
- [ ] 真实 Telegram 群组、成员变更和重启 offset 验收。
- [ ] 反垃圾、欢迎语、自动封禁等群管功能仍待开发。

## [0.1.8] - 2026-11-21

### Telegram Bot 管理 API（PR #6，Go CI 全绿）

- 新增 Owner 鉴权保护的 Bot 列表、创建、重命名、删除 API。
- 创建 Bot 前调用 Telegram `getMe` 验证 Token；区分 Token 无效与 Telegram 网络/服务异常。
- Bot Token 使用 AES-256-GCM 加密后保存，接口仅返回 Token 掩码；新增 `BOT_TOKEN_ENCRYPTION_KEY` 配置及密钥持久化说明。
- 新增 Bot Telegram username 唯一索引、Bot 列表群组计数及加密相关测试。
- 新注册 Bot 保持 inactive；Telegram polling/update worker 仍待开发。
- [x] Go test/vet/build CI 全部通过。
- [x] V5 Bot 管理页面代码与 V5 CI 完成；真实端到端验收待 PR #6/#7 合并。

### Owner 认证与 V5 接入

- PR #3 Owner 登录、持久化 Bearer 会话、退出撤销、密码修改和 Redis 登录限流的 Go CI 已通过。
- PR #5 已提交 V5 登录页、用户信息、退出登录、密码修改真实 API 接入；V5 spacing、TypeScript 与生产构建 CI 已通过。
- [ ] 浏览器端完整登录、退出和密码修改验收。

## [0.1.7] - 2026-11-21

### Owner 登录与会话 API（PR #3，Go CI 已通过，代码审查待完成）

- 新增持久化随机 Bearer 会话，数据库仅保存 SHA-256 token hash，默认 12 小时过期。
- 新增 Owner 登录、当前用户信息、权限码、退出登录和修改密码 API。
- 修改密码验证当前 bcrypt 密码，更新后撤销该 Owner 的全部会话。
- 新增 Redis 登录尝试限流：每个来源 IP 15 分钟最多 5 次尝试。
- 新增会话表 migration 和认证基础单元测试。
- [x] PR #3 Go test/vet/build CI 通过。
- [ ] 数据库集成验证和安全审查通过。
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
