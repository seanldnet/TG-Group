# TG Group Manager — Project Status

- 文档版本：0.1.16
- 更新日期：2026-11-21
- 仓库：seanldnet/TG-Group
- 前端阶段分支：feat/phase-1-vben-v5（PR #1，Draft）
- 后端基础分支：feat/backend-foundation（PR #2，Draft，基于 PR #1）
- Owner 认证分支：feat/auth-session-api（PR #3，Draft，基于 PR #2）
- Bot 管理 API 分支：feat/bot-management-api（PR #6，Draft，基于 PR #3）
- Bot 运行时分支：feat/bot-runtime-worker-v2（PR #7，Draft，基于 PR #6）
- Docker 镜像交付分支：feat/docker-release-pipeline-v2（PR #10，Draft，基于 PR #7）
- V5 认证接入分支：feat/vben-auth-integration-v2（PR #5，Draft，基于 PR #3）
- V5 Bot 页面分支：feat/vben-bot-management（PR #8，Draft，基于 PR #5；V5 CI 全绿）
- 当前状态：PR #2、PR #3 的 Go test/vet/build CI 已通过；PR #5 V5 spacing、TypeScript 与生产构建 CI 已通过；PR #6 Go test/vet/build CI 已全部通过；PR #7 Bot 长轮询/群组成员同步 Go test/vet/build CI 已全部通过；PR #10 云端镜像/静态前端集成已提交，CI 验证中

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
- PR #5 已将 V5 登录、用户信息、退出登录和修改密码接入 Owner API，V5 spacing/TypeScript/生产构建 CI 全绿。
- PR #8 已将 Bot 页面接入真实 Bot 列表/创建/重命名/删除 API，V5 spacing/TypeScript/生产构建 CI 全绿；真实端到端验收待 PR #6/#7 后端合并。
- Dashboard、订单和统一支付页面仍为演示 UI，尚未接入真实 API。

### 后端基础（PR #2）

- 新增 Go HTTP 服务入口、配置加载、PostgreSQL 连接池、嵌入式 SQL migration 和迁移记录表。
- 初始 schema 包含唯一平台 Owner、Bot、群组、群成员、订单（保留 Bot 归属）及平台设置；不包含 Tenant/Agent/代理销售表。
- 全新数据库仅在不存在 Owner 时创建 bcrypt 管理员记录，默认用户名/密码为 `admin / 123456`；启动逻辑不更新已有账号密码。
- 新增 Redis 连接、存活/就绪探针及优雅关闭。
- 新增 image-only Compose：应用、PostgreSQL、Redis，含健康检查和持久化 volume；Compose 不含 `build:`。
- PR #2 的 Go CI（test、vet、build）已通过 GitHub Actions。
- 默认管理员记录已初始化；Owner 登录/session API 已在 PR #3 实现，包含 12 小时会话、退出撤销、密码修改后撤销全部会话，以及 Redis 登录尝试限流；PR #3 Go CI 已通过。
- PR #5 已提交 V5 Owner 登录、用户信息、退出登录和修改密码 API 接入；V5 spacing、TypeScript 与生产构建 CI 已通过；仍未完成浏览器端真实登录验收。
- PR #6 新增 Owner 认证保护的 Bot 列表/创建/重命名/删除 API；创建前通过 Telegram getMe 验证 Token，Token 使用 AES-256-GCM 加密存储，API 仅返回掩码；Go CI 全绿。
- PR #7 新增 Telegram getUpdates 长轮询 worker，自动同步群组和观察到的成员，持久化 update offset，Token 被 Telegram 拒绝时标记 Bot error；Go test/vet/build CI 全绿。
- PR #10 新增根目录多阶段 Dockerfile，云端构建 V5 静态资源与 Go 服务；同一端口提供前端、`/api/*` 和健康检查。
- PR #10 新增 GHCR 多架构构建/发布工作流：PR 只构建不推送，main/tag 发布 latest、版本/SHA 标签并记录 digest；CI 验证中，尚未实际发布镜像。
- 当前仅完成群组/成员同步，不包含反垃圾、欢迎语、自动封禁等群管规则执行。
- GHCR 发布工作流已在 PR #10 提交；实际 GHCR 镜像发布、Docker 首次安装/升级和 digest 验收尚未完成，Compose 仍引用待发布镜像。

## 最终交付与部署基线（强制）

用户明确要求两种互相独立的免编译部署方式，均为正式交付验收条件。

### A. Docker Compose 一键部署（内置数据库）

- Compose 必须同时编排 TG Group 应用、PostgreSQL 和 Redis；数据库及 Redis 使用持久化 volumes。
- GHCR 提供云端预构建应用镜像；Compose 正式文件只允许 `image:`，禁止 `build:`。
- 服务器只需 Docker Engine 与 Compose 插件；不得下载源码或安装编译工具链。
- 全新数据库首次启动自动迁移并初始化默认管理员：用户名 `admin`，密码 `123456`。
- 默认管理员只在无管理员记录时创建；升级、重启不得覆盖已有管理员密码。
- Owner 登录/session API 已在 PR #3 提交；PR #5 V5 登录/退出/改密接入 CI 全绿，浏览器验收待做；Bot 管理 API 在 PR #6、长轮询运行时在 PR #7。Docker 多阶段镜像与 GHCR 发布工作流在 PR #10，仍需实际镜像发布、首次安装/升级/回滚验收。

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
- [ ] V5 登录/session API 浏览器端真实验收。
- [ ] PR #7 Bot polling/update worker CI 与真实 Telegram 群组验收。
- [ ] 反垃圾、欢迎语、自动封禁等群管规则执行。
- [ ] Docker 首次部署自动启动应用、PostgreSQL、Redis。
- [ ] Docker 首次登录 `admin / 123456` 成功；升级/重启不重置密码。
- [ ] Docker Compose 无 `build:`，镜像 digest 可核验。
- [ ] PR #10 多架构镜像 CI 通过并实际发布 GHCR latest/version/SHA 标签。
- [ ] PostgreSQL/Redis 数据持久化、健康检查、迁移和升级验证通过。
- [ ] 宝塔安装向导成功配置数据库、创建用户指定管理员并锁定安装入口。
- [ ] Garble 多平台构建、校验和与 smoke test 通过。

详细部署要求见 `docs/DEPLOYMENT.md`。文档规格和已提交的基础代码不代表正式部署交付已经完成；不得提前宣称完成。

## 近期进展（0.1.11）

- PR #10 Docker multi-arch build-only CI（linux/amd64、linux/arm64）已通过；PR 构建不推送镜像，GHCR 正式发布仍待合并后执行。
- PR #11（feat/docker-one-click-install，基于 PR #10）新增 APP_IMAGE 可配置 Compose、Docker 一键安装/升级脚本和 digest 固定回滚说明。
- 安装脚本首次生成随机 PostgreSQL/Redis 密码与 Bot Token 加密密钥；检测到已有 .env 时不覆盖任何配置。
- PR #11 的 shell 静态检查、Compose contract CI、实际安装/升级和回滚验收尚未完成。
- 宝塔网页安装向导与 Garble 多平台 Release 仍待独立开发；Dashboard、订单、支付真实 API 和群管规则也仍待开发。

## Garble 二进制发布流水线（当前分支）

- 新增 `.github/workflows/garble-release.yml`：固定 Go 1.23.12 与 Garble v0.14.2，目标为 linux/amd64、linux/arm64。
- 新增 `scripts/release/build-garble.sh`：实际执行 `garble -literals -tiny build`，写入版本/commit/build 元数据并生成每个目标架构的 SHA-256。
- Go 服务新增 `--version` / `version` 输出，供二进制 smoke test 验证构建元数据；CI 使用 QEMU 在目标架构 Alpine 容器内启动二进制执行版本检查。
- Pull Request 只构建并上传短期 CI artifacts；只有推送 `v*` tag 才创建 GitHub Release，发布两种 Linux 架构二进制、聚合 SHA256SUMS 与构建元数据。
- [x] PR #13 Garble linux/amd64、linux/arm64 构建、QEMU 版本 smoke test、CI artifact 上传和后端 Go test/vet/build 均通过（Actions run 44841835411、44841835542）。
- PR #14（feat/baota-web-installer，基于 PR #13）新增宝塔网页安装向导：PostgreSQL/Redis 连通性测试、数据库迁移、Owner 创建/凭据校验、0600 配置文件持久化和安装锁；Go test/vet/build 与 PostgreSQL 17/Redis 7 集成 CI 已通过，宝塔真实服务器验收待做。
- [ ] 二进制需在最终集成的完整应用分支上重新构建；当前 PR 链仍基于 PR #11 的 Docker 安装链，尚未包含独立的 V5 认证/Cap 功能分支。


## 宝塔网页安装向导（PR #14，Go 与数据库集成 CI 已通过；真实宝塔验收待做）

- 首次启动未配置 `DATABASE_URL` 且找不到 `APP_CONFIG_FILE` 时进入安装专用 HTTP 模式，不会启动 Bot worker 或暴露管理 API。
- 新增中文安装页面，支持 PostgreSQL 主机/端口/数据库/用户名/密码/SSL mode，Redis 地址/密码/DB，以及用户自设 Owner 用户名/密码。
- 安装前可单独测试 PostgreSQL 与 Redis；提交安装时再次验证连接、执行嵌入式数据库迁移并初始化 Owner。
- Owner 密码长度要求 12–72 bytes，数据库只存 bcrypt hash；Owner 密码不写入配置文件。若数据库已有 Owner，必须提交匹配的用户名和密码，避免安装过程覆盖现有账号。
- 首次安装自动生成 32-byte Bot Token AES-256-GCM 加密密钥并写入 0600 配置文件；若绑定的数据库已有 Bot 记录，必须通过服务环境提供原 `BOT_TOKEN_ENCRYPTION_KEY`，否则拒绝安装，避免已有 Token 无法解密。
- 安装配置原子写入 `APP_CONFIG_FILE`（默认 `/var/lib/tg-group/config.json`），文件权限 0600；安装锁存在后安装页/API fail closed。首次启动还会生成独立 256-bit 安装令牌（同目录 `.install-token`，权限 0600），令牌通过宝塔服务日志提示，并通过 `X-Install-Token` 保护安装 API。
- 安装成功后需在宝塔面板重启服务，应用读取配置文件进入正常运行模式；Docker Compose 的 `DATABASE_URL` 环境变量启动路径保持不变。
- [x] PR #14 Go test/vet/build CI 全绿，PostgreSQL 17 + Redis 7 服务容器的安装集成测试通过（Actions run 44843294641）。
- [x] CI 已通过真实 PostgreSQL 17/Redis 7 容器的首次安装、Owner bcrypt、配置密钥持久化、安装锁和已有 Owner 密码保护集成测试；[ ] 宝塔面板实际守护进程重启、反向代理和浏览器登录验收待真实部署。
- [ ] 最终 Garble Release 需在集成 V5/Cap 和安装向导的完整分支重新构建。


## 宝塔正式交付包（PR #16，CI 运行中）

- Garble Release 工作流新增独立 Vben V5 前端构建任务，打包 `tg-group-web-assets.tar.gz`；正式 Release 同时提供 linux/amd64、linux/arm64 Go 二进制、V5 静态资源包、SHA256SUMS 和 BUILD-METADATA.txt。
- Release 静态资源构建固定 Node 22.18.0、pnpm 11.16.0；前端执行 `pnpm --filter @vben/web-antd run build`。
- 新增 `scripts/deploy/tg-group.service`：以专用无登录用户运行，监听 127.0.0.1:8080，配置文件写入 /var/lib/tg-group，启用 systemd 基础安全限制。
- 部署文档补充二进制/静态资源安装、首次安装令牌读取、Nginx HTTPS 反向代理、服务重启和升级步骤。
- [x] Garble + V5 静态资源构建、linux/amd64/arm64 混淆构建、QEMU smoke test 和 PR artifacts 全部通过（Actions run 44843856015；在 rebase 前的等价 staging branch 上运行，PR #16 保留相同 workflow 实现）。
- [ ] 宝塔真实服务器上传、解包、服务启动、安装向导、登录和升级验收。
- [ ] 最终发布必须在集成 V5 登录/Cap 的完整分支重新构建；本分支前端资源只代表当前基线，不代表 PR #12 Cap 已集成。


## Vben V5 Owner 登录与新版 Cap 验证集成（PR #17，CI 全绿）

- 将 PR #5 的真实 Owner 登录、退出、修改密码流程与 PR #12 的新版 Cap 实现集成到 Docker/宝塔交付链。
- 登录页读取公开 Cap 配置；启用时提供 checkbox、floating、invisible 三种模式，提交时携带一次性 captchaToken；每次登录尝试后重新挂载控件。
- Cap Widget 固定从 jsDelivr 加载 `@cap.js/widget@0.1.63`；不使用旧版 Widget files，不引入 V2 兼容层。
- 平台设置支持 Cap 服务域名、Site Key、Secret、启停和模式选择；已删除设置页中用户不需要的两组说明提示框。
- Go 登录接口在校验账号密码前调用 Cap `siteverify`；超时、非 2xx、无效 JSON、success=false 均拒绝登录。
- Cap Secret 使用 AES-256-GCM 加密存储；密钥优先 `CAP_SECRET_ENCRYPTION_KEY`，默认回退到安装向导生成并持久化的 `BOT_TOKEN_ENCRYPTION_KEY`。
- 新增 Cap URL 校验、Secret 加解密、siteverify 请求/响应单元测试；PR #17 CI 待运行。
- [x] Go test/vet/build 与 Cap 单元测试通过（Actions run 44845396660）。
- [x] Vben V5 spacing、TypeScript、production build 和 artifact 上传通过（Actions run 44845396655）。
- [ ] 浏览器配置用户自建新版 Cap 后完成 checkbox/floating/invisible 三模式真实 E2E。
- [x] PR #17 的 Vben V5 静态资源包已成功构建并上传（Garble run 44845396665 的 web-assets job）。
- [x] Garble linux/amd64、linux/arm64 混淆构建、QEMU 版本 smoke test、前端静态资源打包和 PR artifacts 全部通过（Actions run 44845607727）；正式 Tag Release 未执行。
- [x] Docker 多架构 build-only 验证通过（Actions run 44845979949）；PR 构建未登录 GHCR、未推送镜像。


## PR #17 集成分支最新验证校准（0.1.16）

- PR #17 head：`a0bf48f894117cbc3031d4ec0e7414dddf61c29a`；PR 仍为 Draft，base 为 `feat/baota-release-bundle-v2`，未合并到 main。
- [x] Go test、vet、build 与 PostgreSQL 17 / Redis 7 服务容器集成验证通过（Actions run 44845979907）。
- [x] Vben V5 自动间距/单 Owner 架构检查、TypeScript、生产构建及 artifact 上传通过（Actions run 44845979927）。
- [x] Docker image-only Compose contract 与 linux/amd64、linux/arm64 多架构 build-only 通过（Actions run 44845979949）；PR 不发布 GHCR 镜像。
- [x] V5 静态资源构建、Garble linux/amd64 与 linux/arm64 混淆构建、QEMU smoke test、PR artifacts 通过（Actions run 44845980018）；正式 Tag Release 未执行。
- 用户已确认 Cap 人机验证当前使用正常；但尚无分别覆盖 checkbox、floating、invisible 三模式的可复核浏览器 E2E 记录，因此不将三模式完整验收标记为通过。
- [ ] 浏览器端 Owner 登录、登出、改密完整验收。
- [ ] 用户自建 Cap 实例三种模式逐项 E2E 验收。
- [ ] Docker 正式 GHCR 发布、生产 digest 验收及宝塔真实服务器验收。
