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
- Go API：健康检查、邮箱注册、登录、当前用户、退出登录及 15 分钟访问令牌 / 30 天滚动 Refresh Token 会话续期。
- PostgreSQL 版本化迁移和注册会话集成测试。
- Telegram Bot Token 加密管理、后台长轮询、群组自动发现、消息日志、成员增量采集和手动成员处罚。
- 群组规则配置/执行记录、入群数学题和可选频道关注验证。
- 网站 CAP Standalone siteverify 服务端验证（需部署配置）。
- 平台 Telegram 注册/身份 Bot 第一阶段：私聊 `/start` 自动创建平台账号并绑定 Telegram ID、生成随机初始密码；`/account` 查询本人账号；网站支持用户 ID 登录；网站账号也可通过一次性挑战绑定 Telegram。需配置平台 Bot 凭据后进行真实联调。

平台 Bot 的管理员可视化配置、Inline Keyboard 主菜单、套餐/订单查询和群管 Bot Token 提交流程，以及完整群管矩阵、支付、AI 审核仍在开发中。

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

## Docker 一键部署（无需服务器编译或手动配置）

支持 Docker Engine + Docker Compose Plugin 的 Linux 服务器。部署脚本会在 Ubuntu/Debian 或 RHEL 系（dnf）环境自动安装 Docker（如尚未安装），自动生成数据库密码和 Bot Token 加密密钥，拉取 GHCR 预构建镜像并启动 PostgreSQL、API、Web。服务器不会执行 Go 或前端构建。

首次部署只需执行：

```bash
curl -fsSL https://raw.githubusercontent.com/seanldnet/Tg-group/main/deploy/install.sh | sudo bash
```

- 访问：`http://服务器公网IP:18680`
- 测试管理员：用户名 `admin`，密码 `123456`
- 数据库只在 Docker 内网开放，不映射到宿主机端口。
- 数据持久化在 Docker volume `tg_group_postgres`；安装生成的数据库与 Bot 加密密钥位于 `/opt/tg-group/.env`。
- 平台注册/身份 Bot 与用户群管 Bot 分离。当前平台 Bot 凭据暂时通过 `/opt/tg-group/.env` 配置：`TG_GROUP_TELEGRAM_AUTH_BOT_TOKEN` 和 `TG_GROUP_TELEGRAM_AUTH_BOT_USERNAME`（用户名不带 `@`），两项需同时填写；修改后执行 `docker compose -f docker-compose.yml up -d --force-recreate api`。管理员网页配置页正在开发，尚未交付。
- 查看日志：`cd /opt/tg-group && docker compose logs -f`
- 更新镜像：`cd /opt/tg-group && docker compose pull && docker compose up -d`
- 停止：`cd /opt/tg-group && docker compose down`（不会删除数据库卷）；彻底删除数据需另行执行 `docker compose down -v`。

**首次使用前的一次性仓库操作：**GHCR 新建镜像默认可能是 Private，GitHub Actions 的仓库令牌不能代替包所有者修改可见性。请仓库所有者打开 [API 镜像包](https://github.com/users/seanldnet/packages/container/tg-group-api) 和 [Web 镜像包](https://github.com/users/seanldnet/packages/container/tg-group-web)，在 Package settings 中将 Visibility 设为 **Public**。这是 GitHub 侧的一次性设置，不需要在服务器登录镜像仓库，也不需要编辑部署配置文件。

**安全提示：**这是测试用默认管理员密码，任何公网环境都必须在测试后立即修改密码并限制访问。请在云服务器安全组/防火墙放行 TCP 18680。
