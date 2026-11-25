# TG-Group 开发日志

所有代码、配置、依赖、测试、修复和部署变更均须在此记录。只记录已实际发生并可验证的工作，不将计划标为完成。

## 2026-11-25

### 文档初始化
- **变更**：建立产品需求基线和开发计划。
- **新增文件**：
  - `docs/REQUIREMENTS.md`：产品定位、注册入口、账号唯一性、多租户、群管功能矩阵、双端等价要求、支付、CAP、赞助商、AI、shadcn/ui 强制规范、Docker 交付和验收纪律。
  - `docs/DEVELOPMENT-PLAN.md`：按工程底座、身份租户、支付、Bot/群组、群管双端、CAP/赞助商/AI、部署交付拆分阶段。
- **仓库**：`seanldnet/Tg-group`，默认分支 `main`。
- **提交**：`3dcd7d525a464bf81457a38750272853d087c1c6`（需求文档）。
- **测试**：已确认需求文档写入 GitHub 成功；尚未有应用代码，因此无构建/功能测试结果。
- **状态**：文档基线已创建；应用开发尚未开始。
- **待办**：写入开发计划与本日志；开始 React + TypeScript + shadcn/ui 工程底座。后端技术栈仍需确认，不在未确认情况下擅自选择。


### 前端工程底座（第一步）
- **变更**：向仓库写入 React + TypeScript + Vite 前端基础文件。
- **新增文件**：`package.json`、`index.html`、`tsconfig.json`、`tsconfig.app.json`、`tsconfig.node.json`、`vite.config.ts`、`src/main.tsx`、`src/App.tsx`、`src/index.css`、`.gitignore`、`README.md`。
- **UI 约束**：当前仅放置工程占位入口，没有自建业务 UI 控件；后续界面必须通过 shadcn/ui 官方 CLI 添加的官方组件组合实现。
- **官方依据**：shadcn/ui Vite 安装指南与 CLI 文档： https://ui.shadcn.com/docs/installation/vite 、https://ui.shadcn.com/docs/cli 。当前官方流程采用 `shadcn@latest` CLI。
- **测试**：尚未执行 `pnpm install`、shadcn CLI、typecheck 或 production build。尝试在执行环境克隆 GitHub 仓库时因 DNS/网络不可用失败，因此不能声称构建通过。
- **状态**：基础文件已写入 GitHub；shadcn/ui CLI 初始化和官方组件落盘仍待执行；后端尚未开始。
- **下一步**：在可联网的构建/开发环境运行官方 shadcn CLI，安装官方组件并完成构建验证；随后继续建立前端页面框架。后端技术栈需先确认。


### 需求条款澄清
- **变更**：将“首次登录强制修改密码”明确标记为安全建议/待项目所有者确认，避免把尚未明确确认的建议写成已定需求。
- **文件**：`docs/REQUIREMENTS.md`。
- **测试**：已通过 GitHub 文件读取/更新确认写入；未涉及代码构建。


### 后端语言确认
- **变更**：项目所有者明确指定后端使用 Go（Golang）。
- **文件**：同步更新 `docs/REQUIREMENTS.md`、`docs/DEVELOPMENT-PLAN.md` 与 `README.md`。
- **技术边界**：Go 作为后端语言已确定；HTTP 框架/路由库、模块边界及异步任务实现仍需结合 Telegram Bot、Webhook、队列、支付与 AI 适配需求评估，不在未评估前擅自固化。
- **测试**：本次仅更新项目文档，尚未创建 Go 源码、安装依赖或运行 Go 测试；不代表后端工程已搭建。
- **下一步**：完成 Go 后端架构选型并建立可运行工程骨架，再配置 PostgreSQL/Redis 连接、迁移和基础健康检查。


### Go 后端基础服务（第一批代码）
- **变更**：建立 Go 模块与最小可运行 HTTP 服务，使用 Go 标准库 `net/http`，未引入第三方 Go Web 框架。
- **新增文件**：
  - `go.mod`
  - `cmd/api/main.go`：服务启动、JSON 结构化日志、SIGINT/SIGTERM 监听及优雅停机。
  - `internal/config/config.go`：读取 HTTP 端口和运行环境，校验端口范围。
  - `internal/config/config_test.go`：默认配置与非法端口测试。
  - `internal/httpapi/handler.go`：`GET /healthz` 健康检查端点。
  - `internal/httpapi/handler_test.go`：健康检查响应和未知路由 404 测试。
- **运行配置**：默认监听 `:8080`；可通过 `TG_GROUP_HTTP_PORT` 与 `TG_GROUP_ENV` 配置。
- **验证**：在临时本地工作目录按仓库源码建立同一 Go module，执行 `gofmt -w .`、`go test ./...`、`go vet ./...`，均通过。配置包 2 项测试、HTTP API 包 2 项测试通过；main 包无独立测试。
- **边界**：本批仅完成服务底座和健康检查；尚未接入 PostgreSQL、Redis、认证、租户、Telegram Bot 或业务 API。临时目录验证不等于已在 GitHub checkout 中构建。
- **下一步**：建立数据库迁移与 PostgreSQL/Redis 连接层，再实现认证和租户模型；前端继续严格按 shadcn/ui 官方 CLI 生成组件。
