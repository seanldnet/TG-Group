# 本地开发与测试

## 环境要求

- Go 1.23+
- Node.js 22+
- pnpm 10+
- Docker Engine 与 Docker Compose

## 启动依赖服务

在仓库根目录执行：

```bash
docker compose -f compose.dev.yml up -d
```

默认启动 PostgreSQL 16（宿主机 5432）和 Redis 7（宿主机 6379），数据保存在 Docker named volumes。

开发环境默认数据库连接：

```text
postgres://tg_group:tg_group_dev@127.0.0.1:5432/tg_group?sslmode=disable
```

Redis 当前仅提供开发基础设施，Go 服务尚未接入 Redis 功能。

## 启动 Go API

```bash
go mod tidy
BOT_TOKEN_ENCRYPTION_KEY="<填入首次生成并持久保存的 Base64 密钥>" DATABASE_URL="postgres://tg_group:tg_group_dev@127.0.0.1:5432/tg_group?sslmode=disable" go run ./cmd/api
```

首次启动会自动执行 PostgreSQL 迁移。Bot 加密密钥只生成一次并妥善保存，服务重启必须继续使用同一密钥；不要每次启动重新生成。可用 `openssl rand -base64 32` 生成。API 默认监听 `http://127.0.0.1:8080`。

## 启动前端

另开终端：

```bash
pnpm install
TG_GROUP_API_PROXY="http://127.0.0.1:8080" pnpm dev
```

打开 Vite 输出的本地地址。开发服务器会将 `/api/*` 代理到 Go API。

## 当前可测试流程

1. 首页：查看 TG-Group 产品介绍。
2. 注册：`/register`，提交称呼、邮箱和至少 10 个字符的密码。
3. 注册成功后自动进入 `/dashboard`。
4. 工作台调用 `GET /api/v1/auth/me` 展示当前账号。
5. 退出登录后会话撤销，可重新访问 `/login` 登录。\n6. 配置 `BOT_TOKEN_ENCRYPTION_KEY` 后，可使用 Bot 管理 API 验证并登记 Telegram Bot Token。
7. 将 Bot 添加到测试群组，发送一条消息或触发 Bot 成员状态更新；运行中的 API 会通过 getUpdates 长轮询发现群组并写入 `telegram_groups`。
8. 可在 PostgreSQL 中检查发现结果：`SELECT telegram_chat_id,title,chat_type,last_seen_at FROM telegram_groups ORDER BY last_seen_at DESC;`

注意：当前运行时使用 getUpdates 长轮询。若 Bot 已配置 Telegram webhook，getUpdates 会与 webhook 冲突；本版本尚未提供 webhook 自动切换，请先在测试 Bot 上清理 webhook 后再验证长轮询。不要对生产 Bot 执行清理操作，除非已确认其 webhook 迁移计划。

也可以直接检查：

- `GET /healthz`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/logout`\n- `GET /api/v1/bots`
- `GET /api/v1/groups`\n- `POST /api/v1/bots`\n- `PATCH /api/v1/bots/{botID}/status`\n- `PUT /api/v1/bots/{botID}/token`\n- `DELETE /api/v1/bots/{botID}`

API 请求和响应格式见 [API 文档](API.md)。

## 停止依赖服务

```bash
docker compose -f compose.dev.yml down
```

如需同时删除本地数据库和 Redis 持久化数据，请谨慎执行：

```bash
docker compose -f compose.dev.yml down -v
```

`compose.dev.yml` 是本地开发依赖环境，不是最终生产部署方案。生产交付仍要求云端构建并发布镜像，服务器只拉取镜像运行。


## Telegram 群消息日志验证

1. 在网站登录后打开 `/dashboard/groups`。
2. 将已登记并启用的测试 Bot 加入测试群组，发送一条普通文字消息。
3. 在群组列表对应行打开“消息日志”，或请求 `GET /api/v1/groups/{groupID}/events?limit=50`。
4. 日志应显示发送者名称、Telegram 用户 ID、消息类型、文字内容和发送时间。
5. 可在 PostgreSQL 检查：`SELECT telegram_user_id,sender_username,message_type,message_text,telegram_sent_at FROM telegram_message_events ORDER BY received_at DESC LIMIT 20;`

消息日志当前保存文字和 Caption，不保存媒体文件本体；暂未配置自动保留期限。请仅使用测试群组验证，避免把不应进入开发数据库的真实用户消息写入测试环境。

## Telegram 群成员列表验证

1. 将测试 Bot 设置为测试群组管理员；Telegram 只有在 Bot 满足管理员条件时才会投递 `chat_member` 更新。
2. 确保长轮询已运行，且 `allowed_updates` 包含 `chat_member`（当前代码已配置）。
3. 邀请一名测试账号加入群组、退出群组，再调整一名测试管理员的权限。
4. 在网站群组列表点击对应群组的“成员”，或请求 `GET /api/v1/groups/{groupID}/members?limit=200`。
5. 应看到用户 ID、用户名、成员状态、权限快照、加入时间和最近活动时间。成员采集从 Bot 开始收到更新后增量建立，不回填历史成员。
6. 可在 PostgreSQL 检查：`SELECT telegram_user_id,username,status,can_delete_messages,can_restrict_members,joined_at,last_seen_at FROM telegram_group_members ORDER BY last_seen_at DESC LIMIT 50;`

请使用测试群组和测试账号验证。成员管理页会实时读取 Bot 权限；确保 Bot 是管理员并授予“封禁用户/限制成员”权限后，再对普通测试成员执行限时禁言、解除禁言、封禁和解封。页面会要求确认操作，禁言时需要输入分钟数。不要对生产群组或真实用户测试处罚动作。

可直接验证：
- `GET /api/v1/groups/{groupID}/permissions`
- `POST /api/v1/groups/{groupID}/members/{userID}/action`，请求体示例：`{"action":"mute","duration_minutes":30}`

支持 action：`ban`、`unban`、`mute`、`unmute`。执行后检查 Telegram 群内实际状态、本地 `telegram_group_members.status` 和 `audit_logs`。Telegram API 操作与本地数据库并非分布式事务，若数据库在 Telegram 执行成功后发生故障，需要人工核对 Telegram 状态。


## Telegram 入群数学题验证测试

1. 使用独立测试 Bot 和 Telegram supergroup；将 Bot 设为群管理员并授予“封禁用户/限制成员”权限。普通基础群组不支持当前验证流程。
2. 网站登录后进入测试群组的“规则管理”，新增规则类型“入群验证（数学题）”，配置 `challenge_type=math`、`timeout_seconds=120`、`failure_action=kick`。如需频道关注验证，增加 `required_channels`，例如 `["@your_public_channel"]`；Bot 必须加入每个目标频道并建议设为管理员。可将失败动作改为 `restrict`，超时范围 30–600 秒。
3. 使用另一个测试账号加入群组。Bot 应立即限制该成员发言，并发送随机加法题与三个按钮选项。
4. 使用加入者本人点击正确答案；成功后 Bot 应读取群默认权限并恢复该成员权限。其他成员点击按钮应收到“该按钮不属于你的验证”提示。
5. 再用另一个测试账号加入但不答题；到期后若配置 `kick`，Bot 应封禁后立即解封以移出群组；若配置 `restrict`，成员将继续保持受限状态。
6. 通过 PostgreSQL 检查挑战状态：`SELECT telegram_user_id,status,failure_action,expires_at FROM telegram_join_verification_challenges ORDER BY created_at DESC LIMIT 20;`

**重要**：数学题是基础验证，不是 CAP.js，也不能抵御所有自动化账号。若 Bot 权限不足、Telegram 无法返回群默认权限或 API 调用失败，需检查服务日志和群成员状态。仅在测试群组/测试账号执行，不要在生产群组直接试验自动踢人。


## CAP.js 本地/联调配置

项目接入官方 Cap Standalone（服务端）与官方 `cap-widget@0.1.58`。先按 [Cap Standalone 官方快速开始](https://trycap.dev/zh/guide/standalone/) 部署 Cap 和 Valkey/Redis，在 Cap 控制台创建站点密钥，取得 site key 与该站点的 secret key。Cap 的公开 URL 必须能被浏览器访问；正式环境必须使用 HTTPS，并在 Cap 的 CORS 设置中允许 TG-Group 前端域名。

启动 Go API 时设置以下变量（不要提交真实 secret 到仓库）：

```bash
CAP_BASE_URL="https://captcha.example.com" \
CAP_SITE_KEY="<站点公钥>" \
CAP_SECRET="<站点私钥>" \
DATABASE_URL="postgres://tg_group:tg_group_dev@127.0.0.1:5432/tg_group?sslmode=disable" \
go run ./cmd/api
```

三项 CAP 配置必须同时设置；全部留空时 CAPTCHA 功能关闭。可选设置 `CAP_WIDGET_URL` 指向自托管或镜像的官方 `widget.js`，例如 `https://captcha.example.com/assets/widget.js`；不设置则使用固定版本 CDN。若使用 Cap Standalone 的 assets server，请按官方文档固定 `WIDGET_VERSION`/`WASM_VERSION`，并确保浏览器可加载对应 WASM 资源。配置完成后，访问 `GET /api/v1/security/captcha` 应返回 `enabled=true` 和公开 endpoint（不含 secret）。注册页和登录页加载官方验证组件；完成后提交的 token 会由 Go 后端调用 Cap Standalone 的 `/{site_key}/siteverify` 校验。验证失败、token 重放或 Cap 服务不可用时，注册/登录会被拒绝。

验收步骤：
1. 浏览器打开注册页和登录页，确认显示 CAP 官方验证组件且可完成验证。
2. 完成验证后注册或登录，确认成功。
3. 不完成验证直接调用注册/登录 API，应返回 400。
4. 把已使用过的 token 再提交一次，应被拒绝；Cap token 为一次性。
5. 临时停止 Cap 服务后提交，API 应返回 503，不得绕过验证继续创建账号/会话。
6. 检查 `GET /api/v1/security/captcha` 的响应，确认没有 `CAP_SECRET` 或任何站点私钥字段。

本地不配置 CAP 时，页面不会显示验证组件，原有开发流程保持可用。Telegram 群内使用 Inline Keyboard，不能直接渲染浏览器 CAP 组件；群成员验证继续使用数学题与可选频道关注检查。


## 登录设备管理测试

1. 在两个独立浏览器会话登录同一个账号。
2. 从任意一个会话调用 `GET /api/v1/auth/sessions?limit=20`，应看到两条会话，并准确标记当前设备。
3. 尝试撤销当前设备对应的 session ID，应返回 400，当前会话仍然有效。
4. 撤销另一设备的 session ID，应返回 204；该设备旧 access token 和 refresh token 均应失效。
5. 当前设备刷新会话列表，应显示被撤销设备的状态为已退出；安全记录中出现 `auth.session_revoke`。
6. 使用另一个账号尝试撤销该 session ID，应无法影响原账号的会话。

## 账号安全审计记录测试

1. 登录账号后调用 `GET /api/v1/auth/security-events?limit=20`，应返回当前账号的注册/登录、退出、会话续期、密码修改和 Telegram 身份操作事件（仅显示实际执行过的操作）。
2. 退出登录后，数据库 `audit_logs` 应新增 `auth.logout` 记录，包含当前账号 actor、来源 IP 和时间。
3. 使用另一个账号调用该接口，只能看到另一个账号自身的事件，不能读取同租户其他用户的认证历史。
4. limit 默认 20，允许 1–100；传入 0、101 或非数字值应返回 400。
5. 安全事件只返回操作元数据和来源 IP，不返回密码、Access Token、Refresh Token 或 Token 哈希。

## 登录失败频率限制测试

1. 使用测试账号和固定测试来源 IP，连续提交 5 次错误密码；前 5 次应返回 401。
2. 在 15 分钟滚动窗口内继续使用正确密码登录，同一邮箱应被限制并返回 429，响应头包含 `Retry-After: 900`。
3. 从同一来源 IP 使用另一个账号登录，也应受 IP 维度限制；从新 IP 使用已被限制的邮箱仍应受邮箱维度限制。
4. 使用全新的邮箱和 IP 应可正常登录。
5. 等待失败记录逐步超出 15 分钟窗口后，原邮箱/IP 应恢复登录。不要在生产账号或生产 IP 上执行失败次数测试。
6. 数据库仅保存 email/IP 标识的 SHA-256 摘要，不保存明文登录密码或明文邮箱/IP 到限流表。

默认阈值为同一邮箱或来源 IP 在滚动 15 分钟内 5 次失败。当前使用 PostgreSQL 实现限流，无 Redis 依赖；反向代理部署时，应用按 HTTP 连接的 RemoteAddr 识别来源 IP，不信任客户端自行提交的 X-Forwarded-For。

## 修改密码与会话安全测试

1. 使用同一账号在两个浏览器或两个独立无痕会话分别登录，确认两端均可访问工作台。
2. 在第一个会话打开账号工作台“修改登录密码”，输入正确当前密码和新的 10–72 字节密码。
3. 修改成功后，第一个会话应继续保持登录，并收到新 access/refresh token。
4. 第二个会话的旧 access token 应无法继续访问 API；其 refresh token 也应返回 401。第二个会话需使用新密码重新登录。
5. 使用旧密码登录应返回 401；使用新密码登录应成功。
6. 使用错误当前密码、少于 10 字节的新密码、超过 72 字节的新密码或与旧密码相同的新密码提交，均应被拒绝，且原密码和现有会话不应被修改。
7. 检查 `audit_logs` 中是否出现 `auth.password_change` 操作记录。

## Telegram 网站账号身份绑定测试

先使用 Telegram 官方 `@BotFather` 创建平台专用身份 Bot，取得 Token 和 Bot username。启动 API 时增加：

```bash
TELEGRAM_AUTH_BOT_TOKEN="<平台身份 Bot Token>" \
TELEGRAM_AUTH_BOT_USERNAME="<平台身份 Bot username>" \
DATABASE_URL="postgres://tg_group:tg_group_dev@127.0.0.1:5432/tg_group?sslmode=disable" \
go run ./cmd/api
```

两项 Telegram 配置必须同时设置；身份 Bot Token 只放在后端环境变量/密钥管理系统，不提交到仓库。该 Bot 使用独立 getUpdates 长轮询，不能同时启用 webhook，也不能与租户自己的群管 Bot 共用 Token。

1. 登录网站打开账号工作台，找到“Telegram 身份绑定”。
2. 点击“绑定 Telegram”，网站创建 5 分钟有效的一次性绑定链接并跳转 Telegram。
3. 在平台身份 Bot 私聊窗口点击 Start。
4. Bot 应回复绑定成功；返回网站刷新工作台，应显示 Telegram ID 和官方 Update 中的用户名/名称。
5. 再创建一个绑定链接并尝试使用旧链接，应因旧挑战失效或过期而失败。
6. 使用第二个网站账号尝试绑定同一个 Telegram 账号，应失败；一个 Telegram ID 全局只能绑定一个网站账号。
7. 点击“解除绑定”，确认 Telegram 资料清空且数据库 `audit_logs` 新增 `auth.telegram_unlink` 记录。
8. 检查 `users.telegram_id`，该值只能由 Telegram 身份 Bot 的 Update 处理流程写入，前端没有输入 Telegram ID 的表单或 API 字段。

未配置平台身份 Bot 时，网站会显示未绑定状态，但创建绑定链接接口返回 503；不能把这一状态误认为 Telegram 绑定功能已经完成真实环境验收。

## Refresh Token 会话续期测试

1. 注册或登录，检查响应同时包含 `token`、`expires_at`、`refresh_token` 和 `refresh_expires_at`。
2. 使用返回的 refresh_token 调用 `POST /api/v1/auth/refresh`，确认获得新的 access token 和 refresh token。
3. 再次提交旧 refresh_token，应返回 401；该会话族所有访问会话均被撤销，旧 access token 和刚签发的 access token 都不能继续访问受保护 API。
4. 使用新账号会话重新登录，执行退出登录；随后使用该会话 refresh_token 续期应返回 401。
5. 在网站正常登录后等待访问令牌过期（15 分钟），触发工作台或受保护 API 请求，浏览器应自动续期并重试，不应要求用户重新登录。
6. 在浏览器 Network 面板确认多个同时返回 401 的请求只触发一次 refresh 请求；刷新令牌不得出现在 URL、前端日志或服务端日志。
7. 删除浏览器当前会话存储中的 refresh token，再等待 access token 过期，受保护 API 应要求重新登录。

测试需要启用 PostgreSQL；GitHub CI 的集成测试覆盖轮换、重放撤销和退出登录撤销。


## 警告递进处罚自动重试队列验证（新增，待 CI/集成环境验收）

- 数据库迁移新增 `000019_warning_escalation_retry_queue`，需由正常启动迁移流程自动应用；不要手工只更新其中一个迁移目录。仓库同时维护 `internal/database/migrations`（Go embed 使用）与根目录 `migrations`，两份 SQL 必须保持一致。
- API 进程启动后，在 Bot 加密服务可用时会启动警告递进处罚恢复 worker。worker 每 10 秒扫描 PostgreSQL 队列，不依赖 Redis。
- 首次处罚失败后 30 秒进入自动重试；后续采用指数退避，最长 1 小时，自动尝试上限 8 次。超过 5 分钟仍处于 `processing` 的记录会被回收并重新领取。
- 达到自动重试上限后记录继续保留为 `failed`，网站管理员仍可通过原执行历史入口手动重试；手动重试会重新开启自动重试计数。
- 单元测试：`go test ./internal/bots -run WarningEscalationRetry -count=1`。
- 全量后端测试：`go test ./...`；静态检查：`go vet ./...`。
- 真实测试必须使用专用 Telegram 测试群和普通测试成员，分别覆盖成功、Bot 权限不足、目标管理员保护、Telegram 网络超时、自动重试、达到上限后人工重试。
- **执行语义说明**：队列使用至少一次执行语义。若 Telegram 已完成处罚，但服务在写入成功状态前崩溃，超过 lease 后可能再次调用 Telegram API。测试需核对封禁/禁言状态与重复调用的实际影响；不得宣称跨 Telegram API 与 PostgreSQL 已实现分布式 exactly-once。
- 本文档描述新 worker 的预期验证步骤；在 GitHub CI 与真实 Telegram 测试完成前，不代表本批已验收。
