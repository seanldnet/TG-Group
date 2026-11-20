# 交付与部署标准

本文档是 TG Group 后续开发、CI/CD、发布和生产交付的强制基线。正式交付提供两种互相独立的安装方式：Docker Compose 一键部署，以及宝塔面板上传 Garble Go 二进制后通过网页安装向导部署。

## 1. Docker Compose 一键部署（内置 PostgreSQL + Redis）

Docker 方式必须做到“拉取镜像、启动容器、打开后台即可使用”，服务器不编译、不需要手工安装数据库。

### 部署组成

- GHCR 发布 TG Group 应用预构建镜像。
- Compose 同时编排 TG Group 应用、PostgreSQL 数据库和 Redis 服务。
- PostgreSQL 与 Redis 使用独立持久化 volume；容器升级或重建不得丢失业务数据。
- Compose 使用服务健康检查和启动依赖，应用在 PostgreSQL/Redis 就绪后再启动。
- 首次启动自动执行数据库迁移和系统初始化，迁移必须幂等；重复执行 `docker compose up -d` 不得重复创建管理员或重置已有密码。

### 默认管理员登录

首次全新部署自动初始化默认管理员：

- 用户名：`admin`
- 密码：`123456`

初始化完成后，用户打开部署地址即可进入登录页面，使用以上账号密码登录管理后台。默认账号只在首次初始化、且数据库中尚无管理员时创建；升级或重启不得覆盖用户修改后的管理员凭据。

安全要求：默认凭据是首次安装引导凭据。部署完成后应尽快修改密码；生产环境必须通过反向代理启用 HTTPS，并限制管理后台访问范围。不得把默认密码写入日志或公开运行状态接口。

### Bot Token 加密密钥

添加 Telegram Bot 时，服务端会先调用 Telegram `getMe` 验证 Token，再使用 AES-256-GCM 加密后写入 PostgreSQL。必须在 `.env` 中设置独立的 32 字节十六进制密钥：

```bash
openssl rand -hex 32
```

将输出写入 `BOT_TOKEN_ENCRYPTION_KEY`。同一套数据库在升级、迁移或恢复备份时必须保留原密钥；丢失密钥将导致已保存的 Bot Token 无法解密。不要把密钥提交到 Git 或公开日志。

### Telegram Bot 运行要求

Bot 使用 Telegram `getUpdates` 长轮询接收群组事件。不要为同一个 Bot 同时配置 Telegram webhook；需要先移除旧 webhook，避免 Telegram 返回 polling conflict。将 Bot 添加到目标群组；需要同步成员变更时，应赋予 Bot 所需的群组管理员权限，并在 BotFather/Telegram 设置中允许接收相应更新。

### 镜像与服务器要求

- GitHub Actions 在云端完成构建、测试并发布镜像到 GHCR。
- 发布至少提供可跟踪的版本标签和 `latest` 标签；同时记录不可变镜像 digest，生产验收以 digest 为准。
- 服务器只需要 Docker Engine 与 Docker Compose 插件。
- 正式 Compose 文件必须使用预构建镜像的 `image:` 字段；**禁止出现 `build:`**。
- 用户只需配置必要的端口、域名和安全环境变量，然后执行 `docker compose pull` 与 `docker compose up -d`。
- 禁止要求服务器拉取源码、安装 Go/Garble/Node/pnpm 或执行源码编译、Docker build。
- 提供完整 Compose 文件、`.env.example`、一键安装和升级命令。升级只拉取新镜像并重建应用容器，PostgreSQL/Redis 数据卷保持不变。

## 2. 宝塔面板 + Garble Go 二进制安装

这是独立于 Docker 的另一种部署方式，面向使用宝塔面板和已有数据库服务的用户。此模式不要求 Docker，也不要求用户在服务器编译 Go 源码。

### 用户安装流程

1. 在宝塔面板文件管理中，将对应 Linux 架构的 Garble 混淆 Go 二进制上传到服务器。
2. 按发布说明赋予可执行权限，并通过宝塔面板启动程序/配置守护进程。
3. 浏览器访问程序提供的安装页面，进入网页安装向导。
4. 在向导中填写 PostgreSQL 数据库连接信息（主机、端口、数据库名、数据库用户名、数据库密码及必要的 SSL 参数）。
5. 设置后台管理员用户名和管理员密码。
6. 安装向导验证数据库连接、自动创建/升级所需表结构、初始化管理员和基础配置。
7. 安装完成后跳转到管理后台登录页，使用刚才设置的管理员账号密码登录。

安装向导必须提供清晰的连接错误提示、配置校验和安装状态提示。数据库迁移与管理员初始化必须具备事务/幂等保护；安装成功后应锁定首次安装入口，不能通过再次访问安装页覆盖现有管理员密码或重置数据库。

### Redis 依赖

如果应用运行需要 Redis，宝塔二进制模式应在安装向导或首次启动配置中支持填写 Redis 地址、端口、密码和 DB 编号；也可在部署说明中明确兼容的本机 Redis 安装方式。不得假定 Docker Compose 内置的 Redis 在宝塔二进制模式下自动存在。

### Garble 构建要求

- 正式 Release 二进制必须通过 **Garble 混淆构建**，不能把普通 `go build` 产物改名后当作正式交付。
- CI 使用固定版本的 Go 和 Garble，在受控构建环境中构建；记录 Go 版本、Garble 版本、构建参数、目标 OS/ARCH、Git commit 和版本号。
- 按支持的平台发布独立文件，文件名包含应用版本与 OS/ARCH；同时发布 SHA-256 校验和。
- 发布前执行单元测试、构建后启动/版本检查及安装向导 smoke test。
- 普通未混淆 Go 二进制可以用于本地开发或 CI 调试，但不得作为正式 Release 资产。

Garble 混淆不等于加密，也不能保证源码不可逆向；不得对外宣传为绝对防逆向。

## 3. 两种部署方式的边界

| 项目 | Docker Compose | 宝塔 + Garble 二进制 |
|---|---|---|
| 应用交付 | GHCR 预构建镜像 | 上传 Garble 混淆 Go 二进制 |
| 数据库 | Compose 内置 PostgreSQL | 安装向导填写已有 PostgreSQL 连接信息 |
| Redis | Compose 内置 Redis | 配置已有 Redis（若应用运行需要） |
| 初始化 | 首次启动自动迁移 | 网页安装向导自动迁移 |
| 管理员 | 默认 `admin / 123456` | 用户在安装向导自行设置 |
| 服务器编译 | 不需要 | 不需要 |
| 安装入口 | 启动后直接访问后台登录页 | 上传并启动二进制后访问网页安装页 |

两种方式都必须保证数据库迁移安全、数据持久化、升级说明清晰。Docker 的默认管理员凭据不得套用到宝塔安装向导；宝塔模式也不得绕过安装向导使用固定默认密码。

## 4. Release 资产与验收清单

每个正式版本应提供：

1. GHCR 应用镜像：版本标签、`latest` 标签和不可变 digest。
2. Docker Compose 文件（包含应用、PostgreSQL、Redis）及 `.env.example`。
3. Docker 一键安装/升级脚本或等价命令。
4. Garble 混淆 Go 二进制（按支持平台分别提供）。
5. 宝塔安装说明、网页安装向导和配置/守护进程说明。
6. 二进制及相关发布文件的 SHA-256 校验和。
7. 版本说明、配置变更、数据库迁移说明和回滚注意事项。

正式发布验收必须确认：

- [ ] 全新 Docker 部署自动启动 PostgreSQL、Redis 和应用，无需服务器编译。
- [ ] 首次 Docker 初始化可使用 `admin / 123456` 登录；重启/升级不会重置已修改的密码。
- [ ] PostgreSQL/Redis 数据卷持久化，健康检查和升级流程通过。
- [ ] Compose 只使用预构建 `image:`，不含 `build:`。
- [ ] 宝塔上传 Garble 二进制后能打开安装页面。
- [ ] 安装向导可验证 PostgreSQL 连接、执行迁移并创建用户设置的管理员账号。
- [ ] 安装完成后安装入口被锁定，不能重置管理员或覆盖配置。
- [ ] Garble 构建步骤真实执行；二进制平台、版本、commit 和 SHA-256 信息齐全。
- [ ] Docker 与宝塔两种方式分别完成全新安装及升级 smoke test。

## 5. 明确禁止

- 禁止把服务器现场编译作为正式部署方案。
- 禁止 Compose 正式交付文件包含 `build:`。
- 禁止要求 Docker 用户手工安装 PostgreSQL/Redis 或安装 Go/Node 工具链。
- 禁止将普通 Go 编译产物标记为 Garble 混淆版本。
- 禁止宝塔二进制模式跳过网页安装向导或使用固定管理员默认密码。
- 禁止在没有真实 CI、镜像 digest、安装向导及目标平台验证证据时宣称交付完成。
