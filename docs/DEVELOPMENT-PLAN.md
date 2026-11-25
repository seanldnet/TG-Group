# TG-Group 开发计划

> 本计划从需求基线拆解实施顺序。阶段状态以实际代码、构建和测试证据为准。后端使用 Go 标准库 net/http 与 pgx/v5；CI 镜像仓库等未确认事项不得擅自定案。

## 当前状态（2026-11-25）
- [x] 新建空仓库并确认默认分支为 main。
- [x] 建立产品需求基线：`docs/REQUIREMENTS.md`。
- [x] 建立前端工程骨架（React + TypeScript + Vite）基础文件。
- [x] 使用官方 shadcn/ui Button、Badge、Card 组件实现产品介绍首页；前端生产构建已通过。
- [x] 通过云端 Node/pnpm 环境执行 shadcn/ui 官方 CLI，生成 components.json 和 16 个官方组件源码，并提交 pnpm-lock.yaml。
- [x] 确认后端语言为 Go（Golang）。
- [x] Go 后端采用标准库 `net/http`，不引入第三方 Web 框架；建立基础服务入口、配置和健康检查。
- [x] 建立 PostgreSQL pgx 连接池、Ping、版本化迁移执行器与身份/租户基础 schema（尚未完成 Redis 和后台任务）。
- [x] 增加 PostgreSQL 16 / Redis 7 本地开发 Compose 配置与启动文档（Compose 本身尚未在当前执行环境启动验证；Redis 暂未接入 Go 服务）。
- [x] 完成邮箱注册/登录、bcrypt 密码哈希、30 天 Bearer 会话、当前用户查询和退出撤销 API；租户/用户状态参与会话鉴权。
- [ ] 补齐 refresh token、密码找回、邮箱验证、登录限流、Telegram ID 绑定、角色授权及跨租户资源访问测试。
- [x] 网站首页、注册页、登录页和账号工作台已连接第一版 Go 认证 API。
- [ ] Telegram 注册管理 Bot、refresh token、密码找回、邮箱验证、登录限流和 Telegram ID 绑定。
- [x] 完成 Telegram Bot Token 验证、AES-256-GCM 加密存储、租户范围查询、启停、同 Bot 身份 Token 轮换与解绑 API。
- [x] 完成网站 Bot 管理页面：列表、添加、启停、轮换、解绑，使用官方 shadcn/ui 组件（前端 CI build 已通过）。
- [ ] 建立 Telegram Bot 长轮询/Webhook 运行时、群组自动发现和 Bot 管理员权限检测。
- [ ] 建立统一规则/权限/审计模型。
- [ ] 按需求矩阵逐项实现群管功能，并完成网站与 Telegram 双端配置。
- [ ] 接入 USDT、EPUSDT、GMPay 支付适配器并完成沙箱/真实联调。
- [ ] 接入 CAP 官方新版并完成端到端验证。
- [ ] 实现赞助商管理与双端展示/统计。
- [ ] 实现用户自配置 AI Provider、模型发现、审核管线和成本记录。
- [ ] Go/PostgreSQL 注册会话集成测试与前端生产构建已纳入 CI；完整业务测试和双端验收仍待完成。
- [ ] 建立云端 CI 构建与预构建镜像发布。
- [ ] 完成 Docker Compose 一键部署、自动端口分配和首次初始化。

## 阶段 0：规范与工程底座
1. 需求基线、开发计划、开发日志。
2. React + TypeScript 前端工程。
3. shadcn/ui 官方 CLI 初始化；页面 UI 仅使用 shadcn/ui 官方组件。
4. 代码格式、类型检查、测试和 CI 基础。
5. Go 后端使用标准库 `net/http` 建立服务入口、配置、结构化日志、优雅停机和健康检查。
6. PostgreSQL pgx 连接池与版本化迁移执行器。

**验收**：干净环境可安装依赖；前端构建通过；官方 shadcn/ui 组件可正常渲染；无其他 UI 组件库混入。

## 阶段 1：身份、租户与账号入口
- 用户注册、登录、密码重置、首次修改密码。
- Telegram ID 唯一绑定及解除/重新绑定规则。
- Telegram 注册管理 Bot 的注册对话流程。
- 两种入口映射同一用户主体。
- 租户隔离、审计和安全测试。

**验收**：网站注册和 Telegram 注册用户可在两端登录/识别同一账号；唯一性约束和跨租户访问测试通过。

## 阶段 2：套餐、订单和支付
- 套餐与授权模型。
- USDT、EPUSDT、GMPay provider adapter。
- 订单状态机、签名验证、幂等、对账、自动开通。
- 真实支付协议逐项核对。

**验收**：每个渠道完成创建订单、支付确认、重复通知、错误金额/签名和超时测试。

## 阶段 3：Bot 与群组资产
- 多 Bot Token 管理和验证。
- Bot 到群组的绑定关系。
- 群权限检测、Bot 状态、Webhook/Polling 策略。
- 多群隔离和配置模板。

**验收**：一个用户多 Bot、多群工作；其他用户无法访问其资源；权限不足有明确反馈。

## 阶段 4：群管核心与双端交互
- 规则引擎、权限模型、执行队列和审计日志。
- 按需求矩阵逐项实现每个群管功能。
- 网站端和 Telegram Inline Keyboard 端均可完整配置。
- 双端共享配置和冲突控制。

**验收**：逐项执行功能矩阵测试；每项功能两端配置与执行结果一致。

## 阶段 5：CAP、赞助商与 AI
- CAP 官方新版接入。
- 赞助商平台级/群级配置、展示和统计。
- 用户自助 AI Provider 管理、模型发现、内容审核、人工复核和成本记录。

**验收**：CAP 端到端验证；赞助商双端操作；AI 文本/图片/文档按模型能力测试，并验证失败保护。

## 阶段 6：交付与运维
- 监控、日志、备份、迁移和健康检查。
- 云端 CI 构建多架构或目标架构镜像。
- Docker Compose 一键启动 PostgreSQL、Redis 和应用。
- 自动分配并持久化五位数宿主机端口。
- 首次初始化管理员和强制改密。

**验收**：全新服务器仅安装 Docker 后可拉取镜像启动；服务器无源码编译；重启端口稳定；备份恢复演练通过。

## 当前待确认/风险
- 后端语言已确认 Go；当前 HTTP 层使用 Go 标准库，不引入第三方 Web 框架；数据访问和队列方案仍待实现。
- CI 平台与容器镜像仓库尚未确认。
- USDT 链、确认数、费率、退款规则尚未确认。
- 支付服务商接口地址/协议版本/沙箱账号需后续提供或核实。
- Telegram Bot API 对部分管理能力有固有限制，需按具体功能核实。
- AI 服务商模型发现与视觉/文件能力存在差异，必须 provider-specific 验证。


### Telegram Bot 运行时当前边界
- 当前服务启动时会为平台状态为 active 的 Bot 启动独立长轮询 worker，并将 Telegram 更新中的群组/超级群/频道信息写入 telegram_groups。
- Bot 管理状态和 Telegram 真实权限状态仍是两回事；需要后续调用 getChatMember/getChatAdministrators 等接口完成权限检测。
- Telegram 同一 Bot 的 getUpdates 与 webhook 不能同时工作；当前未实现 webhook 切换与 webhook 清理流程。
- 群组自动发现需要真实 Bot Token、Bot 加入目标群组以及 Telegram 实际投递相应更新；CI 单元测试不能替代真实 Bot 联调。
