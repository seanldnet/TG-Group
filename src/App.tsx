import { useEffect, type ReactNode } from "react"
import {
  ArrowRight,
  Bot,
  Crown,
  CreditCard,
  Wallet,
  ClipboardList,
  Fingerprint,
  Layers3,
  LockKeyhole,
  MessageSquareText,
  ShieldCheck,
  Sparkles,
  UsersRound,
  Workflow,
  type LucideIcon,
} from "lucide-react"

import DashboardLayout from "@/components/DashboardLayout"
import LoginPage from "@/pages/LoginPage"
import RegisterPage from "@/pages/RegisterPage"
import DashboardPage from "@/pages/DashboardPage"
import SecurityPage from "@/pages/SecurityPage"
import PlatformSettingsPage from "@/pages/PlatformSettingsPage"
import AdminUsersPage from "@/pages/AdminUsersPage"
import AccountLinksPage from "@/pages/AccountLinksPage"
import BotsPage from "@/pages/BotsPage"
import GroupsPage from "@/pages/GroupsPage"
import GroupEventsPage from "@/pages/GroupEventsPage"
import GroupMembersPage from "@/pages/GroupMembersPage"
import GroupMemberListsPage from "@/pages/GroupMemberListsPage"
import GroupRulesPage from "@/pages/GroupRulesPage"
import GroupRuleEventsPage from "@/pages/GroupRuleEventsPage"
import PersonalCenterPage from "@/pages/PersonalCenterPage"
import BusinessWorkspacePage from "@/pages/BusinessWorkspacePage"
import AgentCenterPage from "@/pages/AgentCenterPage"
import AdminAgentLevelsPage from "@/pages/AdminAgentLevelsPage"
import AdminPaymentsPage from "@/pages/AdminPaymentsPage"
import WalletPaymentsPage from "@/pages/WalletPaymentsPage"
import PlanCatalogPage from "@/pages/PlanCatalogPage"
import OrdersPage from "@/pages/OrdersPage"
import LicensesPage from "@/pages/LicensesPage"
import NotFoundPage from "@/pages/NotFoundPage"
import AdminOrdersPage from "@/pages/AdminOrdersPage"
import AdminLicensesPage from "@/pages/AdminLicensesPage"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

type Feature = {
  icon: LucideIcon
  title: string
  description: string
  label: string
}

const features: Feature[] = [
  {
    icon: Bot,
    title: "多 Bot 集中管理",
    description: "在一个控制台接入和管理多个 Telegram Bot，集中查看接入状态与群组归属。",
    label: "Bot 资产",
  },
  {
    icon: UsersRound,
    title: "多群组独立配置",
    description: "每个群组拥有独立规则与权限边界，也可以逐步使用模板完成批量配置。",
    label: "群组管理",
  },
  {
    icon: ShieldCheck,
    title: "入群验证与群安全",
    description: "围绕入群验证、反广告、反垃圾、成员处罚和群保护构建安全管理流程。",
    label: "安全防护",
  },
  {
    icon: Workflow,
    title: "规则自动化",
    description: "通过关键词、自动回复、媒体限制和阶梯处罚规则，减少重复的人工管理。",
    label: "自动化",
  },
  {
    icon: MessageSquareText,
    title: "网站与 Telegram 双端",
    description: "两个入口共享同一账号、权限与业务数据，群主可以按使用习惯选择管理方式。",
    label: "双端协同",
  },
  {
    icon: ClipboardList,
    title: "操作审计与追溯",
    description: "记录关键操作的执行人、目标和时间，让管理变更有迹可循。",
    label: "审计记录",
  },
  {
    icon: Crown,
    title: "代理等级与自助升级",
    description: "规划代理等级、权益与升级价格，用户可自主购买升级，支付确认后自动生效。",
    label: "代理业务",
  },
  {
    icon: Wallet,
    title: "余额、套餐与订单",
    description: "统一账户余额、资金流水、套餐目录、订单状态和授权记录，避免业务数据分散。",
    label: "账户业务",
  },
  {
    icon: CreditCard,
    title: "多渠道在线支付",
    description: "按服务商官方协议接入 USDT、EPUSDT 与 GMPay，执行签名核验、幂等确认和自动开通。",
    label: "支付业务",
  },
]

const principles = [
  {
    icon: LockKeyhole,
    title: "租户数据隔离",
    description: "账号只能访问授权范围内的 Bot、群组和业务数据，权限在后端统一校验。",
  },
  {
    icon: Fingerprint,
    title: "统一身份体系",
    description: "网站账号与 Telegram 注册入口共享用户主体，避免两套账号体系相互割裂。",
  },
  {
    icon: Layers3,
    title: "模块化持续扩展",
    description: "以统一的数据模型和规则引擎为基础，逐步扩展支付、AI 审核与运营能力。",
  },
]

function FeatureCard({ feature }: { feature: Feature }) {
  const Icon = feature.icon

  return (
    <Card className="h-full border-border/70 bg-card/90 shadow-sm transition-all duration-200 hover:-translate-y-1 hover:shadow-md">
      <CardHeader>
        <div className="mb-3 flex size-11 items-center justify-center rounded-xl bg-primary/8 text-primary">
          <Icon className="size-5" />
        </div>
        <div className="flex items-center justify-between gap-3">
          <CardTitle className="text-base">{feature.title}</CardTitle>
          <Badge variant="secondary">{feature.label}</Badge>
        </div>
        <CardDescription className="pt-1 leading-6">
          {feature.description}
        </CardDescription>
      </CardHeader>
    </Card>
  )
}

function HomePage() {
  return (
    <main className="min-h-svh overflow-hidden bg-background text-foreground">
      <header className="sticky top-0 z-20 border-b border-border/70 bg-background/90 backdrop-blur-xl">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-5 sm:px-8">
          <a href="#top" className="flex items-center gap-3" aria-label="TG-Group 首页">
            <span className="flex size-9 items-center justify-center rounded-xl bg-primary text-primary-foreground">
              <ShieldCheck className="size-5" />
            </span>
            <span>
              <span className="block text-base font-semibold tracking-tight">TG-Group</span>
              <span className="hidden text-[11px] text-muted-foreground sm:block">
                Telegram 社群管理平台
              </span>
            </span>
          </a>

          <nav className="hidden items-center gap-7 text-sm text-muted-foreground md:flex">
            <a className="transition-colors hover:text-foreground" href="#features">功能介绍</a>
            <a className="transition-colors hover:text-foreground" href="#principles">产品原则</a>
            <a className="transition-colors hover:text-foreground" href="#roadmap">开发进度</a>
          </nav>

          <div className="flex items-center gap-2">
            <Button asChild variant="ghost" size="sm">
              <a href="/login">登录</a>
            </Button>
            <Button asChild size="sm">
              <a href="/register">注册账号</a>
            </Button>
          </div>
        </div>
      </header>

      <section id="top" className="relative isolate">
        <div className="pointer-events-none absolute inset-0 -z-10 bg-[radial-gradient(ellipse_at_75%_15%,rgba(59,130,246,0.13),transparent_42%),linear-gradient(to_bottom,var(--background),var(--background))]" />
        <div className="mx-auto grid max-w-7xl items-center gap-14 px-5 pb-20 pt-20 sm:px-8 lg:grid-cols-[1.08fr_0.92fr] lg:gap-20 lg:pb-28 lg:pt-28">
          <div>
            <Badge variant="outline" className="mb-6 gap-2 rounded-full px-3 py-1.5 text-xs">
              <span className="size-1.5 rounded-full bg-emerald-500" />
              面向 Telegram 社群运营
            </Badge>
            <h1 className="max-w-3xl text-4xl leading-[1.12] font-semibold tracking-tight sm:text-5xl lg:text-6xl">
              一个控制台，
              <span className="block text-primary">管理所有 Telegram 社群</span>
            </h1>
            <p className="mt-7 max-w-2xl text-base leading-8 text-muted-foreground sm:text-lg">
              TG-Group 致力于把多个管理 Bot、多个群组和日常安全规则整合到同一个平台。
              网站控制台与 Telegram 管理入口共享统一账号和配置，让社群管理更清晰、更高效。
            </p>
            <div className="mt-9 flex flex-wrap gap-3">
              <Button asChild size="lg">
                <a href="/register">
                  创建账号
                  <ArrowRight className="ml-2 size-4" />
                </a>
              </Button>
              <Button asChild variant="outline" size="lg">
                <a href="#features">了解功能</a>
              </Button>
            </div>
            <div className="mt-10 flex flex-wrap items-center gap-x-7 gap-y-3 text-sm text-muted-foreground">
              <span className="inline-flex items-center gap-2">
                <ShieldCheck className="size-4 text-primary" />
                多租户隔离设计
              </span>
              <span className="inline-flex items-center gap-2">
                <Workflow className="size-4 text-primary" />
                统一规则管理
              </span>
            </div>
          </div>

          <div className="relative mx-auto w-full max-w-xl">
            <div className="absolute -inset-5 -z-10 rounded-[2rem] bg-primary/5 blur-2xl" />
            <Card className="overflow-hidden border-border/80 bg-card shadow-2xl shadow-primary/10">
              <div className="flex items-center justify-between border-b border-border px-5 py-4">
                <div className="flex items-center gap-3">
                  <div className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                    <Layers3 className="size-5" />
                  </div>
                  <div>
                    <p className="text-sm font-semibold">社群管理总览</p>
                    <p className="text-xs text-muted-foreground">产品界面示意</p>
                  </div>
                </div>
                <Badge variant="secondary">建设中</Badge>
              </div>
              <CardContent className="space-y-5 p-5 sm:p-6">
                <div className="grid grid-cols-2 gap-3">
                  <div className="rounded-xl border border-border bg-muted/35 p-4">
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Bot className="size-4" />
                      管理 Bot
                    </div>
                    <p className="mt-3 text-2xl font-semibold tracking-tight">多 Bot</p>
                    <p className="mt-1 text-xs text-muted-foreground">统一接入与状态管理</p>
                  </div>
                  <div className="rounded-xl border border-border bg-muted/35 p-4">
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <UsersRound className="size-4" />
                      Telegram 群组
                    </div>
                    <p className="mt-3 text-2xl font-semibold tracking-tight">多群组</p>
                    <p className="mt-1 text-xs text-muted-foreground">规则独立、权限隔离</p>
                  </div>
                </div>

                <div className="space-y-3">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-semibold">平台能力规划</p>
                    <span className="text-xs text-muted-foreground">模块示意</span>
                  </div>
                  {[
                    { icon: ShieldCheck, title: "入群验证与群安全", detail: "验证、反垃圾、成员处罚" },
                    { icon: Workflow, title: "自动化规则", detail: "关键词、自动回复、媒体限制" },
                    { icon: ClipboardList, title: "操作审计", detail: "关键变更记录与追踪" },
                  ].map((item) => {
                    const Icon = item.icon
                    return (
                      <div key={item.title} className="flex items-center gap-3 rounded-xl border border-border/70 p-3">
                        <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/8 text-primary">
                          <Icon className="size-4" />
                        </span>
                        <div className="min-w-0 flex-1">
                          <p className="text-sm font-medium">{item.title}</p>
                          <p className="mt-0.5 text-xs text-muted-foreground">{item.detail}</p>
                        </div>
                        <Sparkles className="size-4 shrink-0 text-muted-foreground/60" />
                      </div>
                    )
                  })}
                </div>
                <p className="rounded-lg bg-muted/50 px-3 py-2 text-xs leading-5 text-muted-foreground">
                  说明：此区域为产品概念示意，不代表当前已经接入真实 Bot 或群组数据。
                </p>
              </CardContent>
            </Card>
          </div>
        </div>
      </section>

      <section id="features" className="border-y border-border/70 bg-muted/25 py-20 sm:py-24">
        <div className="mx-auto max-w-7xl px-5 sm:px-8">
          <div className="mx-auto mb-12 max-w-2xl text-center">
            <Badge variant="outline" className="mb-4">核心能力</Badge>
            <h2 className="text-3xl font-semibold tracking-tight sm:text-4xl">
              社群运营与账户业务，统一管理
            </h2>
            <p className="mt-4 text-sm leading-7 text-muted-foreground sm:text-base">
              围绕 Bot 管理、代理体系、套餐支付和授权管理建设完整网站。所有模块按真实接口逐步交付，未接通的业务不会伪装成已上线功能。
            </p>
          </div>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {features.map((feature) => (
              <FeatureCard key={feature.title} feature={feature} />
            ))}
          </div>
          <p className="mt-6 text-center text-xs text-muted-foreground">
            以上包含已建立的基础能力与正在建设的业务规划；代理升级、余额、套餐和支付等模块完成真实后端闭环前，不视为可用服务。
          </p>
        </div>
      </section>

      <section id="principles" className="py-20 sm:py-24">
        <div className="mx-auto max-w-7xl px-5 sm:px-8">
          <div className="grid gap-12 lg:grid-cols-[0.8fr_1.2fr] lg:items-start">
            <div>
              <Badge variant="outline" className="mb-4">产品原则</Badge>
              <h2 className="text-3xl font-semibold tracking-tight sm:text-4xl">
                让管理边界清楚，
                <span className="block text-primary">让每次操作可追溯</span>
              </h2>
              <p className="mt-5 max-w-lg text-sm leading-7 text-muted-foreground sm:text-base">
                平台从身份、权限和数据隔离开始建设。无论通过网站还是 Telegram 管理，
                都由同一套后端规则控制访问范围和操作结果。
              </p>
            </div>
            <div className="grid gap-4 sm:grid-cols-3">
              {principles.map((item) => {
                const Icon = item.icon
                return (
                  <Card key={item.title} className="h-full border-border/70 shadow-sm">
                    <CardHeader>
                      <div className="mb-3 flex size-10 items-center justify-center rounded-xl bg-primary/8 text-primary">
                        <Icon className="size-5" />
                      </div>
                      <CardTitle className="text-base">{item.title}</CardTitle>
                      <CardDescription className="pt-1 leading-6">
                        {item.description}
                      </CardDescription>
                    </CardHeader>
                  </Card>
                )
              })}
            </div>
          </div>
        </div>
      </section>

      <section id="roadmap" className="pb-20 sm:pb-24">
        <div className="mx-auto max-w-7xl px-5 sm:px-8">
          <Card className="overflow-hidden border-primary/15 bg-primary/[0.035] shadow-none">
            <CardContent className="flex flex-col gap-7 p-7 sm:p-10 md:flex-row md:items-center md:justify-between">
              <div className="max-w-2xl">
                <Badge variant="secondary" className="mb-4">项目建设中</Badge>
                <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">
                  网站业务优先，逐步交付完整业务闭环
                </h2>
                <p className="mt-3 text-sm leading-7 text-muted-foreground">
                  当前已建立 Go 服务、PostgreSQL 身份数据模型和第一版注册登录 API，
                  正在完善前端组件体系与后续业务模块。尚未上线的能力不会作为已完成服务对外承诺。
                </p>
              </div>
              <Button asChild variant="outline" size="lg" className="shrink-0">
                <a href="https://github.com/seanldnet/Tg-group" target="_blank" rel="noreferrer">
                  查看 GitHub 仓库
                  <ArrowRight className="ml-2 size-4" />
                </a>
              </Button>
            </CardContent>
          </Card>
        </div>
      </section>

      <footer className="border-t border-border/70 py-7">
        <div className="mx-auto flex max-w-7xl flex-col gap-3 px-5 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:px-8">
          <p>© {new Date().getFullYear()} TG-Group. Telegram 社群管理平台。</p>
          <p>统一管理 · 权限隔离 · 安全可追溯</p>
        </div>
      </footer>
    </main>
  )
}


export default function App() {
  const pathname = window.location.pathname
  useEffect(() => {
    const titles: Record<string, string> = {
      "/": "TG-Group｜Telegram 社群管理平台",
      "/login": "登录｜TG-Group",
      "/register": "注册账号｜TG-Group",
      "/dashboard": "管理中心｜TG-Group",
      "/dashboard/payments": "USDT 充值与支付｜TG-Group",
    }
    document.title = titles[pathname] || (pathname.startsWith("/dashboard/") ? "业务工作台｜TG-Group" : "页面不存在｜TG-Group")
  }, [pathname])
  if (pathname === "/login") return <LoginPage />
  if (pathname === "/register") return <RegisterPage />
  let page: ReactNode = null
  if (pathname === "/dashboard") page = <DashboardPage />
  else if (pathname === "/dashboard/platform") page = <PlatformSettingsPage />
  else if (pathname === "/dashboard/admin/users") page = <AdminUsersPage />
  else if (pathname === "/dashboard/admin/agent-levels") page = <AdminAgentLevelsPage />
  else if (pathname === "/dashboard/admin/payments") page = <AdminPaymentsPage />
  else if (pathname === "/dashboard/payments") page = <WalletPaymentsPage />
  else if (pathname === "/dashboard/plans") page = <PlanCatalogPage />
  else if (pathname === "/dashboard/orders") page = <OrdersPage />
  else if (pathname === "/dashboard/licenses") page = <LicensesPage />
  else if (pathname === "/dashboard/admin/plans") page = <PlanCatalogPage admin />
  else if (pathname === "/dashboard/admin/orders") page = <AdminOrdersPage />
  else if (pathname === "/dashboard/admin/licenses") page = <AdminLicensesPage />
  else if (pathname === "/dashboard/account/profile") page = <PersonalCenterPage />
  else if (pathname === "/dashboard/account/agent") page = <AgentCenterPage />
  else if (pathname === "/dashboard/account/links") page = <AccountLinksPage />
  else if (pathname === "/dashboard/security/password") page = <SecurityPage section="password" />
  else if (pathname === "/dashboard/security/sessions") page = <SecurityPage section="sessions" />
  else if (pathname === "/dashboard/security/logs") page = <SecurityPage section="logs" />
  else if (pathname === "/dashboard/bots") page = <BotsPage />
  else if (pathname === "/dashboard/groups") page = <GroupsPage />
  else {
    const groupEventsMatch = pathname.match(/^\/dashboard\/groups\/([0-9a-f-]+)\/events$/i)
    const groupMembersMatch = pathname.match(/^\/dashboard\/groups\/([0-9a-f-]+)\/members$/i)
    const groupMemberListsMatch = pathname.match(/^\/dashboard\/groups\/([0-9a-f-]+)\/member-lists$/i)
    const groupRulesMatch = pathname.match(/^\/dashboard\/groups\/([0-9a-f-]+)\/rules$/i)
    const groupRuleEventsMatch = pathname.match(/^\/dashboard\/groups\/([0-9a-f-]+)\/rule-events$/i)
    if (groupEventsMatch) page = <GroupEventsPage groupID={groupEventsMatch[1]} />
    else if (groupMembersMatch) page = <GroupMembersPage groupID={groupMembersMatch[1]} />
    else if (groupMemberListsMatch) page = <GroupMemberListsPage groupID={groupMemberListsMatch[1]} />
    else if (groupRuleEventsMatch) page = <GroupRuleEventsPage groupID={groupRuleEventsMatch[1]} />
    else if (groupRulesMatch) page = <GroupRulesPage groupID={groupRulesMatch[1]} />
    else if (pathname.startsWith("/dashboard/")) page = <BusinessWorkspacePage pathname={pathname} />
  }
  if (page) return <DashboardLayout>{page}</DashboardLayout>
  if (pathname === "/") return <HomePage />
  return <NotFoundPage />
}