import { useEffect, useState, type ReactNode } from "react"
import {
  Activity, Bot, ChevronRight, ClipboardList, CreditCard, Crown, FileText,
  LayoutDashboard, Link2, ListChecks, LogOut, Menu, Network, Package,
  Receipt, Settings, ShieldCheck, Sparkles, UsersRound,
  Wallet, Webhook, X, UserRound, ChartNoAxesCombined, Handshake, type LucideIcon,
} from "lucide-react"
import { clearToken, currentUser, logout, type User } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"

type NavItem = { href: string; label: string; icon: LucideIcon }
const workspace: NavItem[] = [
  { href: "/dashboard", label: "总览", icon: LayoutDashboard },
  { href: "/dashboard/bots", label: "机器人管理", icon: Bot },
  { href: "/dashboard/groups", label: "群组管理", icon: UsersRound },
  { href: "/dashboard/bot-sponsors", label: "机器人赞助商", icon: Handshake },
]
const account: NavItem[] = [
  { href: "/dashboard/account/profile", label: "个人中心", icon: UserRound },
  { href: "/dashboard/account/links", label: "账号关联", icon: Link2 },
  { href: "/dashboard/security/password", label: "修改密码", icon: ShieldCheck },
  { href: "/dashboard/security/sessions", label: "登录设备", icon: Activity },
  { href: "/dashboard/security/logs", label: "安全日志", icon: ClipboardList },
]
const business: NavItem[] = [
  { href: "/dashboard/agents", label: "代理管理", icon: Network },
  { href: "/dashboard/agents/levels", label: "代理等级", icon: Crown },
  { href: "/dashboard/agents/commissions", label: "佣金管理", icon: Wallet },
  { href: "/dashboard/agents/api-keys", label: "代理 API", icon: Webhook },
  { href: "/dashboard/plans", label: "套餐管理", icon: Package },
  { href: "/dashboard/orders", label: "订单管理", icon: Receipt },
  { href: "/dashboard/payments", label: "支付管理", icon: CreditCard },
  { href: "/dashboard/licenses", label: "授权管理", icon: ShieldCheck },
  { href: "/dashboard/finance", label: "财务概览", icon: FileText },
  { href: "/dashboard/ai", label: "AI 智能审核", icon: Sparkles },
  { href: "/dashboard/tasks", label: "定时任务", icon: ListChecks },
  { href: "/dashboard/analytics", label: "数据统计", icon: ChartNoAxesCombined },
  { href: "/dashboard/audit", label: "操作审计", icon: ClipboardList },
]
const administration: NavItem[] = [
  { href: "/dashboard/admin/users", label: "用户管理", icon: UsersRound },
  { href: "/dashboard/admin/agents", label: "代理审核", icon: Network },
  { href: "/dashboard/admin/plans", label: "平台套餐", icon: Package },
  { href: "/dashboard/admin/orders", label: "平台订单", icon: Receipt },
  { href: "/dashboard/admin/payments", label: "支付渠道", icon: CreditCard },
  { href: "/dashboard/admin/licenses", label: "全站授权", icon: ShieldCheck },
  { href: "/dashboard/admin/ai", label: "AI 服务配置", icon: Sparkles },
  { href: "/dashboard/admin/audit", label: "平台审计", icon: ClipboardList },
  { href: "/dashboard/platform", label: "平台设置", icon: Settings },
]
const roleLabel = (role?: string) => role === "admin" ? "管理员" : "普通用户"

export default function DashboardLayout({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [menuOpen, setMenuOpen] = useState(false)
  const [signingOut, setSigningOut] = useState(false)
  const pathname = window.location.pathname

  useEffect(() => {
    let active = true
    currentUser().then(value => { if (active) setUser(value) }).catch(() => {
      clearToken()
      window.location.replace("/login")
    })
    return () => { active = false }
  }, [])

  async function handleLogout() {
    setSigningOut(true)
    try { await logout() } finally { clearToken(); window.location.replace("/login") }
  }

  function renderItems(items: NavItem[]) {
    return items.map(({ href, label, icon: Icon }) => {
      const active = pathname === href || (href === "/dashboard/groups" && pathname.startsWith("/dashboard/groups/"))
      return <a key={href} href={href} onClick={() => setMenuOpen(false)}
        aria-current={active ? "page" : undefined}
        className={"group relative flex items-center gap-3 rounded-lg px-3 py-2.5 text-[13px] font-medium transition-colors " + (active ? "bg-primary/[0.08] text-primary" : "text-muted-foreground hover:bg-muted/80 hover:text-foreground")}>
        {active && <span className="absolute inset-y-2 left-0 w-[3px] rounded-r-full bg-primary" />}
        <Icon className="size-[17px] shrink-0 opacity-85" />
        <span className="min-w-0 flex-1">{label}</span>
        {active && <ChevronRight className="size-3.5 opacity-60" />}
      </a>
    })
  }

  const section = (title: string, items: NavItem[]) => <section className="mt-5 px-3">
    <p className="px-3 pb-2 text-[10px] font-semibold tracking-[.12em] text-muted-foreground/75">{title}</p>
    <nav className="space-y-1">{renderItems(items)}</nav>
  </section>

  const sidebar = <div className="flex h-full flex-col bg-background">
    <a href="/dashboard" className="flex h-[68px] shrink-0 items-center gap-3 px-5">
      <span className="flex size-9 items-center justify-center rounded-xl bg-primary text-primary-foreground"><ShieldCheck className="size-[19px]" /></span>
      <span><span className="block text-[15px] font-semibold tracking-tight">TG-Group</span><span className="mt-0.5 block text-[10px] tracking-[.12em] text-muted-foreground">GROUP MANAGEMENT</span></span>
    </a>
    <Separator />
    <div className="flex-1 overflow-y-auto pb-5">
      {section("工作台", workspace)}
      {section("账户", account)}
      {section("业务管理", business)}
      {user?.role === "admin" && section("平台管理", administration)}
    </div>
    <div className="shrink-0 p-4"><Separator className="mb-4" />
      <a href="/dashboard/account/profile" className="mb-3 flex items-center gap-2.5 rounded-lg p-1.5 hover:bg-muted">
        <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold">{(user?.display_name || user?.email || "U").slice(0, 1).toUpperCase()}</div>
        <div className="min-w-0 flex-1"><p className="truncate text-xs font-medium">{user?.display_name || user?.email || "正在加载"}</p><p className="mt-0.5 text-[11px] text-muted-foreground">{roleLabel(user?.role)}</p></div>
        <Badge variant="outline" className="px-1.5 py-0 text-[10px] font-normal">在线</Badge>
      </a>
      <Button variant="outline" size="sm" className="h-9 w-full justify-start text-xs" onClick={handleLogout} disabled={signingOut}>
        <LogOut className="mr-2 size-3.5" />{signingOut ? "正在退出..." : "退出登录"}
      </Button>
    </div>
  </div>

  return <div className="min-h-svh bg-[#f6f7f9] dark:bg-background">
    <aside className="fixed inset-y-0 left-0 z-30 hidden w-[248px] border-r border-border/80 bg-background lg:block">{sidebar}</aside>
    <div className="lg:pl-[248px]">
      <header className="sticky top-0 z-20 flex h-[60px] items-center justify-between border-b border-border/80 bg-background/95 px-4 backdrop-blur-xl sm:px-6">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" className="lg:hidden" onClick={() => setMenuOpen(v => !v)} aria-label="打开导航菜单">{menuOpen ? <X className="size-5" /> : <Menu className="size-5" />}</Button>
          <span className="text-[13px] font-medium text-foreground/80">管理中心</span>
        </div>
        <a href="/dashboard/account/profile" className="flex items-center gap-2.5 rounded-lg px-2 py-1.5 hover:bg-muted">
          <span className="hidden max-w-56 truncate text-xs text-muted-foreground sm:inline">{user?.email || "账户"}</span>
          <span className="flex size-8 items-center justify-center rounded-full border bg-muted/60 text-xs font-semibold">{(user?.display_name || user?.email || "U").slice(0, 1).toUpperCase()}</span>
        </a>
      </header>
      {menuOpen && <div className="fixed inset-0 z-40 bg-black/30 backdrop-blur-[2px] lg:hidden" onClick={() => setMenuOpen(false)}>
        <aside className="h-full w-[280px] border-r bg-background shadow-2xl" onClick={e => e.stopPropagation()}>{sidebar}</aside>
      </div>}
      <div className="min-h-[calc(100svh-60px)] min-w-0">{children}</div>
    </div>
  </div>
}
