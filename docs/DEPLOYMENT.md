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

### 镜像构建流水线

仓库根目录的 `Dockerfile` 是正式镜像构建入口：GitHub Actions 在云端先构建 Vben V5 前端静态资源，再编译 Go 服务并把前端资源复制进最终镜像。应用通过同一端口提供管理后台静态页面、`/api/*` API 和健康检查。

- Pull Request：执行多架构镜像构建验证，不推送镜像。
- 推送到 `main`：发布 GHCR `latest`、分支标签和 SHA 标签。
- 推送 `v*` 标签：发布对应版本标签和 SHA 标签。
- 正式发布记录 `linux/amd64`、`linux/arm64` 镜像 digest；生产验收以 digest 为准。
- Compose 只引用 GHCR 镜像，不引用 Dockerfile，也不包含 `build:`。

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
2. 为运行用户创建仅其可读写的配置目录，例如 `/var/lib/tg-group`；程序默认配置文件为 `/var/lib/tg-group/config.json`，也可通过 `APP_CONFIG_FILE` 指定其他路径。
3. 按发布说明赋予可执行权限，并通过宝塔面板启动程序/配置守护进程。首次启动时若没有 `DATABASE_URL` 环境变量且配置文件不存在，程序只开放安装页面，不启动 Bot worker 或管理 API；同时生成 256-bit 随机安装令牌，保存在 `config.json.install-token`（权限 0600），并写入宝塔服务日志。
4. 浏览器访问服务地址，进入中文网页安装向导。
5. 填写 PostgreSQL 主机、端口、数据库名、用户名、密码和 SSL 模式；填写 Redis 地址（必须包含端口）、密码和 DB 编号。
6. 点击“测试数据库连接”验证 PostgreSQL 与 Redis；完成安装时服务端会再次验证连接。
7. 设置平台 Owner 用户名和密码（12–72 bytes）。服务端执行嵌入式 migration，以 bcrypt hash 保存 Owner 密码；密码不会写入配置文件。
8. 安装成功后配置文件以原子方式写入，权限为 0600。页面提示通过宝塔面板重启 TG Group 服务；重启后程序加载配置并进入管理后台模式。

安装页面的连接测试和完成 API 均要求输入首次启动生成的安装令牌（通过 `X-Install-Token` 请求头传递）；请从宝塔服务日志中获取令牌，不要公开分享日志。安装向导会拒绝覆盖已存在的配置文件。若数据库中已有 Owner，必须输入匹配的用户名和密码才能继续绑定该数据库；不会重置或覆盖现有 Owner。若迁移或连接失败，页面返回错误并允许修复连接后重试。安装成功后安装页面和安装 API 均 fail closed。

### Redis 依赖

如果应用运行需要 Redis，宝塔二进制模式应在安装向导或首次启动配置中支持填写 Redis 地址、端口、密码和 DB 编号；也可在部署说明中明确兼容的本机 Redis 安装方式。不得假定 Docker Compose 内置的 Redis 在宝塔二进制模式下自动存在。

### Garble 构建要求

- 正式 Release 二进制必须通过 **Garble 混淆构建**，不能把普通 `go build` 产物改名后当作正式交付。
- CI 使用固定版本的 Go 和 Garble，在受控构建环境中构建；记录 Go 版本、Garble 版本、构建参数、目标 OS/ARCH、Git commit 和版本号。
- 按支持的平台发布独立文件，文件名包含应用版本与 OS/ARCH；同时发布 SHA-256 校验和。
- 发布前执行 Go 单元测试/vet、构建后版本检查及目标架构 QEMU smoke test；网页安装向导完成后，再将安装向导 smoke test 纳入正式发布门槛。
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

## 6. Docker 一键安装、升级与回滚

### 首次安装

服务器需预先安装并启动 Docker Engine 与 Docker Compose v2 插件。若 GHCR package 为私有包，先使用具备 read:packages 权限的 GitHub Token 登录：

```bash
echo "$GHCR_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

一键安装（默认安装到 /opt/tg-group，默认从 main 下载 Compose 文件）：

```bash
curl -fsSL https://raw.githubusercontent.com/seanldnet/TG-Group/main/scripts/deploy/install-docker.sh | sudo bash
```

脚本检查 Docker daemon 和 Compose v2，下载并检查 image-only Compose 配置，首次生成权限为 0600 的 .env，并使用 OpenSSL 生成独立随机 PostgreSQL 密码、Redis 密码和 32 字节 Bot Token 加密密钥。默认首次 Owner 为 admin / 123456。登录后应立即修改默认密码。

可通过环境变量指定安装版本或目录：

```bash
curl -fsSL https://raw.githubusercontent.com/seanldnet/TG-Group/vX.Y.Z/scripts/deploy/install-docker.sh | sudo TG_GROUP_REF=vX.Y.Z TG_GROUP_DIR=/opt/tg-group bash
```

### 升级

再次执行同一安装脚本即可拉取所选 ref 对应 Compose 配置、拉取镜像并重建应用。若安装目录中已有 .env，脚本不会覆盖它；数据库密码、Redis 密码、Bot Token 加密密钥、管理员密码及当前 APP_IMAGE 均保持不变。PostgreSQL/Redis 数据卷不会因应用容器升级而删除。

### 固定 digest 回滚

生产环境应记录每次验收镜像的不可变 digest。回滚时编辑安装目录中的 .env，将 APP_IMAGE 固定到之前验收的 digest：

```dotenv
APP_IMAGE=ghcr.io/seanldnet/tg-group@sha256:替换为已验收的真实digest
```

然后执行：

```bash
cd /opt/tg-group
sudo docker compose --env-file .env -f compose.yaml pull app
sudo docker compose --env-file .env -f compose.yaml up -d app
sudo docker compose --env-file .env -f compose.yaml ps
```

不要猜测或手工拼接 digest；必须使用 GitHub Actions 发布摘要中的真实值。数据库迁移通常向前兼容性需单独确认，镜像回滚不等于数据库 schema 自动回滚。

### 安全与边界

- 安装脚本不会自动开放云防火墙端口；请按实际部署配置安全组和 HTTPS 反向代理。
- .env 包含数据库凭据和加密密钥，必须限制文件权限并纳入安全备份，禁止提交到 Git。
- 如果 GHCR package 为私有包，服务器必须先完成 GHCR 登录；脚本不会收集或保存 GitHub Token。
- PR 构建只验证镜像，不会发布镜像。只有合并到 main 后触发发布流水线，才可使用公开/授权的 GHCR 镜像。

### 当前流水线状态边界

- Garble 工作流位于独立开发分支，尚未以真实 CI 结果证明构建和 Release 成功；PR 阶段不会发布正式 Release。
- 当前 PR 只负责 Garble 构建和版本 smoke test，不包含网页安装向导。安装向导必须在后续阶段实现数据库连接测试、迁移、用户自设 Owner 和不可重复安装锁，不能以静态安装页面代替真实初始化。
- Garble 是混淆工具，不是加密工具；二进制仍需通过正式版本验收与安全审查。


### 当前宝塔安装向导实现状态（PR #14）

- 未配置数据库时进入独立安装模式；安装模式不注册登录、Bot 管理或 Bot polling handlers。
- 首次启动生成稳定的 256-bit 随机令牌，文件权限 0600；令牌通过服务日志提供，并用于安装 API constant-time 校验。安装成功后删除令牌文件。
- 配置文件默认位于 `/var/lib/tg-group/config.json`，通过 `APP_CONFIG_FILE` 可覆盖；服务启动时要求它是权限不宽于 0600 的普通文件。
- 安装完成后必须从宝塔守护进程管理界面重启服务。当前安装完成接口不会尝试自行重启进程。
- 数据库迁移与 Owner 初始化是幂等操作；如果安装后配置文件写入失败，使用相同 Owner 用户名和密码重试，不会重置已创建的密码。
- PR #14 的真实 PostgreSQL/Redis 集成验收尚待执行；不得仅凭单元测试宣称宝塔部署已完成。
