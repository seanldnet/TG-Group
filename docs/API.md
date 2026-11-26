# TG-Group API（当前已实现部分）

基础前缀：`/api/v1`。请求与响应均使用 JSON。需要登录的接口使用：

```http
Authorization: Bearer <token>
```

## 健康检查

`GET /healthz`

返回 `200` 和服务状态。该端点不代表数据库一定可用。

## CAP.js 人机验证

CAP 采用官方 Cap Standalone 服务端和官方 `cap-widget` Web Component。服务端配置 `CAP_BASE_URL`、`CAP_SITE_KEY`、`CAP_SECRET` 三项环境变量后启用；三项必须同时设置。可选 `CAP_WIDGET_URL` 用于指定官方 widget 脚本的自托管/镜像地址，默认固定使用官方 `cap-widget@0.1.58` CDN。公网地址必须使用 HTTPS，只有 localhost/loopback 开发地址允许 HTTP。

- `GET /api/v1/security/captcha`：公开读取是否启用及前端组件 endpoint；绝不返回站点 secret。
- 网站注册、登录请求在启用时必须携带 `captcha_token`。
- Go 服务端向 Cap Standalone 的 `/{site_key}/siteverify` 发送 `secret` 与 `response`，仅接受 HTTP 成功且 JSON `success=true`；验证服务超时/不可用时失败关闭，不创建会话。
- CAP token 一次性使用；注册或登录业务失败后，前端重新加载组件获取新 token。
- 前端固定使用官方 `cap-widget@0.1.58`，CAP 站点 URL 必须可由用户浏览器访问，并在 Cap 的 CORS 设置中允许 TG-Group 网站域名。

`CAP_SECRET` 只保存在后端环境变量/密钥管理系统中。不要放入前端变量、代码仓库或 API 响应。Telegram 群内无法直接渲染浏览器组件，入群验证当前继续使用 Telegram Inline Keyboard 数学题与频道成员校验；CAP.js 不替代该流程。

## 注册

`POST /api/v1/auth/register`

请求：

```json
{
  "email": "owner@example.com",
  "password": "at-least-10-characters",
  "display_name": "社群管理员",
  "captcha_token": "仅在 CAP 启用时必填"
}
```

- 邮箱会去除首尾空格并转换为小写。
- 密码长度限制为 10–72 字节，使用 bcrypt 哈希保存。
- 注册时在同一数据库事务内创建租户、用户、会话和审计记录。
- 邮箱全局唯一；每个新注册用户会创建自己的租户。
- 成功返回 `201`，响应包含一次性展示的 Bearer token、过期时间和用户资料。

## 登录

`POST /api/v1/auth/login`

请求：

```json
{
  "email": "owner@example.com",
  "password": "at-least-10-characters",
  "captcha_token": "仅在 CAP 启用时必填"
}
```

成功返回 `200` 和新会话 token。锁定/停用账号或停用租户不能登录。

登录失败限制：同一规范化邮箱或同一来源 IP 在滚动 15 分钟内累计 5 次密码验证失败后，后续登录请求返回 `429 Too Many Requests`，并附带 `Retry-After: 900`。邮箱与 IP 标识在数据库中仅保存 SHA-256 摘要；相同标识的并发登录通过 PostgreSQL advisory lock 串行检查，避免并发绕过阈值。失败记录按 24 小时保留边界进行分批清理。成功登录不增加失败计数。

## 会话续期（Refresh Token）

注册和登录成功后，服务端返回：
- `token` / `expires_at`：短时访问令牌，默认有效期 15 分钟。
- `refresh_token` / `refresh_expires_at`：刷新令牌，默认有效期 30 天。

刷新令牌仅保存 SHA-256 摘要。客户端将两种令牌保存在当前浏览器会话存储中，不写入 URL、日志或跨会话持久化存储。

`POST /api/v1/auth/refresh`

请求：

```json
{ "refresh_token": "当前刷新令牌" }
```

成功返回新的访问令牌和新的刷新令牌；旧刷新令牌立即失效。刷新令牌按 family 绑定会话族，检测到已轮换令牌被重放时，会撤销该会话族下的刷新令牌和访问会话。账号或租户停用时也会拒绝续期。

网站 API 客户端在受保护请求收到 401 后自动尝试续期并重试原请求；并发请求共用同一个续期操作，避免并发轮换误触发重放保护。续期失败会清理本地令牌并要求重新登录。

退出登录会同时撤销当前访问会话及其 refresh-token family。刷新令牌不通过 Cookie 自动发送，当前实现继续使用 JSON 请求体，需配合 HTTPS 部署并防范 XSS。

## 修改登录密码

`POST /api/v1/auth/password/change`

需要 Bearer 登录。请求：

```json
{
  "current_password": "当前密码",
  "new_password": "新的至少 10 个字符的密码"
}
```

新密码长度为 10–72 字节，不能与当前密码相同。服务端先验证当前会话和原密码，再在同一 PostgreSQL 事务中更新 bcrypt 密码哈希、撤销该用户所有既有访问会话及 refresh token，并为当前设备签发一组新的 access/refresh token。成功返回新的会话对象；其他设备会立即失效，需要使用新密码重新登录。密码修改操作写入 `audit_logs`。原密码错误返回 401，参数/密码长度错误返回 400。

## 当前用户

`GET /api/v1/auth/me`

携带 Bearer token，返回当前用户资料。会话过期、撤销或租户/用户停用时返回 `401`。

## 退出登录

`POST /api/v1/auth/logout`

携带 Bearer token，将当前会话标记为撤销。成功返回 `204`。

退出操作会写入 `auth.logout` 审计事件。

## 账号安全记录

`GET /api/v1/auth/security-events?limit=20`

需要 Bearer 登录。返回当前用户最近的认证安全事件，默认 20 条，limit 范围 1–100。仅查询当前用户作为 actor 的 `auth.*` 审计记录，包含操作类型、资源、来源 IP 和时间；不会返回同租户其他用户的记录。覆盖注册、登录、退出、会话续期、密码修改及 Telegram 身份绑定/解绑。

## Telegram Bot 管理

所有 Bot 接口均要求 Bearer 登录认证，并按当前用户所属租户过滤数据。

### 添加并验证 Bot

`POST /api/v1/bots`

请求：

```json
{ "token": "从 @BotFather 获取的 Bot Token" }
```

服务端调用 Telegram Bot API 的 `getMe` 验证 Token，验证成功后使用 AES-256-GCM 加密保存。数据库保存 Token 指纹用于唯一性检查；响应不会返回 Token 明文。

### 查询 Bot

`GET /api/v1/bots`

返回当前租户已登记的 Bot 列表，不包含 Token 或密文。

### 查询已发现群组

`GET /api/v1/groups`

返回当前租户的群组、超级群组和频道列表；服务端按当前用户的 tenant_id 过滤，不接受客户端指定租户。

### 启用/停用 Bot

`PATCH /api/v1/bots/{botID}/status`

请求：

```json
{ "status": "active" }
```

状态只允许 `active` 或 `disabled`。

### 轮换 Token

`PUT /api/v1/bots/{botID}/token`

请求格式与添加 Bot 相同。新 Token 必须通过 Telegram 验证，且必须属于同一个 Telegram Bot 身份。

### 解绑 Bot

`DELETE /api/v1/bots/{botID}`

删除当前租户对应 Bot 记录及已保存密文。

### Bot Token 加密配置

启动 API 前设置 `BOT_TOKEN_ENCRYPTION_KEY`。值必须是 Base64 编码的 32 字节随机密钥，可使用：

```bash
openssl rand -base64 32
```

生产环境应将密钥保存在密钥管理系统或受保护的环境变量中，并做好独立备份；更换密钥前必须先规划数据重新加密，否则旧密文将无法解密。未配置密钥时 Bot 管理 API 返回 `503`。

### 群组自动发现运行时

服务配置了有效的 `BOT_TOKEN_ENCRYPTION_KEY` 后，API 进程会为平台状态为 `active` 的 Bot 启动独立 Telegram `getUpdates` 长轮询 worker。当前监听更新类型包括 message、edited_message、channel_post、my_chat_member 和 chat_member。

发现的 group、supergroup、channel 会写入 `telegram_groups`，并通过每个 Bot 独立的 update offset 避免成功处理的更新重复消费。worker 会随 Bot 启停状态协调，并在 API 收到终止信号时停止。

群组发现是事件驱动的：Bot 需要先加入目标群组/频道并收到 Telegram 更新。当前提供 `GET /api/v1/groups` 查询当前租户已发现的群组，响应字段包含群组 ID、Chat ID、标题、类型、来源 Bot 和最近发现时间；当收到 Bot 被移出群组或封禁的 my_chat_member 更新时，状态会标记为 left 或 kicked。

### 群消息事件日志

`GET /api/v1/groups/{groupID}/events?limit=50`

返回指定群组最近的消息事件，`limit` 可选，范围为 1–100，默认 50。响应包含 Telegram 用户 ID、发送者用户名/名称、消息类型、文本或媒体说明文字、Telegram 发送时间和平台接收时间。查询必须携带 Bearer token，SQL 同时按当前租户和 groupID 过滤；其他租户无法读取该群日志。

当前保存消息文本/Caption 和发送者资料，不下载或保存媒体文件本体。消息事件没有自动清理策略，删除对应 Bot/群组时通过外键级联删除。部署者应将群消息日志视为用户内容数据，限制数据库和管理后台访问，并在正式上线前确定保留期限及隐私告知。


### 群成员列表

`GET /api/v1/groups/{groupID}/members?limit=100`

返回当前租户指定群组最近活动的成员资料，`limit` 可选，范围为 1–200，默认 100。包含 Telegram 用户 ID、用户名、显示名称、Bot 标识、成员状态、管理员权限快照、加入时间和最近活动时间。查询同时按当前租户和 groupID 过滤。

成员资料通过 Telegram `chat_member` 更新以及普通消息中的 `new_chat_members` / `left_chat_member` 事件增量维护。Telegram 仅向满足条件的 Bot 投递成员状态更新：Bot 必须在群内具备管理员身份，且 `chat_member` 必须列入 `allowed_updates`。因此本列表是 Bot 加入并开始接收更新后的增量记录，不会自动回填 Bot 加入前的完整历史成员名单。


### Bot 群组权限实时检测

`GET /api/v1/groups/{groupID}/permissions`

通过 Telegram `getChatMember` 实时读取当前群组内 Bot 自身身份及权限，返回 Bot 状态、管理群组、删除消息、限制成员、邀请成员、固定消息、管理话题等权限。群组和 Bot 均按当前租户查询；群组或 Bot 停用时不执行检测。群主身份按 Telegram 语义视为拥有管理权限。

### 群成员处罚

`POST /api/v1/groups/{groupID}/members/{userID}/action`

请求示例：

```json
{ "action": "mute", "duration_minutes": 60 }
```

支持 `ban`（封禁）、`unban`（解封）、`mute`（限时禁言）、`unmute`（解除禁言）。禁言时长为 1–525600 分钟；其他动作的 duration_minutes 必须为 0 或省略。执行前服务端实时核验 Bot 是群管理员/群主且具备 `can_restrict_members`，并向 Telegram 查询目标成员身份；不会处罚群主或管理员，也会拒绝与目标当前状态不匹配的动作。解除禁言时先读取群组默认权限并按群组默认权限恢复，不会直接把成员提升为全部权限；若 Telegram 未返回默认权限则拒绝解除禁言。Telegram 操作成功后更新本地成员状态并写入 audit_logs。外部 Telegram 操作与本地数据库事务无法构成分布式原子事务，因此若后续本地审计写入失败，接口会返回错误并在日志中体现“Telegram action succeeded”，应人工核对 Telegram 实际状态。


### 群组规则配置（第一阶段：配置管理）

规则数据按当前登录用户的租户和群组隔离。支持以下类型：`keyword_filter`、`sensitive_word`、`anti_spam`、`anti_ad`、`welcome`、`leave_notice`、`join_verification`。

- `GET /api/v1/groups/{groupID}/rules`：查询当前群组规则。
- `POST /api/v1/groups/{groupID}/rules`：创建规则。
- `PUT /api/v1/groups/{groupID}/rules/{ruleID}`：完整更新规则。
- `DELETE /api/v1/groups/{groupID}/rules/{ruleID}`：删除规则。

创建/更新请求示例（关键词过滤）：

```json
{
  "rule_type": "keyword_filter",
  "name": "广告关键词",
  "enabled": true,
  "priority": 100,
  "config": { "words": ["广告", "推广"], "action": "delete" }
}
```

配置由服务端按规则类型进行严格字段校验。关键词/敏感词使用 `words` 数组；反广告使用 `patterns` 数组；反刷屏使用 `max_messages`、`window_seconds`；过滤动作目前仅接受安全子集 `delete`；欢迎和离群通知使用 `text`。优先级范围为 0–1000，数值越小越靠前。同一群组内同类型规则名称不可重复。

入群验证配置示例：`{"challenge_type":"math","timeout_seconds":120,"failure_action":"kick","required_channels":["@tg_group_news"]}`。仅支持 `math` 数学题，超时 30–600 秒；超时处理支持 `restrict`（保持禁言）或 `kick`（封禁后立即解封，实现移出群组）。`required_channels` 可选，最多 10 个，支持公开频道用户名（如 `@channel_name`）或 Telegram 数字频道 ID（`-100...`）；答题正确后，系统会逐个调用 `getChatMember` 校验成员身份，未加入全部频道时不解除群组限制。Bot 必须加入每个目标频道且具备足以查询成员状态的权限（建议设为频道管理员）；若 Telegram 无法返回成员状态则失败关闭，保持成员受限。当前仅对 Telegram supergroup 执行；Bot 必须是群管理员/群主且具备 `can_restrict_members`。新成员入群后立即限制发言，Bot 发送带 Inline Keyboard 的随机加法题；仅该成员本人可点击验证按钮。答对后读取群组默认权限并按默认权限恢复，不会授予超出群组默认设置的权限。挑战过期由后台长轮询 worker 周期处理。若 Bot 缺少权限、群组不是 supergroup 或无法读取群默认权限，会跳过或保留限制并记录服务端日志。

规则执行引擎已接入长轮询消息持久化流程：新消息事件成功插入后才执行规则，重复 update 不会重复执行。关键词/敏感词和反广告按不区分大小写的子串匹配；反刷屏按同一群组、同一发送者在配置时间窗内的已记录消息数量触发。过滤类规则当前只执行删除当前命中消息，并在调用 Telegram 前实时检查 Bot 是否为群管理员且具备 `can_delete_messages`；权限不足时跳过并记录原因。欢迎和离群通知分别基于消息中的 `new_chat_members` 与 `left_chat_member` 事件发送，模板支持 `{user}`、`{username}`、`{group}` 占位符。执行结果写入 `telegram_group_rule_events`，并以群组、update、规则、动作和目标用户做幂等保护。规则执行失败不会阻塞已持久化消息的 update offset。

`GET /api/v1/groups/{groupID}/rule-events?limit=50` 返回当前群组最近的规则动作记录，limit 范围 1–100，包含动作、状态、目标用户/消息 ID、执行时间和失败/跳过原因；网站在规则管理页提供“执行记录”入口。查询按当前租户与群组双重过滤。

当前执行边界：反广告使用配置的简单子串匹配，不是正则或语义识别；反刷屏只删除触发阈值的当前消息，不会批量清理历史消息或自动禁言；尚未实现 chat_member 更新驱动的离群通知、频道关注验证、CAP.js 官方服务端验证和高级人机验证；入群数学题是基础验证，不代表 CAP 已接入。真实 Telegram 权限与消息联调仍需在测试群组验收。

### 当前边界

- 长轮询当前记录 message、edited_message、channel_post 更新中的发送者和消息内容；频道帖子可能没有普通用户发送者，因此 Telegram 用户 ID 为空。Telegram Bot 的群组隐私模式会限制其可接收的普通群消息；若需要记录群内全部普通消息，需要按 Telegram 官方规则关闭该 Bot 的群组隐私模式，或将 Bot 设为群管理员，并确保群组授予相应权限。
- 图片、文件、视频、贴纸、语音等只记录消息类型及 Caption，不保存媒体文件。
- 尚未实现消息全文搜索、分页游标、导出、自动保留期限、Webhook 模式、全量成员同步、自动处罚规则和完整群管规则引擎。成员处罚当前仅覆盖手动封禁/解封/限时禁言/解除禁言。
- Bot 的 active/disabled 是平台侧管理状态；后台 worker 会按该状态启动/停止长轮询，但不代表 Telegram 已授予 Bot 管理员权限。

## 通用错误

错误响应格式：

```json
{ "error": "说明" }
```

主要状态码：`400` 请求格式或参数错误；`401` 未认证或凭据错误；`403` 账号停用；`409` 邮箱已注册；`503` 未配置数据库，认证服务不可用。

## 当前边界

- 认证 API 需要设置 `DATABASE_URL` 并完成启动迁移。
- 当前 token 是 30 天有效期的随机不透明 Bearer token；数据库只保存 SHA-256 摘要，不保存明文 token。
- 暂未实现 refresh token、密码找回、邮箱验证、登录限流、Telegram ID 绑定和基于角色的业务授权。
- 注册会话与 Bot 生命周期 PostgreSQL 集成测试已纳入 GitHub Actions；Bot 管理 API 仍需真实 Telegram Token 和部署环境进行端到端验收。


## Telegram 身份绑定

平台身份 Bot 与用户自行接入的群管 Bot 是两套独立配置。启动 API 时同时配置：

- `TELEGRAM_AUTH_BOT_TOKEN`：平台专用身份 Bot Token，仅后端读取。
- `TELEGRAM_AUTH_BOT_USERNAME`：该 Bot 的用户名，不含前导 `@` 也可以，服务会自动去掉前导 `@`。

配置两项后，API 进程启动独立的 Telegram `getUpdates` 长轮询 worker。该专用 Bot 不得同时配置 webhook，也不要将同一个 Token 添加为租户群管 Bot。

### 查询绑定状态

`GET /api/v1/auth/telegram`

需要 Bearer 登录。返回 `linked` 和 Telegram 官方 Update 获取的用户资料。

### 创建绑定链接

`POST /api/v1/auth/telegram/link`

需要 Bearer 登录，不需要请求体。服务端生成 5 分钟有效的一次性随机挑战值，数据库只保存 SHA-256 摘要，并返回 `https://t.me/{bot_username}?start={challenge}`。

用户跳转平台身份 Bot 并点击 Start。后端只接受私聊消息，且必须满足 Telegram Update 的 `message.from.id == message.chat.id`；绑定的 Telegram ID、用户名和名称全部取自 Telegram 官方 Update，网站请求不能提交或覆盖这些字段。成功后挑战立即消费，并在同一数据库事务中更新用户资料与写入审计记录。

### 解除绑定

`DELETE /api/v1/auth/telegram`

需要 Bearer 登录。解除绑定会清空 Telegram ID 与资料，并写入审计日志。Telegram ID 全局唯一；同一个 Telegram 账号不能同时绑定多个网站账号。

**安全边界**：挑战码短时有效、只保存摘要、一次性消费；旧挑战在创建新挑战时失效。真实 Telegram Bot Token、用户名和网络可达性需要部署环境配置并完成真实 Telegram 联调后，才算端到端验收。
