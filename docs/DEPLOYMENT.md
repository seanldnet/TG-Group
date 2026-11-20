# 交付与部署标准

本文档是 TG Group 后续开发、CI/CD、发布和生产交付的强制基线。任何阶段不得把“服务器现场编译”作为正式部署步骤。

## 1. Docker 一键部署（正式默认交付）

最终生产部署必须采用预构建 Docker 镜像：

- GitHub Actions 在云端完成构建、测试并发布镜像到 GHCR。
- 发布至少提供可跟踪的版本标签和 `latest` 标签；同时记录不可变镜像 digest，生产验收以 digest 为准，不能只依赖可变标签。
- 服务器只需要安装 Docker Engine 与 Docker Compose 插件，并提供部署配置及环境变量。
- 正式 `docker-compose.yml` / Compose 文件必须使用预构建镜像的 `image:` 字段；**禁止出现 `build:`**，禁止要求服务器拉取源码、安装 Node/Go 工具链或执行源码编译。
- 一键部署流程只执行镜像拉取和容器编排，例如 `docker compose pull` 后 `docker compose up -d`。不得在服务器执行 `go build`、`garble build`、`pnpm build` 或 Docker build。
- 数据库迁移和必要的初始化/seed 应由发布镜像内的启动入口或独立一次性 migration service 自动、安全地执行；必须保证重复启动幂等，并在迁移失败时明确失败，不得静默启动不兼容版本。
- Compose 应提供持久化数据卷、健康检查、合理的服务依赖和重启策略；升级通过拉取新镜像并重建容器完成，业务数据不得因升级丢失。
- 提供可直接复制执行的一键安装/升级命令和完整的 `.env.example`；不得要求用户手工进入容器编译或构建前端。

## 2. Go 独立二进制交付（必须 Garble 混淆）

除 Docker 镜像外，正式 Release 还必须提供独立 Go 可执行文件：

- Release 二进制必须通过 **Garble 混淆构建**，不能把普通 `go build` 产物改名后当作正式交付。
- CI 使用固定版本的 Go 和 Garble，在受控构建环境中构建；记录 Go 版本、Garble 版本、构建参数、目标 OS/ARCH、Git commit 和版本号。
- 构建流程应使用 Garble 的混淆构建能力（例如 `garble build`，并按项目需要启用字面量混淆等选项）；具体参数应经过兼容性测试后固定。
- 按支持的平台发布独立文件，文件名包含应用版本与 OS/ARCH；同时发布 SHA-256 校验和。
- 发布前执行单元测试、构建后启动/版本检查及必要的 smoke test；确保二进制可在目标系统运行。
- 普通未混淆 Go 二进制可以用于本地开发或 CI 调试，但不得作为正式 Release 资产，也不得替代 Garble 构建。

Garble 混淆不等于加密，也不能保证源码不可逆向；它是正式二进制的符号/代码混淆要求，不得对外宣传为绝对防逆向。

## 3. Release 资产与验收清单

每个正式版本应同时交付：

1. GHCR Docker 镜像：版本标签、`latest` 标签和不可变 digest。
2. Docker Compose 部署文件及 `.env.example`。
3. 一键安装/升级脚本或等价的明确命令。
4. Garble 混淆后的 Go 二进制（按支持平台分别提供）。
5. 二进制及相关发布文件的 SHA-256 校验和。
6. 版本说明、配置变更、数据库迁移说明和回滚注意事项。

发布验收必须确认：

- [ ] GitHub Actions 构建、测试和安全扫描通过。
- [ ] GHCR 镜像已推送，digest 可核验。
- [ ] Compose 使用 `image:`，没有 `build:`；在干净服务器上无需编译即可拉取并启动。
- [ ] 数据库迁移/初始化可自动执行且重复启动安全。
- [ ] Garble 构建步骤真实执行，Release 中不存在以普通 `go build` 产物冒充的正式二进制。
- [ ] 二进制平台、版本、commit 和 SHA-256 信息齐全。
- [ ] 完成 Docker 全新部署、升级和基本运行 smoke test。

## 4. 明确禁止

- 禁止把服务器现场编译作为正式部署方案。
- 禁止 Compose 正式交付文件包含 `build:`。
- 禁止要求服务器安装 Go、Garble、Node.js、pnpm 或下载源码后编译。
- 禁止只发布源码压缩包或普通 Go 二进制而缺少预构建 Docker 镜像。
- 禁止将普通 Go 编译产物标记为 Garble 混淆版本。
- 禁止在没有真实 CI、镜像 digest 和目标平台验证证据时宣称交付完成。
