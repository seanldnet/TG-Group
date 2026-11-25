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
