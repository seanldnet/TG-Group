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
DATABASE_URL="postgres://tg_group:tg_group_dev@127.0.0.1:5432/tg_group?sslmode=disable" go run ./cmd/api
```

首次启动会自动执行 PostgreSQL 迁移。API 默认监听 `http://127.0.0.1:8080`。

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
5. 退出登录后会话撤销，可重新访问 `/login` 登录。

也可以直接检查：

- `GET /healthz`
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/me`
- `POST /api/v1/auth/logout`

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
