# TG-Group

Telegram 多用户、多 Bot、多群组管理平台。

## 技术约束

- Web：React + TypeScript + Vite。
- Backend：Go（Golang），HTTP 使用标准库 `net/http`，PostgreSQL 使用 pgx/v5。
- UI：严格使用 shadcn/ui 官方 CLI 生成的官方组件源码，不自制替代控件，不混用其他 UI 组件库。
- 数据：PostgreSQL + Redis（Redis 当前仅配置本地开发依赖，Go 服务尚未接入）。
- 生产部署目标：云端 CI 预构建镜像，服务器只拉取镜像运行，不在服务器编译源码。

## 当前可运行功能

- TG-Group 产品介绍首页。
- 网站注册、登录页面。
- 登录后账号工作台、当前用户信息和退出登录。
- Go API：健康检查、邮箱注册、登录、当前用户、退出登录。
- PostgreSQL 版本化迁移和注册会话集成测试。
- Telegram Bot Token 加密管理、后台长轮询、群组自动发现、消息日志、成员增量采集和手动成员处罚。
- 群组规则配置/执行记录、入群数学题和可选频道关注验证。
- 网站 CAP Standalone siteverify 服务端验证（需部署配置）。
- Telegram 身份绑定首批代码：一次性挑战、专用平台身份 Bot 自动获取 Telegram ID、全局唯一约束、解绑和审计；需配置平台 Bot 凭据后联调。

Refresh token、密码找回、邮箱验证、登录限流、完整角色授权、Telegram 注册管理 Bot 对话流程、完整群管矩阵、支付、AI 审核和生产部署仍在开发中.

## 本地启动

环境要求：Go 1.23+、Node.js 22+、pnpm 10+、Docker Compose。

1. 启动 PostgreSQL 和 Redis：

```bash
docker compose -f compose.dev.yml up -d
```

2. 启动 Go API：

```bash
go mod tidy
DATABASE_URL="postgres://tg_group:tg_group_dev@127.0.0.1:5432/tg_group?sslmode=disable" go run ./cmd/api
```

3. 另开终端启动前端：

```bash
pnpm install
TG_GROUP_API_PROXY="http://127.0.0.1:8080" pnpm dev
```

打开 Vite 输出的地址即可访问首页。若需测试 Telegram 身份绑定，额外配置专用平台 Bot 环境变量，见 [本地测试文档](docs/LOCAL-TESTING.md)。注册、登录和工作台路径：

- `/` 首页
- `/register` 注册
- `/login` 登录
- `/dashboard` 账号工作台

详细步骤与 API 请求格式见下方文档。

## 项目文档

- [产品需求基线](docs/REQUIREMENTS.md)
- [首页介绍文案](docs/HOMEPAGE-CONTENT.md)
- [API 文档](docs/API.md)
- [本地开发与测试指南](docs/LOCAL-TESTING.md)
- [开发计划](docs/DEVELOPMENT-PLAN.md)
- [开发日志](docs/DEVELOPMENT-LOG.md)
