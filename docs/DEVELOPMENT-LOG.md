# TG-Group 开发日志

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
