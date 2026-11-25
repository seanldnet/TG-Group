# TG-Group

Telegram 多用户、多 Bot、多群组管理平台。

## 技术约束

- Web：React + TypeScript + Vite。
- Backend：Go（Golang）；具体 HTTP 框架/路由方案待架构评估。
- UI：只允许使用 shadcn/ui 官方组件及其组合，不混用其他 UI 组件库。
- 数据：PostgreSQL + Redis。
- 部署：最终使用 Docker Compose，服务器只拉取云端预构建镜像，不在服务器编译源码。

## 本地前端工程

需要 Node.js 与 pnpm 仅用于开发环境：

```bash
pnpm install
pnpm dlx shadcn@latest init -t vite
pnpm dlx shadcn@latest add button card input label badge separator skeleton sonner dialog dropdown-menu form select switch table tabs textarea tooltip
pnpm dev
```

生产服务器不执行以上开发命令；生产部署使用云端 CI 构建并发布的容器镜像。

## 项目文档

- [产品需求基线](docs/REQUIREMENTS.md)
- [开发计划](docs/DEVELOPMENT-PLAN.md)
- [开发日志](docs/DEVELOPMENT-LOG.md)

后端语言已确认使用 Go；具体框架与基础设施库将在架构评估后确定。
