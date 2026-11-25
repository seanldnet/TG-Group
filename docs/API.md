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

### 当前边界

- 已完成 Token 验证、加密存储、租户范围列表/状态/轮换/解绑 API。
- 尚未启动 Telegram 长轮询或 Webhook 接收器，也未完成 Bot 群组自动发现和权限检测。
- Bot 的 active/disabled 是平台侧管理状态；尚未建立运行时 worker，因此不代表 Bot 已开始或停止接收消息。

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
- API 代码已提交，但 PostgreSQL 集成测试和完整端到端验证仍待执行。
