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
