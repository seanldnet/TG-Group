# TG Group Manager — Project Status

- 文档版本：0.1.5
- 更新日期：2026-11-21
- 仓库：seanldnet/TG-Group
- 阶段开发分支：feat/phase-1-vben-v5
- 阶段一 PR：#1（Draft）
- 当前阶段：阶段一（Vben Admin V5 单用户多机器人管理 UI）
- 阶段状态：UI 骨架和架构调整已提交；架构调整后的 CI 与视觉验收仍待完成

## 产品架构基线

- 单一平台所有者统一管理多个 Telegram Bot。
- 不提供多商户注册、多租户隔离、商户套餐或代理销售体系。
- Telegram 群成员仍是正常业务管理对象。
- 每个 Bot 可独立关联群组并配置群管规则；平台支付和安全配置统一管理。

## 最终交付与部署基线（强制）

用户明确要求两种互相独立的免编译部署方式，均为正式交付验收条件。

### A. Docker Compose 一键部署（内置数据库）

- Compose 必须同时编排 TG Group 应用、PostgreSQL 和 Redis；数据库及 Redis 使用持久化 volumes。
- GHCR 提供云端预构建应用镜像；Compose 正式文件只允许 `image:`，禁止 `build:`。
- 服务器只需 Docker Engine 与 Compose 插件，执行镜像 pull 和 compose up；不得下载源码或安装编译工具链。
- 全新数据库首次启动自动迁移并初始化默认管理员：用户名 `admin`，密码 `123456`，部署后打开后台即可登录。
- 默认管理员只在无管理员记录的全新数据库中创建；升级、重启、重复初始化不得覆盖已经修改的管理员密码。
- 提供健康检查、自动迁移、持久化、升级/回滚说明及一键命令。

### B. 宝塔面板 + Garble Go 二进制（网页安装向导）

- Release 提供对应 Linux OS/ARCH 的 Garble 混淆 Go 二进制；用户通过宝塔文件管理上传并启动，不在服务器编译。
- 启动后通过浏览器访问内置网页安装向导。
- 向导要求填写 PostgreSQL 主机、端口、数据库名、用户名、密码及必要连接参数，并验证连接。
- 用户在向导中自行设置管理员用户名和密码；向导自动执行数据库迁移、创建管理员和初始化基础配置。
- 安装成功后锁定安装入口，重复访问不得覆盖管理员密码或重置数据库。
- 若应用运行依赖 Redis，宝塔模式需允许填写 Redis 连接信息或按文档配置已有 Redis。

### Garble 构建约束

- 正式 Go 二进制必须由固定版本 Garble 构建；普通 `go build` 产物不能作为正式 Release。
- CI 记录 Go/Garble 版本、参数、commit、目标 OS/ARCH，发布 SHA-256，并运行安装向导 smoke test。
- Garble 是混淆，不是加密，不得宣传为绝对防逆向。

### 发布验收门槛

- [ ] Docker 首次部署自动启动应用、PostgreSQL、Redis。
- [ ] Docker 首次登录 `admin / 123456` 成功；升级/重启不重置密码。
- [ ] Docker Compose 无 `build:`，镜像 digest 可核验。
- [ ] PostgreSQL/Redis 数据持久化，健康检查、迁移和升级验证通过。
- [ ] 宝塔上传 Garble 二进制后可打开网页安装向导。
- [ ] 安装向导成功配置数据库、创建用户指定的管理员并锁定安装入口。
- [ ] Garble 多平台构建、SHA-256 和 smoke test 通过。
- [ ] 两种部署方式均提供清晰的安装、升级和回滚文档。

详细规范见 `docs/DEPLOYMENT.md`。文档已记录要求不代表部署实现已经完成；不得提前宣称正式交付。

## 当前阶段与技术基线

- 官方 Vue Vben Admin V5，固定上游 commit：`e726bc92780c1023c2393042871470d413ad79dc`。
- UI 变体：`apps/web-antd`；不使用 V2 API 或兼容层。
- 五个 UI 页面：运营总览、机器人管理、订单（按 Bot 筛选）、平台设置、统一支付配置。
- 当前页面仍是 UI 骨架；真实 Bot API、业务数据库、支付、安装向导、Docker 发布和 Garble 构建尚未完成。
- PR #1 保持 Draft；架构调整后的 TypeScript/生产构建和视觉验收待完成。
