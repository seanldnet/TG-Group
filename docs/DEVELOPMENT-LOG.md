# TG-Group 开发日志

## 2026-11-26（平台注册管理 Bot：Telegram 自助注册第一阶段）

- **业务实现**：平台专用 Telegram Bot 收到用户私聊 `/start` 后，调用认证服务创建普通平台账号；Telegram ID 在创建事务中直接写入用户记录，避免先注册再绑定的竞态。为账号生成随机初始密码，bcrypt 哈希保存；Bot 私聊一次性返回用户 UUID 和初始密码。
- **账号登录**：网站登录接口现在允许使用用户 UUID 或邮箱作为账号标识；登录页面提示支持“邮箱、用户 ID 或管理员账号”。Telegram 注册用户可用 Bot 返回的用户 ID + 初始密码登录。
- **重复注册保护**：使用 PostgreSQL advisory transaction lock 串行化相同 Telegram ID 的注册；数据库唯一约束作为最终保护。重复 /start 不创建第二个账号，提示用户使用原账号。
- **身份绑定兼容**：保留网站注册用户的 `/start <challenge>` 一次性绑定流程；Bot 只处理用户私聊，忽略群聊和机器人账号；绑定失败会返回明确提示。
- **测试**：新增 PostgreSQL 集成测试覆盖 Telegram 注册、重复注册拒绝、用户 ID 登录和身份绑定检查。需等待 CI 中 Go/PostgreSQL 测试、go vet 与前端构建结果；未在真实 Telegram Bot 上联调。
- **安全说明**：初始密码仅由 Bot 私聊发送一次，日志不记录密码或 Token；平台 Bot 仍由 `TELEGRAM_AUTH_BOT_TOKEN` 与 `TELEGRAM_AUTH_BOT_USERNAME` 环境变量配置，管理员可视化配置尚未实现。注册菜单、套餐查询/购买、提交群管 Bot Token 等流程仍待开发。
- **涉及文件**：`internal/auth/telegram_register.go`、`internal/auth/service.go`、`internal/auth/service_integration_test.go`、`internal/telegramauthbot/poller.go`、`src/pages/LoginPage.tsx`、`docs/API.md`、`docs/DEVELOPMENT-PLAN.md`。
- **下一步**：完成管理员可视化平台 Bot 配置（加密存储与运行时安全重载），再开发注册 Bot Inline Keyboard 主菜单、用户/套餐/群管 Bot 账户管理对话，并补齐真实 Bot 联调。


## 2026-11-26（管理后台导航与登录设备会话修复）

- **用户反馈**：已登录工作台没有持久侧边栏，点击品牌区域会回到公开首页；登录设备列表把同一浏览器每次 15 分钟 Token 续期都记为一台新设备；用户添加的群管 Bot 已保存但没有通用指令菜单。
- **前端导航**：新增 `src/components/DashboardLayout.tsx`，使用仓库 shadcn/ui 官方 Button、Badge、Separator 组合统一控制台框架；桌面端固定侧边栏、移动端抽屉导航，提供首页、机器人、群组、账号安全、登录设备和安全日志入口。所有 /dashboard 路由统一套用布局。
- **首页跳转修复**：Dashboard 品牌链接由公开首页 `/` 改为受保护工作台 `/dashboard`；登录页检测当前浏览器已有的有效会话，自动恢复进入工作台，避免用户从公开首页再次点击登录后重复输入密码。
- **会话修复**：Refresh Token 轮换不再每 15 分钟插入一条新的 `user_sessions`；改为在同一个 refresh family 内更新现有设备会话的 access token 摘要、过期时间和最近 User-Agent/IP，设备记录 ID 保持不变。历史上已生成的重复记录不会自动合并或删除。
- **测试**：扩展 PostgreSQL 集成测试，断言 refresh 前后的当前设备 session ID 保持一致；需要 CI 的 Go/PostgreSQL 测试、go vet 和前端构建结果确认。
- **尚未完成**：平台管理员设置页面及平台注册/身份 Bot 的数据库配置尚未实现；平台身份 Bot 与用户群管 Bot 是不同职责，不应混用同一 Token。群管 Bot 当前运行时主要处理群组发现、消息记录、已配置规则和验证回调，并没有完整通用命令菜单/双端管理工作流，因此“添加成功”不代表已具备完整指令交互。真实 Telegram Bot 联调仍需逐项验收。
- **涉及文件**：`src/App.tsx`、`src/components/DashboardLayout.tsx`、`src/pages/DashboardPage.tsx`、`internal/auth/refresh.go`、`internal/auth/service_integration_test.go`、`docs/DEVELOPMENT-PLAN.md`、`docs/DEVELOPMENT-LOG.md`。
- **下一步**：验证 CI；继续建设平台管理员设置和 Telegram 注册管理 Bot 对话流程，再逐项完成群管 Bot 命令菜单及功能双端闭环。


所有代码、配置、依赖、测试、修复和部署变更均须在此记录。只记录已实际发生并可验证的工作，不将计划标为完成。

## 2026-11-25

### 文档初始化
- **变更**：建立产品需求基线和开发计划。
- **新增文件**：
  - `docs/REQUIREMENTS.md`：产品定位、注册入口、账号唯一性、多租户、群管功能矩阵、双端等价要求、支付、CAP、赞助商、AI、shadcn/ui 强制规范、Docker 交付和验收纪律。
  - `docs/DEVELOPMENT-PLAN.md`：按工程底座、身份租户、支付、Bot/群组、群管双端、CAP/赞助商/AI、部署交付拆分阶段。
- **仓库**：`seanldnet/Tg-group`，默认分支 `main`。
- **提交**：`3dcd7d525a464bf81457a38750272853d087c1c6`（需求文档）。
- **测试**：已确认需求文档写入 GitHub 成功；尚未有应用代码，因此无构建/功能测试结果。
- **状态**：文档基线已创建；应用开发尚未开始。
- **待办**：写入开发计划与本日志；开始 React + TypeScript + shadcn/ui 工程底座。后端技术栈仍需确认，不在未确认情况下擅自选择。


### 前端工程底座（第一步）
- **变更**：向仓库写入 React + TypeScript + Vite 前端基础文件。
- **新增文件**：`package.json`、`index.html`、`tsconfig.json`、`tsconfig.app.json`、`tsconfig.node.json`、`vite.config.ts`、`src/main.tsx`、`src/App.tsx`、`src/index.css`、`.gitignore`、`README.md`。
- **UI 约束**：当前仅放置工程占位入口，没有自建业务 UI 控件；后续界面必须通过 shadcn/ui 官方 CLI 添加的官方组件组合实现。
- **官方依据**：shadcn/ui Vite 安装指南与 CLI 文档： https://ui.shadcn.com/docs/installation/vite 、https://ui.shadcn.com/docs/cli 。当前官方流程采用 `shadcn@latest` CLI。
- **测试**：尚未执行 `pnpm install`、shadcn CLI、typecheck 或 production build。尝试在执行环境克隆 GitHub 仓库时因 DNS/网络不可用失败，因此不能声称构建通过。
- **状态**：基础文件已写入 GitHub；shadcn/ui CLI 初始化和官方组件落盘仍待执行；后端尚未开始。
- **下一步**：在可联网的构建/开发环境运行官方 shadcn CLI，安装官方组件并完成构建验证；随后继续建立前端页面框架。后端技术栈需先确认。


### 需求条款澄清
- **变更**：将“首次登录强制修改密码”明确标记为安全建议/待项目所有者确认，避免把尚未明确确认的建议写成已定需求。
- **文件**：`docs/REQUIREMENTS.md`。
- **测试**：已通过 GitHub 文件读取/更新确认写入；未涉及代码构建。


### 后端语言确认
- **变更**：项目所有者明确指定后端使用 Go（Golang）。
- **文件**：同步更新 `docs/REQUIREMENTS.md`、`docs/DEVELOPMENT-PLAN.md` 与 `README.md`。
- **技术边界**：Go 作为后端语言已确定；HTTP 框架/路由库、模块边界及异步任务实现仍需结合 Telegram Bot、Webhook、队列、支付与 AI 适配需求评估，不在未评估前擅自固化。
- **测试**：本次仅更新项目文档，尚未创建 Go 源码、安装依赖或运行 Go 测试；不代表后端工程已搭建。
- **下一步**：完成 Go 后端架构选型并建立可运行工程骨架，再配置 PostgreSQL/Redis 连接、迁移和基础健康检查。


### Go 后端基础服务（第一批代码）
- **变更**：建立 Go 模块与最小可运行 HTTP 服务，使用 Go 标准库 `net/http`，未引入第三方 Go Web 框架。
- **新增文件**：
  - `go.mod`
  - `cmd/api/main.go`：服务启动、JSON 结构化日志、SIGINT/SIGTERM 监听及优雅停机。
  - `internal/config/config.go`：读取 HTTP 端口和运行环境，校验端口范围。
  - `internal/config/config_test.go`：默认配置与非法端口测试。
  - `internal/httpapi/handler.go`：`GET /healthz` 健康检查端点。
  - `internal/httpapi/handler_test.go`：健康检查响应和未知路由 404 测试。
- **运行配置**：默认监听 `:8080`；可通过 `TG_GROUP_HTTP_PORT` 与 `TG_GROUP_ENV` 配置。
- **验证**：在临时本地工作目录按仓库源码建立同一 Go module，执行 `gofmt -w .`、`go test ./...`、`go vet ./...`，均通过。配置包 2 项测试、HTTP API 包 2 项测试通过；main 包无独立测试。
- **边界**：本批仅完成服务底座和健康检查；尚未接入 PostgreSQL、Redis、认证、租户、Telegram Bot 或业务 API。临时目录验证不等于已在 GitHub checkout 中构建。
- **下一步**：建立数据库迁移与 PostgreSQL/Redis 连接层，再实现认证和租户模型；前端继续严格按 shadcn/ui 官方 CLI 生成组件。


### PostgreSQL 身份数据模型与首页内容（第二批）
- **新增**：`migrations/000001_identity.up.sql`、`migrations/000001_identity.down.sql`，建立 tenants、users、user_sessions、audit_logs 四张基础表，包含 Telegram ID / email 唯一约束及租户关联。
- **新增**：`internal/database/database.go`，使用 pgx/v5 建立连接池、Ping 和嵌入式版本迁移执行器；迁移版本写入 schema_migrations，重复启动不会重复执行已登记迁移。
- **新增**：`internal/database/migrations/000001_identity.up.sql`，供 Go embed 在二进制内携带迁移文件。
- **修改**：配置增加 DATABASE_URL；服务启动时配置了数据库地址便连接 PostgreSQL 并自动执行迁移，未配置则保留 API-only 模式并输出警告。
- **新增**：`docs/HOMEPAGE-CONTENT.md`，首页首屏、产品价值、功能介绍和建设中状态文案；明确不宣传尚未上线的功能。
- **新增**：`.github/workflows/ci.yml`，推送 main / PR 时执行 gofmt 检查、go test 和 go vet。
- **验证边界**：GitHub 文件写入成功；当前执行环境有 Go 1.23.2，但 pgx/v5 与 x/crypto 依赖未缓存且无法访问外网，因此本批尚未在本地完成编译、测试。CI 工作流已写入，尚无云端运行结果，不能标记测试通过。
- **前端首页状态**：已完成首页介绍文案，尚未生成实际页面。由于当前环境没有 pnpm 且无法访问 npm，shadcn/ui 官方 CLI 仍未运行；为遵守“只能使用官方 shadcn/ui 组件”的要求，本批没有自行编写任何 UI 控件或伪造首页组件。
- **下一步**：补齐数据库集成测试与注册/登录 API，再在可联网环境执行官方 shadcn CLI，落地首页及登录注册页面。


### 注册登录 API 与会话认证（第三批）
- **新增**：`internal/auth/service.go`，实现邮箱注册、bcrypt 密码哈希、登录校验、随机 Bearer 会话 token、仅保存 token SHA-256 摘要、会话认证和退出撤销；会话有效期 30 天。
- **新增**：`internal/auth/service_test.go`，覆盖邮箱校验、随机 token 和客户端 IP 解析的单元测试。
- **修改**：`internal/httpapi/handler.go`，新增注册、登录、当前用户、退出登录接口；JSON 请求限制 1 MiB、拒绝未知字段和多余 JSON 内容；未配置数据库时返回 503。
- **修改**：`cmd/api/main.go`，数据库启动成功后注入认证服务。
- **修改**：身份迁移允许一个租户关联多个用户；注册仍为每个新用户创建独立租户，后续可扩展租户邀请/成员管理。
- **新增**：`docs/API.md`，记录当前已实现 API 的路径、参数、鉴权方式和未实现边界。
- **CI 修复**：上一轮格式检查失败原因是部分 Go 源码未 gofmt；CI 改为执行 gofmt、go mod tidy，再执行 go test 和 go vet。新一轮云端任务已启动，当前等待结果。
- **安全边界**：当前尚未实现 refresh token、密码找回、邮箱验证、登录限流、Telegram ID 绑定和业务角色授权；认证代码尚未完成 PostgreSQL 集成测试，不能宣称端到端认证已验收。
- **前端**：仍只有工程占位入口；shadcn/ui 官方 CLI 尚未执行，不能开始编写页面 UI 以免违反官方组件约束。


### shadcn/ui 官方组件落地与首页页面（第四批）
- **官方 CLI 实际执行**：通过 GitHub Actions 的 Node 22 / pnpm 10 环境运行 `pnpm dlx shadcn@latest init --base radix --preset nova --yes`，并运行官方 `add` 命令添加基础组件；不是手工复制或仿写组件。
- **官方生成文件**：`components.json`、`pnpm-lock.yaml`、`src/components/ui/` 下的 Badge、Button、Card、Dialog、DropdownMenu、Input、Label、Select、Separator、Skeleton、Sonner、Switch、Table、Tabs、Textarea、Tooltip。
- **路径修正**：首次 CLI 生成时发现根 `tsconfig.json` 缺少 paths，导致官方文件落在错误的 `@/` 目录；已将 `@/* -> ./src/*` 同步到根 tsconfig，并重新通过官方 CLI 生成到 `src/components/ui/`，错误目录已清理。
- **首页实现**：`src/App.tsx` 已改为 TG-Group 产品介绍首页，包含首屏、功能模块、产品原则、开发状态和 GitHub 入口；基础按钮、徽标和卡片全部导入官方 shadcn/ui 组件。页面明确标注产品示意与建设中状态，不把未接入的 Bot/群组数据伪装成真实数据。
- **构建验证**：官方组件生成后的 CLI 工作流完成 `pnpm build`；首页提交后的 CI 前端 job（Node 22、pnpm install --frozen-lockfile、`pnpm build`）已通过。
- **后端验证**：Go + PostgreSQL 集成测试已通过，覆盖注册、邮箱重复、登录、token 认证和退出撤销；`go vet ./...` 通过。
- **CI 调整**：`.github/workflows/ci.yml` 现包含 Go/PostgreSQL 后端测试与前端生产构建两个 job。
- **下一步**：实现前端注册/登录页面并接通当前认证 API；随后补充 refresh token、密码找回、邮箱验证、登录限流和 Telegram ID 绑定。


### 网站注册/登录与账号工作台（第五批）
- **新增**：`src/lib/api.ts`，封装注册、登录、当前用户和退出登录请求；会话 token 暂存在浏览器 sessionStorage。
- **新增**：`src/pages/LoginPage.tsx`、`src/pages/RegisterPage.tsx`，使用官方 shadcn/ui Input、Label、Button、Card、Badge 组件构建表单并连接 Go API。
- **新增**：`src/pages/DashboardPage.tsx`，登录后查询当前用户资料，展示账号/租户信息，并支持退出登录。
- **修改**：`src/App.tsx` 增加首页、`/login`、`/register`、`/dashboard` 路径分发；首页增加登录和注册入口。
- **修改**：`vite.config.ts` 增加开发环境 `/api` 代理，默认转发到 `127.0.0.1:8080`，可用 `TG_GROUP_API_PROXY` 覆盖。
- **新增**：`compose.dev.yml` 提供 PostgreSQL 16 与 Redis 7 本地依赖；`Makefile` 提供常用开发命令；`docs/LOCAL-TESTING.md` 说明本地启动与测试流程。Redis 目前仅作为开发依赖，尚未由 Go 服务使用。
- **验证**：前端 CI job 已完成 `pnpm install --frozen-lockfile` 和 `pnpm build`，通过。Go/PostgreSQL 集成测试、`go vet ./...` 与前端 `pnpm build` 均在 GitHub Actions run `45633604003` 中通过；该 run 同时验证了注册/登录页面的 TypeScript 编译与生产构建。
- **安全边界**：前端 token 当前存储于 sessionStorage，属于第一版开发实现；后续应评估改为 HttpOnly Secure Cookie，并补充 CSRF 防护、限流、邮箱验证与 refresh token。


## 2026-11-26

### Telegram Bot Token 管理 API（第六批，首个可管理闭环）
- **新增迁移**：migrations/000002_telegram_bots.up.sql 与 internal/database/migrations/000002_telegram_bots.up.sql，创建租户归属的 Bot 登记表，Bot Telegram ID 与 Token 指纹全局唯一；保存 AES-GCM 密文与 nonce，不保存 Token 明文。
- **新增服务**：internal/bots/service.go，调用 Telegram Bot API getMe 验证 Token，实现加密保存、租户范围列表、启停状态、同一 Bot 身份的 Token 轮换、解绑及供内部运行时读取的租户范围解密接口。
- **新增 API**：GET/POST /api/v1/bots、PATCH /api/v1/bots/{botID}/status、PUT /api/v1/bots/{botID}/token、DELETE /api/v1/bots/{botID}；均要求 Bearer 会话并以当前用户 tenant_id 作为数据边界。API 响应不返回 Token。
- **配置**：新增 BOT_TOKEN_ENCRYPTION_KEY，必须为 Base64 编码的 32 字节密钥。未配置时 Bot 管理关闭；密钥必须持久保存，不能每次重启重新生成。
- **数据库迁移并发保护**：迁移执行器增加 PostgreSQL advisory lock，避免多个实例/测试并行启动时同时执行同一迁移。
- **测试**：新增 AES-GCM 密文篡改检测、密钥格式单元测试，以及 PostgreSQL Bot 添加/列表/启停/轮换/解绑和跨租户隔离集成测试（依赖 CI 提供 TEST_DATABASE_URL）。
- **文档**：更新 docs/API.md、docs/LOCAL-TESTING.md，记录 API、加密密钥和未完成的 Bot 运行时边界。
- **安全/功能边界**：目前仅完成 Bot 资产登记与管理；尚未实现长轮询/Webhook worker、群组自动发现、Bot 管理员权限检查或群管消息处理。平台侧 active/disabled 状态暂不等于实际 Telegram 收发状态。
- **验证状态**：变更已推送 main，GitHub Actions CI 正在执行；待确认 Go 集成测试、Vet 和前端构建结果后再更新为通过。


### Telegram Bot 管理前端（第七批）
- **新增**：src/pages/BotsPage.tsx，使用已由 shadcn/ui 官方 CLI 生成的 Dialog、Input、Label、Button、Badge、Card、Table 组件，提供 Bot 列表、添加 Token、启停、轮换和解绑操作。
- **修改**：src/lib/api.ts 增加 Bot API 类型与请求方法；src/App.tsx 增加 /dashboard/bots 路由；Dashboard 工作台增加 Bot 管理入口。
- **界面安全**：Token 输入框使用 password 类型；API 响应不含 Token；列表不展示密钥。解绑前要求用户确认。
- **CI 发现并修复**：前端首次构建发现 authenticatedHeaders 返回值推断类型不满足 HeadersInit；明确标注 Record<string,string> 后，GitHub Actions run 45704609255 的前端 build 已通过。
- **后端验证**：GitHub Actions run 45704549842 中 Go 集成测试与 go vet 已通过，覆盖 Bot 生命周期、密文不落明文和跨租户查询隔离。之后补充了停用 Bot 禁止运行时读取 Token 的保护及对应测试，更新后的完整 CI 仍在执行。


### Bot 操作审计与停用保护补强
- **修改**：Bot 添加、启停、Token 轮换、解绑均在同一 PostgreSQL 事务内写入 audit_logs；任一审计写入失败，业务事务回滚，避免操作成功但没有审计记录。
- **修改**：内部 GetToken 仅允许 active 状态的 Bot 读取明文 Token；disabled 状态无法被运行时取出 Token。Telegram 返回 429 或 5xx 时映射为服务暂不可用，不误报 Token 无效。
- **测试**：PostgreSQL 集成测试补充停用状态禁止读取 Token、五类 Bot 生命周期审计记录数量校验；HTTP 测试补充 Bot 服务未配置时各管理路由返回 503。
- **验证状态**：GitHub Actions run 45705033278（commit 7e397920a656adabc7426885767d317d4988e542）Go 测试、PostgreSQL 集成测试、go vet 与前端生产构建均通过；CI 两个 job 均为 success。

- **验收结果补充**：Bot 生命周期的添加、启停、Token 轮换和解绑均以事务方式写入 audit_logs；CI 已验证完整生命周期审计记录数和跨租户隔离。

### Telegram 长轮询运行时与群组自动发现（第八批）
- **新增迁移**：migrations/000003_telegram_groups.up.sql 与 internal/database/migrations/000003_telegram_groups.up.sql；为 Bot 保存 update offset，并新增按租户、Bot 隔离的群组资产表。
- **新增运行时**：internal/telegramruntime/poller.go；API 进程启动后自动扫描 active Bot，为每个 Bot 启动独立 getUpdates 长轮询 worker；每 15 秒协调新增、停用和解绑的 Bot worker。
- **更新处理**：订阅 message、edited_message、channel_post、my_chat_member；识别 group/supergroup/channel 并 upsert 群组资产；每条 update 成功持久化后推进 offset，数据库写入失败时不推进 offset，避免丢失群组事件。
- **生命周期**：长轮询 worker 使用服务 context，收到 SIGINT/SIGTERM 时随 API 优雅停机；Token 不写日志，Telegram 请求错误不输出包含 Token 的 URL。
- **测试**：新增 poller 单元测试，覆盖私聊过滤、普通群/超级群/频道识别和 update chat 提取。
- **功能边界**：本批采用 Telegram getUpdates 长轮询模式；尚未实现 Webhook 模式切换、群组列表管理 API/前端、管理员权限检测、成员事件处理与群管规则执行。Bot 必须被添加到群组/频道并产生受支持的更新，才能被发现；频道更新还取决于 Bot 的权限与 Telegram 投递行为。
- **验证状态**：源码和文档已提交 main；等待 GitHub Actions 对 Go 编译、单元/集成测试、Vet 与前端构建执行结果。未进行真实 Telegram Token 联调，不能宣称真实群组接收已验收。


### 群组查询 API 与网站列表页（第九批）
- **后端**：新增 `GET /api/v1/groups`，从 telegram_groups 查询当前用户租户的群组资产；SQL 强制 tenant_id 过滤，不接受客户端指定租户。
- **前端 API**：src/lib/api.ts 增加 TelegramGroup 类型和 listGroups 请求方法。
- **前端页面**：新增 src/pages/GroupsPage.tsx，基于已由官方 shadcn/ui CLI 生成的 Card、Badge、Button、Table 展示群组标题、Chat ID、类型、来源 Bot 和最近发现时间；增加空状态与刷新。
- **工作台导航**：/dashboard 增加群组列表入口；新增 /dashboard/groups 路由。
- **文案修正**：Bot 管理页面说明更新为后台长轮询运行时已启动，不再显示“尚未启动消息收发 worker”。
- **测试**：HTTP handler 测试覆盖未配置 Bot 服务时群组列表 API 返回 503；poller 单元测试覆盖群组类型识别。
- **验证状态**：代码与文档已提交 main；GitHub Actions Go/PostgreSQL 测试和前端构建正在运行，尚未将本批标记为 CI 通过。真实 Telegram 群组联调仍需部署环境中的有效 Bot Token。


### 群组离开/封禁状态追踪
- **更新解析**：读取 my_chat_member.new_chat_member.status；Bot 离开群组时将群组状态更新为 left，被封禁时更新为 kicked，重新加入后恢复 active。
- **测试**：新增会员状态转换单元测试，覆盖 member、administrator、left、kicked。
- **验证**：已写入 main；Go CI 对本次最新代码仍在执行，等待最终结果。


### Telegram worker 关闭顺序修复
- **修改**：API 收到终止信号后，除关闭 HTTP server 外，还等待 Telegram polling manager 取消并回收各 Bot worker，再允许 PostgreSQL 连接池关闭；避免 worker 在数据库已关闭后仍执行收尾操作。
- **测试状态**：本次变更已推送 main，GitHub Actions CI 正在执行。


### Telegram 群消息事件日志（第十批）
- **数据库**：新增 `internal/database/migrations/000004_telegram_message_events.up.sql`，建立租户、Bot、群组关联的消息事件表；记录 update/message ID、发送者 Telegram ID/用户名/显示名称、消息类型、文本/Caption、Telegram 发送时间与平台接收时间，并为群组时间线和发送者查询建立索引；增加对应 down SQL。
- **运行时**：扩展 Telegram Update 解析，处理 message、edited_message、channel_post；群组事件先 upsert 群组，再幂等写入消息日志，数据库写入失败不推进 update offset。
- **内容边界**：文字与 Caption 最多保存 4096 字符；图片/文件/视频/贴纸/语音等只保存类型和 Caption，不下载媒体文件。当前没有自动保留期限，删除 Bot/群组时外键级联清理日志。
- **后端 API**：新增 `GET /api/v1/groups/{groupID}/events?limit=50`，limit 允许 1–100，SQL 按当前租户与群组 ID 双重过滤。
- **前端**：新增 GroupEventsPage，群组列表增加“消息日志”入口；展示发送者、Telegram 用户 ID、消息类型、内容和时间，组件继续使用官方 shadcn/ui。
- **测试**：新增消息类型/发送者名称解析单测；PostgreSQL 集成测试覆盖消息日志查询和跨租户隔离；HTTP 测试覆盖服务未配置时日志接口返回 503。
- **文档**：更新 API、开发计划和本地测试说明，明确消息内容存储范围及尚未实现的保留期限/搜索/分页/导出。
- **验证状态**：所有代码与文档已写入 main；尚未取得本批最新 GitHub Actions 最终结果，因此暂不标记 CI 通过。真实 Telegram 消息收发仍需测试 Bot 和测试群组联调。


- **日志内容安全修正**：消息文本上限按 Unicode 字符截断，避免按 UTF-8 字节截断导致多字节字符损坏；仍限制单条日志文本最大 4096 个字符。
- **Telegram 可见性说明**：API 文档补充 Bot 群组隐私模式限制；若需要完整群消息日志，需按 Telegram 官方规则关闭隐私模式或授予 Bot 群管理员权限，不能仅凭 Bot 已加入群组就假设能收到全部普通消息。
- **CI 验证结果**：GitHub Actions run `45709322432`（代码提交 `8a0e5c400e1b3db12d7f6e4780f1da3d44a55679`）的 Go/PostgreSQL `go test ./...`、`go vet ./...` 和前端 `pnpm build` 均已通过；消息日志集成测试和跨租户隔离测试包含在该次 Go 测试中。该 CI run 的清理步骤正在结束，核心验证步骤均 success。


### Telegram 群成员增量采集与网站列表（第十一批）
- **数据库**：新增迁移 `000005_telegram_group_members`，在 `internal/database/migrations/` 与根目录 `migrations/` 同步提供 up/down SQL。成员按群组和 Telegram 用户唯一，保存用户资料、成员状态、管理员权限快照、加入时间与最近活动时间。
- **Telegram 运行时**：`allowed_updates` 增加 `chat_member`；解析成员身份变化，并处理消息中的 `new_chat_members` 与 `left_chat_member`。成员写入使用群组/用户唯一键 upsert，重复更新不会产生重复成员记录；只有更新持久化成功后才推进 Bot offset。
- **服务/API**：新增 `ListGroupMembers` 与 `GET /api/v1/groups/{groupID}/members?limit=100`，limit 范围 1–200；查询同时按当前租户和群组 ID 过滤。
- **网站**：新增 `src/pages/GroupMembersPage.tsx`，使用现有 shadcn/ui 官方 Badge、Button、Card、Table 组件；群组列表增加“成员”入口。
- **测试**：新增 chat_member JSON 解析单测；PostgreSQL 集成测试覆盖成员查询与跨租户隔离；HTTP 测试覆盖成员 API 在服务未配置时返回 503。
- **数据边界**：成员表是增量采集，不会自动回填 Bot 加入前的完整成员名单。Telegram 的 `chat_member` 更新要求 Bot 在群内具备管理员身份，且显式订阅该 update 类型。尚未实现全量成员同步、成员处罚、权限主动刷新和真实 Telegram 群组联调。
- **验证状态**：代码、迁移、前端和文档已提交 `main`；本批 GitHub Actions CI 尚待运行结果，当前不标记为测试通过。


- **CI 验证补充**：GitHub Actions run `45713735190`（代码提交 `decbbbd7a6566283eb274683fe87a98f6eae14eb`）两个 Job 均为 success：Go 格式检查、Go/PostgreSQL 测试、`go vet ./...` 全部通过；前端依赖安装与生产构建通过。成员列表 API 与前端页面已通过云端构建验证。真实 Telegram 群成员更新仍需测试群组联调。


### Bot 群组权限实时检测与成员处罚（第十二批）
- **服务端权限检测**：新增 `GET /api/v1/groups/{groupID}/permissions`，使用 Bot 自身 Telegram 用户 ID 调用 `getChatMember` 获取实时身份与权限；群主身份按 Telegram 语义视为拥有管理权限。
- **成员处罚**：新增 `POST /api/v1/groups/{groupID}/members/{userID}/action`，支持封禁、解封、限时禁言和解除禁言。执行前验证当前租户的群组/Bot 关系、Bot active 状态、Bot 管理员身份与 `can_restrict_members`；再通过 Telegram 查询目标成员，禁止处罚管理员/群主，并校验动作与目标当前状态是否匹配。
- **状态与审计**：Telegram 操作成功后，在 PostgreSQL 事务中更新本地成员状态并写入 `audit_logs`。Telegram API 与本地数据库不是分布式事务；若 Telegram 已执行而本地事务失败，接口会明确返回“Telegram action succeeded”错误，要求人工核对。
- **网站**：群成员页展示实时 Bot 权限，增加禁言、解除禁言、封禁、解封操作；危险操作要求确认，禁言时要求输入 1–525600 分钟。
- **测试**：增加非法成员操作参数测试；HTTP 测试覆盖权限与处罚接口在服务未配置时返回 503。
- **文档**：更新 API、开发计划和本地测试说明。
- **验证状态**：代码与文档已提交 main；GitHub Actions 最新 CI 正在运行，尚未取得最终结果。真实 Bot 群组权限和处罚动作必须在测试群组联调后才能验收。


- **权限恢复安全修正**：解除禁言不再将所有 Telegram 发送权限无条件设置为 true；现在先调用 `getChat` 读取群组默认权限，再按群默认权限解除限制。若 Telegram 未返回默认权限则拒绝操作，避免意外扩大成员权限。


- **CI 最终验证**：GitHub Actions run `45717562198`（代码提交 `6cd32d127f78b1c70837612d7ccbdbbcf9f82b56`）Go 格式检查、`go test -v ./...`（含 PostgreSQL 集成测试）、`go vet ./...` 和前端依赖安装/生产构建均 success。此前 CI 暴露出 Bot 服务文件重复声明问题，已删除重复代码块并由本次完整 CI 验证修复。


### 群组规则配置底座（第十三批，配置管理阶段）
- **数据库**：新增迁移 `000006_telegram_group_rules`，在 internal embed migrations 与根目录部署迁移目录同步 up/down SQL；建立租户/群组归属、规则类型、名称、启停、优先级、JSONB 配置和索引。
- **后端**：新增 `internal/bots/group_rules.go`，实现按租户和群组范围查询、创建、更新、删除；写入时通过当前租户查询群组，不接受客户端 tenant_id；同群同类型规则名称唯一。
- **配置校验**：支持 keyword_filter、sensitive_word、anti_spam、anti_ad、welcome、leave_notice；按类型严格拒绝未知 JSON 字段和不合法参数。过滤动作暂时只接受 delete，避免将未经实现的处罚动作写入配置。
- **API**：新增 GET/POST/PUT/DELETE `/api/v1/groups/{groupID}/rules` 及规则 ID 子路径；所有接口要求 Bearer 登录并以当前用户 tenant_id 为边界。
- **前端**：新增 `src/pages/GroupRulesPage.tsx`，使用仓库中官方 shadcn/ui Input、Label、Select、Switch、Textarea、Card、Badge、Button；群组列表增加规则入口。
- **测试**：新增规则类型与配置字段校验单元测试；HTTP handler 增加规则服务未配置时的 503 路由覆盖。尚待 CI 执行。
- **明确边界**：本批仅完成规则配置持久化和管理界面；消息事件接入执行引擎、Telegram 删除消息、欢迎/离群消息发送、命中日志及规则操作审计仍未实现。页面已明确提示保存不代表自动执行。

- **CI 验证补充**：GitHub Actions run `45722797061`（commit `2bc5405c21939da5c2aa088f41e48477ebcb5e5b`）Go 格式检查、`go test ./...`、`go vet ./...` 与前端依赖安装/生产构建均通过。该结果验证当前配置管理代码可编译和既有测试通过，不代表真实 Telegram 自动规则执行已验收。


### Telegram 群组规则自动执行（第十四批，第一阶段）
- **执行接入**：长轮询收到消息后，先幂等写入 `telegram_message_events`；仅当本次确实插入新消息时调用规则引擎。重复 update 不会重复触发规则；规则执行失败会记录结果，不会让已保存消息的 update offset 卡住。
- **文本过滤**：实现关键词过滤、敏感词过滤和反广告的大小写不敏感子串匹配。匹配后只尝试删除当前消息；执行前调用 Telegram `getChatMember` 实时检查 Bot 是否为群管理员且具备 `can_delete_messages`。权限不足或 Telegram API 失败会记录失败/跳过原因，不会伪造成功。
- **反刷屏**：按同一租户、群组、发送者在配置时间窗口内的消息记录数量触发；达到阈值后删除当前触发消息。当前不批量删除历史消息，也不自动禁言。
- **成员通知**：处理消息中的 `new_chat_members` 发送欢迎消息，处理 `left_chat_member` 发送离群通知；模板支持 `{user}`、`{username}`、`{group}`，并限制 Telegram 文本长度。
- **审计幂等**：新增迁移 `000007_telegram_group_rule_events`，记录每条规则动作的 succeeded/skipped/failed 状态；多成员加入同一条消息时按目标用户区分幂等键。
- **安全边界**：Bot 消息不参与文本过滤；规则配置仍严格由服务端校验；规则 API 仍按当前租户和群组过滤。当前反广告是简单子串匹配，不是正则/语义识别；未实现自动禁言/封禁、chat_member 驱动通知、命中记录查询 API/UI、重试队列及真实 Telegram 联调。
- **测试**：新增通知模板占位符与长度上限单测；等待最新 CI 对 Go 编译/测试/Vet 与前端构建验证。

- **CI 验证与修复**：首轮 CI run `45723781019` 发现规则引擎文件存在未使用的 `time` import，已删除并提交修复。最终代码提交 `8ef507a5529570a0a656f56b804ea1b6f6d42e19` 的 GitHub Actions run `45723931636` 两个 Job 均 success：Go 格式检查、PostgreSQL 集成测试、`go vet ./...`、前端依赖安装和生产构建全部通过。CI 证明编译与自动化测试通过，不替代真实 Telegram 测试群联调。


### 群组规则执行记录查询（第十四批补充）
- **后端**：新增 `ListGroupRuleEvents`，仅按当前租户与群组查询最近执行记录；新增 `GET /api/v1/groups/{groupID}/rule-events?limit=50`，limit 范围 1–100。
- **前端**：新增 `src/pages/GroupRuleEventsPage.tsx`，使用官方 shadcn/ui Table、Badge、Card、Button 展示执行时间、动作、状态、目标用户/消息和执行详情；规则配置页增加“执行记录”入口，并修正旧版“规则尚未自动执行”提示。
- **测试**：扩展 PostgreSQL 集成测试，覆盖规则创建/列表、规则事件查询和跨租户不可见；HTTP 测试覆盖执行记录路由在服务未配置时返回 503。
- **文档**：更新 API 与开发计划，明确执行记录 API、当前自动规则能力和未实现边界。

- **集成测试修正与复验**：规则集成测试最初插入在群组创建之前，CI run `45724386591` 因测试代码引用尚未声明的 `groupID` 编译失败；已将断言移动到群组创建之后，提交 `df312bb0f58b0acdc97aa4d50cde9f344686d5fb` 正在执行完整 CI。API 文档换行格式已修正。

- **CI 最终验收**：修正后的集成测试提交 `df312bb0f58b0acdc97aa4d50cde9f344686d5fb` 对应 GitHub Actions run `45724599031` Go/PostgreSQL 测试、`go vet` 和前端生产构建均通过；最后修正规则页重复导航后的提交 `ae3f3520bec219b180904ce0b81168e613d2c0aa` 对应 run `45724721446` 两个 Job 也全部 success。规则配置/执行记录查询的跨租户集成测试现已通过云端 PostgreSQL 验证。


### Telegram 入群数学题验证（第十五批，基础验证阶段）
- **规则类型**：新增 `join_verification`，服务端严格校验 `challenge_type=math`、超时 30–600 秒、失败动作仅允许 `restrict` 或 `kick`。
- **数据模型**：新增迁移 `000008_join_verification`，建立挑战表，按群组/成员唯一保存待验证挑战、答案、状态、超时时间和失败动作；同步 internal 与根目录迁移文件。
- **入群处理**：对新加入的非 Bot 成员，仅在 Telegram supergroup 中执行；实时检查 Bot 是否为群主/管理员并具备 `can_restrict_members`，随后临时关闭成员发送权限，发送随机加法题和三个 Inline Keyboard 选项。
- **答题校验**：订阅 `callback_query` 更新，验证 challenge、租户、Bot、群组、成员 Telegram ID 和有效期；其他成员无法代答。答对后读取 Telegram 群默认权限并原样恢复，不授予超出群默认设置的权限；成功后清理验证按钮。
- **超时处理**：worker 每轮长轮询前扫描到期挑战；`restrict` 保持限制，`kick` 通过封禁后立即解封实现移出群组。失败不会假报成功。
- **安全边界**：当前为数学题基础验证，不是 CAP.js；仅适用于 supergroup。若 Bot 无限制成员权限、群组默认权限不可读取或 Telegram API 调用失败，会跳过/记录错误或保留限制，需管理员检查。尚未实现频道关注验证、CAP.js 服务端验证、chat_member 更新驱动挑战和可配置复杂题型。
- **测试**：新增规则配置合法性测试和数学题选项边界/去重单测；等待云端 CI 完整验证。

- **CI 验证结果**：最新代码提交 `dff3d8f5f623bb92f558a05f142cf994e27747e1` 对应 GitHub Actions run `45725882924` 两个 Job 均 success：Go 格式检查、PostgreSQL 集成测试、`go vet ./...` 和前端生产构建通过。前端 Job 已通过；Go 测试包含新增规则配置与数学题选项边界测试。该结果不替代真实 Telegram supergroup 联调。
- **测试文档**：`docs/LOCAL-TESTING.md` 增加入群验证人工验收步骤，覆盖 Bot 权限、本人答题、超时踢出/持续限制及数据库状态检查。


### 入群验证增加频道关注校验（第十五批第二阶段）
- **配置扩展**：`join_verification` 增加可选 `required_channels`，最多 10 个，接受公开频道用户名 `@name` 或 Telegram 数字频道 ID（`-100...`）；拒绝非法格式和重复频道。
- **数据迁移**：新增 `000009_join_verification_channels`，挑战表保存本次验证所需频道列表，internal 与根目录 up/down SQL 同步。
- **运行时校验**：数学题答对后，逐个调用 Telegram `getChatMember` 检查该用户是否已加入全部指定频道；任一频道未加入时不解除群组发言限制，并提示用户先加入频道。Telegram 无法查询频道成员状态时失败关闭。
- **安全限制**：Bot 必须加入目标频道，建议授予频道管理员身份，以确保 Telegram 允许查询其他用户的成员状态。频道用户名和数字 ID 均可配置；私密邀请链接本身不能作为频道 ID 使用。
- **测试**：增加频道引用格式校验、频道成员状态识别、Telegram API 权限失败关闭和提示文本单测；通过可注入 Telegram API base URL 的 httptest server 模拟 Telegram 响应。
- **文档**：更新 API、开发计划和人工验收步骤。CAP.js 官方服务端验证仍未接入。

- **频道验证测试修复与 CI**：首轮频道验证 CI run `45726378568` 发现运行时测试错误引用了 bots 包内未导出的配置校验 helper，已删除跨包引用；该校验本身由 `internal/bots/group_rules_test.go` 覆盖。修复提交 `38d8fba244e4e4dd6d0a66dad9aa88aa0e096633` 对应 run `45726537680` 的 Go 格式检查、Go/PostgreSQL 测试、`go vet ./...` 和前端生产构建均通过。


### 官方 CAP.js 接入网站认证（第十五批第三阶段）
- **官方方案核实**：采用 Cap Standalone 官方服务端与 `/siteverify`，而不是官方文档已标记为被取代的旧 `@cap.js/server` 集成方式；前端固定使用官方 `cap-widget@0.1.58`。
- **后端配置**：新增 `CAP_BASE_URL`、`CAP_SITE_KEY`、`CAP_SECRET`；三项必须同时配置。远程站点只允许 HTTPS，HTTP 仅允许 loopback 本地开发地址。
- **服务端验证**：新增 CAP verifier，限制请求超时与响应体大小；按官方 JSON 协议提交站点 secret 和浏览器 response token；仅接受 2xx 且 `success=true`。空 token、无效/重放 token、服务异常均失败关闭；secret 不会出现在公开配置接口。
- **API**：新增 `GET /api/v1/security/captcha`，只返回 enabled 和公开 endpoint；网站注册、登录请求支持 `captcha_token`，启用 CAP 后强制服务端校验。
- **前端**：注册页、登录页通过独立 CaptchaField 加载官方 Web Component；失败后重新挂载组件获取新 token。未配置 CAP 时不显示组件，开发环境原有流程兼容。
- **测试**：新增官方 siteverify 请求体/路径、成功/失败/空 token、配置校验、loopback HTTP 限制及公开配置不泄露 secret 测试；更新配置、API 和人工验收文档。
- **限制**：本次接入覆盖网站注册/登录；Telegram 群聊不能直接渲染浏览器 CAP 组件，群内继续采用数学题 Inline Keyboard 与频道成员校验。真实 CAP 端到端仍需部署 Cap Standalone、创建站点密钥并进行浏览器联调。


- **CAP 资源部署补充**：新增可选 `CAP_WIDGET_URL`，支持将官方 `widget.js` 指向自托管/镜像地址；默认使用固定版本 CDN。公开 CAPTCHA 配置只返回该脚本地址，不返回任何 secret。增加 widget URL HTTPS/loopback 校验与配置测试，并同步本地联调文档。若使用 Cap Standalone assets server，仍需按官方说明固定 widget/WASM 版本并保证 WASM 文件浏览器可访问。

- **CAP 阶段 CI 验证**：代码提交 `2a984250097b952837f2cbc9ee49f74a1de6e073` 对应 GitHub Actions run `45728415621` 两个 Job 均 success：Go 格式检查、完整 Go/PostgreSQL 测试、`go vet ./...` 与前端依赖安装和生产构建全部通过。此结果覆盖 CAP verifier、配置与测试、前端 CAP 组件编译；不替代实际 Cap Standalone 公网浏览器联调。


## 2026-11-26（全仓库审查校准）

- **审查范围**：复核 main 分支文件树、REQUIREMENTS、DEVELOPMENT-PLAN、DEVELOPMENT-LOG、API、LOCAL-TESTING、README、HOMEPAGE-CONTENT、CI 工作流及最新 Actions 运行。
- **审查基线**：提交 `453151d85b8e8478cb1042b9bd7fde56841390e3`；对应 CI run `45728546740` 为 success，覆盖 Go/PostgreSQL 测试、go vet 和前端生产构建。
- **已确认代码范围**：网站邮箱认证、Bot Token 加密管理、Telegram 长轮询/群组发现、消息日志、成员增量采集与手动处罚、基础规则配置/执行/记录、数学题及频道关注入群验证、网站 CAP siteverify。
- **未完成项**：可信 Telegram ID 绑定与注册管理 Bot、refresh token/密码找回/邮箱验证/登录限流/完整角色授权、完整群管功能矩阵与自动处罚队列、Telegram 管理端双端等价、支付授权、赞助商、AI 审核、CAP 公网验收、真实 Telegram 端到端及生产部署。
- **文档同步**：校准开发计划日期与后续顺序；修正 README 和首页介绍中过期的功能状态，避免把未开发功能宣传为已交付。
- **验证**：本次仅修改文档，未运行新代码测试；本次提交后的 CI 结果为准。
- **下一步**：从账号体系补齐开始，优先设计可信 Telegram ID 绑定和会话安全；涉及外部邮件/注册 Bot 凭据的流程在配置提供前不宣称完成。


### Telegram 身份自动绑定（第十六批第一阶段，首批代码）
- **数据库**：新增 migration 000010，建立 `telegram_identity_link_challenges`，并给 users 增加 Telegram username/first_name/last_name 字段；internal 与根目录 up/down SQL 同步。
- **绑定服务**：新增 `internal/auth/telegram_link.go`。网站登录用户可创建 5 分钟有效的一次性挑战；数据库仅存 SHA-256 摘要，新挑战会使旧挑战失效。绑定事务锁定挑战、校验未过期/未消费、写入 Telegram ID 和资料、消费挑战并写入审计日志。
- **身份来源**：新增独立 `internal/telegramauthbot` 长轮询 worker。只处理 Telegram 私聊消息中的 `/start <challenge>`，要求 `message.from.id == message.chat.id` 且发送者不是 Bot；ID、用户名和姓名字段均来自 Telegram 官方 Update。该 Bot 与租户自行添加的群管 Bot 分离。
- **API**：新增 `GET /api/v1/auth/telegram`、`POST /api/v1/auth/telegram/link`、`DELETE /api/v1/auth/telegram`。网站 API 不接收 Telegram ID。解除绑定会清空资料、使待处理挑战失效并写审计日志。
- **前端**：账号工作台新增 Telegram 身份绑定卡片，显示绑定状态，点击后跳转 Telegram 官方 Bot；已绑定时可解除绑定。继续使用现有 shadcn/ui 官方 Button/Card 组件。
- **配置**：新增 `TELEGRAM_AUTH_BOT_TOKEN` 与 `TELEGRAM_AUTH_BOT_USERNAME`，必须同时设置；Token 仅在后端使用。平台 Bot 必须独立于租户群管 Bot，采用 getUpdates 时不能同时配置 webhook。
- **测试**：增加绑定/解绑/重复绑定/挑战重放 PostgreSQL 集成测试和 Telegram Bot 配置测试；HTTP 测试增加身份 API 在无数据库时的 503 检查。
- **文档**：更新 README、API、LOCAL-TESTING 和 DEVELOPMENT-PLAN，明确真实 Telegram Bot Token 尚未提供，代码/CI 验证不等于 Telegram 端到端验收。
- **验证状态**：源码与文档已提交 main；等待 GitHub Actions Go/PostgreSQL 测试、go vet 与前端构建结果。真实 Bot 联调待平台身份 Bot Token/username 配置后进行。

- **CI 修复与最终验证**：首次 CI 发现前端 Dashboard 状态变量作用域错误，已修复；PostgreSQL 集成测试先后发现审计 JSON 参数缺少 bigint 类型以及解绑审计复用 UUID/text 参数的问题，均已修复。最终代码提交 `dfbdbb1d2bebc304573f1f90781ed3f9b8381eea` 对应 GitHub Actions run `45731398955`：Go/PostgreSQL 全量测试、`go vet ./...` 和前端 TypeScript 生产构建全部 success。CI 验证覆盖绑定成功、挑战重放、Telegram ID 重复绑定拒绝、解除绑定和配置读取；未覆盖真实 Telegram Bot 网络联调。

- **额外安全加固与 CI**：身份 Bot HTTP 请求错误不再记录原始网络错误，避免错误文本带出含 Token 的 Telegram API URL；平台 Bot username 增加字符集、长度和 `bot` 后缀校验。提交 `cfa5760df716182c8f7e7e2e1886a83693dedd64` 对应 run `45731715648` 最终 success，Go/PostgreSQL 测试、`go vet ./...`、前端生产构建均通过。当前仅缺平台身份 Bot 的真实凭据和 Telegram 网络端到端验收。


### Refresh Token 会话续期（第十六批第二阶段，首批实现）
- **数据库**：新增 migration 000011，`user_sessions` 增加 `family_id`，新增 `user_refresh_tokens` 保存 refresh token SHA-256 摘要、family、过期/撤销时间、替换摘要、User-Agent 和 IP。`internal/database/migrations` 与根目录 `migrations` 同步 up/down 文件。
- **后端**：注册/登录签发 15 分钟 access token 与 30 天 refresh token；新增 `Service.Refresh` 和 `POST /api/v1/auth/refresh`，每次续期轮换两类令牌。已轮换令牌被重放时撤销整个 family 的刷新令牌与访问会话；退出登录撤销当前 family。账号/租户停用时拒绝续期并撤销该 family。
- **前端**：在 sessionStorage 中保存 access/refresh token；受保护请求遇到 401 自动调用 refresh 并重试；通过 single-flight 合并并发续期请求，避免同一旧 refresh token 被同时消费。登录/注册页面同步保存两类令牌。
- **测试**：扩展 PostgreSQL 集成测试，覆盖 refresh 轮换、旧 refresh token 重放触发 family 撤销、退出登录后 refresh token 不可用；HTTP 测试覆盖无数据库时 refresh endpoint 返回 503。
- **文档**：更新 API、LOCAL-TESTING、README 与 DEVELOPMENT-PLAN。
- **验证状态**：代码已提交 main，等待 GitHub Actions 全量 Go/PostgreSQL 测试、go vet 与前端构建。

- **CI 修复与验证**：首次 CI 发现 refresh 审计日志把 UUID actor 参数与文本 resource_id 复用，PostgreSQL 无法推断参数类型；已拆分为独立参数。最终代码提交 `bcf9771347f266ec8dc30c97534a344e4f318d20` 对应 Actions run `45733654763` 全部 success：PostgreSQL 集成测试、`go vet ./...`、前端 TypeScript 生产构建均通过。集成测试覆盖 refresh 轮换、旧 refresh token 重放触发整个 family 撤销、退出登录后 refresh token 失效。
- **当前验收边界**：CI 已通过；尚未在真实部署环境等待 15 分钟验证浏览器自动续期。现有旧版本登录会话没有 refresh family，用户升级后首次重新登录即可获得新双令牌。


### 账号密码修改与全设备会话撤销（第十六批第三阶段，首批实现）
- **服务端**：新增 `POST /api/v1/auth/password/change`，必须携带当前有效 Bearer 会话；先验证当前密码，再以 bcrypt 更新密码哈希。密码长度沿用注册规则 10–72 字节，并拒绝与当前密码相同的值。
- **会话安全**：密码变更在单个 PostgreSQL 事务中撤销该用户全部既有 access sessions 和 refresh tokens，再为当前设备创建新的 refresh family/access session；当前设备保持登录，其他设备必须重新登录。旧 access/refresh token 均失效。
- **审计**：写入 `auth.password_change` 审计记录；审计写入失败时密码更新和会话撤销整体回滚。
- **前端**：账号工作台新增修改密码卡片，包含当前密码、新密码和确认密码；成功后立即替换浏览器内的双令牌，不再要求当前设备重新登录。界面继续使用仓库内 shadcn/ui 官方 Input、Label、Button、Card。
- **测试**：新增 PostgreSQL 集成测试，覆盖原密码错误、弱密码、重复旧密码、成功修改、旧密码无法登录、其他设备 access session 失效、旧 refresh token 失效和新密码登录；HTTP 路由增加未配置数据库时 503 覆盖。
- **文档**：同步 API、LOCAL-TESTING、DEVELOPMENT-PLAN。
- **验证状态**：代码已提交 main；等待 GitHub Actions Go/PostgreSQL 测试、go vet、前端 TypeScript 生产构建完成后再确认验收。


### 登录失败频率限制（第十六批第四阶段，首批实现）
- **数据库**：新增 migration `000012_login_rate_limit`，建立登录失败记录表及邮箱/IP/时间索引；内部 embed migrations 与根目录部署迁移 up/down 文件同步。
- **限流策略**：同一规范化邮箱或同一来源 IP，在滚动 15 分钟内累计 5 次密码验证失败后，后续登录返回 HTTP 429；响应包含 `Retry-After: 900`。失败计数按两个维度分别判断，任一维度达到阈值即限制。
- **隐私与并发**：邮箱和 IP 先加作用域前缀再计算 SHA-256，数据库只保存摘要；同一邮箱/IP 的并发登录通过 PostgreSQL advisory transaction lock 串行检查和记数，避免并发请求越过阈值。失败记录超过 24 小时后分批清理。
- **实现边界**：限流直接使用 PostgreSQL，不假设 Redis 已接入；来源 IP 取 HTTP RemoteAddr，不信任客户端提供的 X-Forwarded-For。若部署在反向代理后，需在未来单独实现可信代理配置后再启用代理 IP 解析。
- **测试**：新增 PostgreSQL 集成测试覆盖邮箱维度、IP 维度、新邮箱/IP 放行；HTTP 单测覆盖 429 和 Retry-After 响应。等待 CI 验证。

- **密码修改 CI 验证**：代码提交 `6d6a8db68d262d15ba089a5976d2788c558af562` 对应 Actions run `45737793039` 的 Go/PostgreSQL 测试、gofmt、go vet 和前端生产构建均通过。
- **登录限流 CI 修复与最终验证**：首轮 CI 曾因错误变量定义提交顺序导致短暂编译失败，已补齐 `ErrLoginRateLimited` 后重新运行。代码提交 `ec9c08c9ffd4907157df63ddfdec87bef6f946d8` 对应 run `45738314492` 的 Go 格式检查、Go/PostgreSQL 集成测试、go vet 和前端 TypeScript 生产构建全部 success；邮箱/IP 双维度限流集成测试通过。


### 账号安全审计记录查询（第十六批第五阶段，首批实现）
- **服务端**：新增 `GET /api/v1/auth/security-events?limit=20`，按当前认证用户的 actor_user_id 查询 `audit_logs` 中的 `auth.*` 事件，默认 20 条、最大 100 条；不会暴露同租户其他用户的登录历史。
- **审计补齐**：退出登录事务现在同步写入 `auth.logout`，记录来源 IP；若审计写入失败，撤销操作事务回滚，避免成功退出却没有审计记录。
- **前端**：账号工作台增加“账号安全记录”列表，显示注册、登录、退出、续期、密码修改、Telegram 绑定/解绑事件及来源 IP/时间。UI 继续使用现有 shadcn/ui 官方 Card、Badge。
- **测试**：PostgreSQL 集成测试验证登录、续期、退出、密码修改、Telegram 绑定/解绑记录可查询；HTTP 测试覆盖无数据库时安全记录路由返回 503。
- **文档**：更新 API、LOCAL-TESTING 和 DEVELOPMENT-PLAN。
- **验证状态**：代码已提交 main；等待 GitHub Actions Go/PostgreSQL 测试、go vet 和前端构建结果。

- **CI 验证**：账号安全审计代码提交 `4173d49d1fda44c3291773d155ca012dbab7f0e5` 对应 Actions run `45738879169` 的 Go 格式检查、Go/PostgreSQL 集成测试、go vet 和前端 TypeScript 生产构建均 success。覆盖认证事件查询、退出审计、当前用户事件隔离和前端安全记录展示。


### 登录设备管理与单设备撤销（第十六批第六阶段，首批实现）
- **服务端**：新增 `GET /api/v1/auth/sessions?limit=20`，列出当前用户最近登录会话，返回设备 User-Agent、来源 IP、创建/过期/撤销时间和当前设备标记；不返回任何 Token 或 Token 哈希。
- **撤销 API**：新增 `DELETE /api/v1/auth/sessions/{sessionID}`，只允许撤销当前账号的其他有效设备；拒绝撤销当前请求所用会话，撤销目标 access session 并同步撤销其 refresh-token family。跨用户会话 ID 不可操作。
- **审计**：成功撤销其他设备时写入 `auth.session_revoke`，并在账号安全记录中显示。
- **前端**：工作台增加“登录设备管理”，可查看设备/IP/登录时间和当前/已退出状态，并单独退出其他设备。
- **测试**：PostgreSQL 集成测试覆盖当前设备保护、其他设备 access/refresh token 失效及审计事件；HTTP 测试覆盖 sessions API 在数据库未配置时返回 503。
- **文档**：更新 API、LOCAL-TESTING 和 DEVELOPMENT-PLAN。
- **验证状态**：代码已提交 main；等待 GitHub Actions Go/PostgreSQL 测试、go vet 和前端生产构建结果。

- **CI 验证**：登录设备管理代码提交 `71e35019c1079307388466be8ff54255329a109c` 对应 Actions run `45739299233` 的 Go 格式检查、Go/PostgreSQL 集成测试、go vet 和前端生产构建均 success。集成测试覆盖当前设备不能撤销、其他设备 access/refresh token 撤销和审计事件。


### Docker 零配置一键部署（第十七批）
- **预构建镜像**：新增 `Dockerfile.api`（Go 静态二进制多阶段构建）和 `Dockerfile.web`（Vite 构建后由 Nginx 提供 SPA 与 API 反向代理）；GitHub Actions 自动发布 GHCR amd64/arm64 镜像：`ghcr.io/seanldnet/tg-group-api:latest`、`ghcr.io/seanldnet/tg-group-web:latest`。部署服务器只执行 pull，不执行编译。
- **一键脚本**：`deploy/install.sh` 支持 Ubuntu/Debian apt 和 RHEL 系 dnf；缺少 Docker 时自动安装；下载 Compose 部署定义；自动生成随机 PostgreSQL 密码与 Bot Token 加密密钥（权限 0600），无需用户填写或编辑配置。
- **服务编排**：PostgreSQL 仅加入 Docker 内部网络，不暴露宿主机端口；API 仅内网可访问；Nginx Web 映射 TCP 18680，反代 `/api` 到 API。数据库使用持久化 volume。
- **管理员**：数据库迁移完成后幂等创建 `admin@tg-group.local` 演示管理员，登录用户名支持 `admin`，密码 `123456`；仅当账号不存在时创建，不会重置已有管理员密码。登录页输入框已支持用户名。
- **测试**：新增 PostgreSQL 集成测试覆盖首次初始化、重复启动不重复创建和 admin 别名登录。
- **使用方式**：`curl -fsSL https://raw.githubusercontent.com/seanldnet/Tg-group/main/deploy/install.sh | sudo bash`，访问 `http://服务器IP:18680`。
- **安全边界**：默认账号密码仅供测试，公网使用前必须修改密码；自动生成的密钥保存在安装目录，用户无需编辑。GHCR 包必须允许匿名拉取。
- **CI 状态**：正在验证镜像构建、推送、公开可拉取及 Go/前端 CI；若 GHCR 包可见性无法由工作流令牌自动调整，需要在仓库所有者侧完成一次 Package visibility 设置，之后服务器部署完全无需登录镜像仓库。

- **最终验证结果**：提交 `95f8c6efe12c8e47bacdb52ffab18b475ff7a858` 的 CI run `45741570144` 全部通过（Go/PostgreSQL 集成测试、go vet、前端构建）；Docker 发布 run `45741570062` 成功，已推送 API/Web 两个 amd64+arm64 镜像。Docker build 在 GitHub Actions 云端完成，目标服务器无需编译。
- **GHCR 匿名拉取限制**：发布工作流可成功推送镜像，但使用仓库 GITHUB_TOKEN 修改个人包可见性返回 404（已在 Actions 日志确认）；需包所有者在 GitHub Packages 对两个包各执行一次 Public 设置。完成后服务器可匿名 docker pull，不需要服务器侧 registry login。README 已补充两个包设置链接。
- **AES 密钥修正**：安装脚本生成的 `BOT_TOKEN_ENCRYPTION_KEY` 改为 Base64 编码的 32 字节随机密钥，匹配服务端 AES-256 key 解码要求；数据库密码仍使用随机 hex。

### Docker 部署脚本热修复
- 用户首次执行时，安装目录中已有旧版 `compose.yaml`，Docker Compose 自动优先选择该文件，导致拉取旧的 `ghcr.io/seanldnet/tg-group:latest` 并返回 denied。
- 安装脚本现在明确指定下载的 `/opt/tg-group/docker-compose.yml`，并显式传入自动生成的 env 文件执行 `pull/up`；日志与停止命令同样固定使用该 compose 文件，避免目录内其他 Compose 文件抢占。
- 修复不删除数据库卷、不要求编辑配置，用户重新执行原一键命令即可。


### Docker 安装器环境变量自修复（部署反馈修复）
- **问题**：用户服务器已有旧版/中断安装生成的 `.env` 文件，但其中缺少或为空的 `TG_GROUP_DB_PASSWORD`、`TG_GROUP_BOT_ENCRYPTION_KEY`。安装器仅在文件不存在时生成密钥，导致 Compose 插值失败。
- **修复**：安装器现在逐项读取已有密钥，保留非空值，仅为缺失/空值生成随机值；原子写回并保留其他环境变量，继续设置 `.env` 权限为 0600。
- **安全/数据边界**：不会主动轮换已存在的数据库密码或加密密钥，不删除 PostgreSQL volume；新生成的密钥持久保存在安装目录。
- **验证边界**：已检查 GitHub main 上脚本内容；尚未在用户服务器执行，需由用户重新运行一键安装并回传结果验证完整部署。


### PostgreSQL 持久卷密码同步修复（部署联调）
- **问题证据**：用户首次部署时 PostgreSQL 容器健康，但 API 日志持续出现 SQLSTATE 28P01（tg_group 密码认证失败）。此前中断/旧安装留下的 PostgreSQL 数据卷保留了初始化密码，而安装器补齐的 .env 密码与数据库角色密码不一致。
- **修复**：安装器拉取镜像后先单独启动 PostgreSQL并等待就绪，再通过容器内配置的 tg_group 初始化超级用户连接执行 `ALTER ROLE tg_group WITH PASSWORD`，将角色密码同步到当前持久化 .env；随后才启动 API 和 Web。
- **数据边界**：只修改数据库登录角色密码，不删除或重建数据库、不清空 volume、不改业务数据；.env 中已存在的密码仍被保留。
- **验证边界**：安装器修复已提交；尚待用户服务器重新执行脚本，确认 API 健康检查通过及 Web 正常启动。


### 可选 CAP 组件默认隐藏修复（2026-11-26）
- **问题**：CAP 未配置或公开配置接口请求失败（例如 API 反向代理返回 502）时，前端 CaptchaField 将配置加载异常显示为“验证组件暂时不可用”，让未启用的可选功能出现在登录/注册页。
- **修复**：配置读取失败时按可选组件处理，直接隐藏 CAP 区域；只有服务端明确返回 `enabled: true` 且资源加载成功时才渲染官方 cap-widget。CAP 已启用时，后端注册/登录仍独立执行 token 校验，不因前端隐藏而关闭服务端验证。
- **提交**：`3aa2dbb20508976e246f30a31900aaa35829db27`。
- **验证边界**：代码已提交 main；等待 GitHub Actions 前端生产构建。用户现场的 502 仍需结合 API 容器健康状态排查，隐藏组件不等于修复 API 502。
