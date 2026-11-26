# TG-Group 开发计划

> 本计划从需求基线拆解实施顺序。阶段状态以实际代码、构建和测试证据为准。后端使用 Go 标准库 net/http 与 pgx/v5；CI 镜像仓库等未确认事项不得擅自定案。

## 当前状态（2026-11-26，审查后校准）
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
- [ ] 补齐 refresh token、密码找回、邮箱验证、登录限流、角色授权及跨租户资源访问测试。
- [ ] Telegram ID 绑定首批代码已写入：一次性挑战、Telegram 官方 Update 自动取 ID、全局唯一绑定、解除绑定与审计；Go/PostgreSQL 测试、go vet、前端构建 CI 已通过；待平台 Bot 真实配置验收。
- [x] 网站首页、注册页、登录页和账号工作台已连接第一版 Go 认证 API。
- [ ] Telegram 注册管理 Bot、refresh token、密码找回、邮箱验证、登录限流。
- [ ] Telegram 身份绑定已进入代码实现阶段；需要配置专用平台 Bot Token/username 后真实联调。
- [x] 完成 Telegram Bot Token 验证、AES-256-GCM 加密存储、租户范围查询、启停、同 Bot 身份 Token 轮换与解绑 API。
- [x] 完成网站 Bot 管理页面：列表、添加、启停、轮换、解绑，使用官方 shadcn/ui 组件（前端 CI build 已通过）。
- [x] 建立 Telegram Bot getUpdates 长轮询运行时、群组自动发现和网站群组列表；消息事件日志、群成员增量采集、实时 Bot 权限检测、手动成员处罚 API 与网站操作入口已实现。Webhook 模式仍待完成。
- [x] 建立租户隔离的群组规则配置数据模型、服务端配置校验和网站规则管理页。
- [x] 接入消息规则执行引擎：关键词/敏感词/反广告删除、反刷屏阈值删除、欢迎/离群通知、权限检查与执行事件幂等记录。
- [x] 增加规则执行记录查询 API 和 shadcn/ui 网站历史页面，按租户/群组隔离。
- [x] 实现入群数学题验证与可选频道关注验证：成员临时限制、本人 Inline Keyboard 验证、频道成员校验、超时处理和默认权限恢复。
- [ ] 扩展统一规则执行/权限/审计模型：chat_member 事件通知、自动处罚队列与重试策略。
- [ ] 按需求矩阵逐项实现群管功能，并完成网站与 Telegram 双端配置。
- [ ] 接入 USDT、EPUSDT、GMPay 支付适配器并完成沙箱/真实联调。
- [x] 接入官方 Cap Standalone siteverify 与网站注册/登录服务端校验，前端固定使用官方 cap-widget。
- [ ] 完成 CAP 公网部署、站点密钥配置、浏览器端与服务端一次性 token 端到端验收（数学题验证不等同于 CAP）。
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


## 最近增量：群消息日志
- [x] 监听群组 message、edited_message、channel_post 更新。
- [x] 记录发送者 Telegram ID、用户名、显示名称、消息类型、文本/Caption 和时间。
- [x] 新增租户隔离的 `GET /api/v1/groups/{groupID}/events`。
- [x] 网站群组列表提供消息日志入口，并新增日志表格页。
- [ ] 消息日志游标分页、搜索、导出和可配置保留期限。
- [ ] 群成员全量同步、成员处罚与 Bot 管理员权限检查及真实 Telegram 群组端到端验收。

## 最近增量：群成员基础数据
- [x] 订阅 Telegram `chat_member` 更新，并处理消息中的新成员加入/离群事件。
- [x] 新增按租户、Bot、群组关联的成员状态表和索引。
- [x] 新增 `GET /api/v1/groups/{groupID}/members`，限制每次最多 200 条，SQL 双重租户/群组过滤。
- [x] 网站群组列表增加成员入口，成员页展示身份、管理员权限快照、加入时间和最近活动。
- [x] Bot 群组权限实时检测（getChatMember）及手动禁言/解禁/封禁/解封 API 与网站操作入口。\n- [ ] 全量成员同步、自动化处罚规则、权限定时刷新和真实 Bot 群组联调。

## 最近增量：Bot 权限检测与成员处罚
- [x] 实时调用 Telegram `getChatMember` 检查 Bot 身份及管理权限。
- [x] 服务端强制验证 `can_restrict_members`，不依赖可能过期的数据库权限快照。
- [x] 新增手动封禁、解封、限时禁言、解除禁言 API；执行前核验目标成员身份，拒绝对管理员/群主处罚。
- [x] 处罚成功后更新本地成员状态，并写入 `audit_logs`。
- [x] 网站成员页展示实时 Bot 权限，并增加对应成员操作入口。
- [ ] 后续补充完整成员全量同步、处罚操作记录查询/撤销、定时任务、规则自动执行和真实 Telegram 测试群验收。


## 审查校准与后续开发基线（2026-11-26）

- 已复核 main 分支文件树、需求基线、开发计划、开发日志、API 文档、本地测试文档、README、首页文案、CI 工作流和最新 Actions 运行记录。
- 审查基线提交：`453151d85b8e8478cb1042b9bd7fde56841390e3`；CI run `45728546740` 的 Go/PostgreSQL 测试、go vet 与前端生产构建成功。
- 已有代码：网站邮箱认证、Bot Token 加密管理、Telegram 长轮询/群组发现、消息日志、成员增量采集与手动处罚、基础规则配置/执行/记录、数学题及频道关注入群验证、网站 CAP siteverify。
- 未验收：真实 Telegram 测试群端到端、CAP 公网端到端、网站与 Telegram 管理端双端等价。
- 未完成模块：可信 Telegram ID 绑定与注册管理 Bot、refresh token/密码找回/邮箱验证/登录限流/完整角色授权、完整群管矩阵与自动处罚队列、支付授权、赞助商、AI 审核、生产镜像与一键部署。

### 后续开发顺序
1. 账号体系补齐：可信 Telegram ID 绑定、refresh token、密码找回/邮箱验证方案、登录限流、角色授权与安全测试。涉及邮件服务商或平台注册 Bot 凭据的部分，在配置未提供前不宣称联调完成。
2. 群管理规则矩阵、自动处罚队列、重试和审计。
3. Telegram 注册/管理 Bot，与网站共用服务和权限，实现双端闭环。
4. 套餐、订单及 USDT/EPUSDT/GMPay 适配，按服务商官方协议核实。
5. 赞助商及用户自配置 AI Provider/审核。
6. CAP 公网验收、真实 Telegram 联调、业务测试。
7. 云端预构建镜像、生产 Compose、持久化五位数端口及初始化流程。


## 最近增量：Telegram 身份自动绑定（首批实现，待验证）
- 网站账号工作台增加 Telegram 身份绑定状态、绑定跳转和解除绑定入口；没有手动填写 Telegram ID 的输入项。
- 新增一次性绑定挑战表，挑战码随机生成、5 分钟有效、数据库只保存 SHA-256 摘要；生成新挑战会使旧挑战失效。
- 新增平台专用 Telegram 身份 Bot 长轮询进程，与用户自行接入的群管 Bot 完全分离。只接受私聊 `/start <challenge>`，并要求 Telegram Update 的 `from.id` 与私聊 `chat.id` 一致。
- Telegram ID、username、first_name、last_name 仅从 Telegram 官方 Update 读取；网站 API 不接收任何 Telegram ID 字段。
- 绑定事务执行挑战消费、users.telegram_id 全局唯一更新和审计日志；解除绑定清空 Telegram 资料、作废待处理挑战并写审计日志。
- 新增 API：`GET /api/v1/auth/telegram`、`POST /api/v1/auth/telegram/link`、`DELETE /api/v1/auth/telegram`。
- 新增配置：`TELEGRAM_AUTH_BOT_TOKEN` 与 `TELEGRAM_AUTH_BOT_USERNAME`，必须同时配置；Token 仅后端读取。
- 新增 PostgreSQL migration 000010（internal 与根目录 up/down 文件同步）、服务集成测试和配置测试；同步 API 与本地测试文档。
- **状态**：代码已提交 main；CI 已通过；尚未配置真实平台 Bot Token，因此不得标记真实 Telegram 联调完成。
