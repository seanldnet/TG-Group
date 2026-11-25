# TG-Group API（当前已实现部分）

基础前缀：`/api/v1`。请求与响应均使用 JSON。需要登录的接口使用：

```http
Authorization: Bearer <token>
```

## 健康检查

`GET /healthz`

返回 `200` 和服务状态。该端点不代表数据库一定可用。

## 注册

`POST /api/v1/auth/register`

请求：

```json
{
  "email": "owner@example.com",
  "password": "at-least-10-characters",
  "display_name": "社群管理员"
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
  "password": "at-least-10-characters"
}
```

成功返回 `200` 和新会话 token。锁定/停用账号或停用租户不能登录。

## 当前用户

`GET /api/v1/auth/me`

携带 Bearer token，返回当前用户资料。会话过期、撤销或租户/用户停用时返回 `401`。

## 退出登录

`POST /api/v1/auth/logout`

携带 Bearer token，将当前会话标记为撤销。成功返回 `204`。

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

支持 `ban`（封禁）、`unban`（解封）、`mute`（限时禁言）、`unmute`（解除禁言）。禁言时长为 1–525600 分钟；其他动作的 duration_minutes 必须为 0 或省略。执行前服务端实时核验 Bot 是群管理员/群主且具备 `can_restrict_members`，并向 Telegram 查询目标成员身份；不会处罚群主或管理员，也会拒绝与目标当前状态不匹配的动作。Telegram 操作成功后更新本地成员状态并写入 audit_logs。外部 Telegram 操作与本地数据库事务无法构成分布式原子事务，因此若后续本地审计写入失败，接口会返回错误并在日志中体现“Telegram action succeeded”，应人工核对 Telegram 实际状态。

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
