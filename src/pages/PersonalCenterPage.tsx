import { useEffect, useState } from "react"
import { ArrowUpRight, CalendarDays, CircleUserRound, Link2, ShieldCheck } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { currentUser, getTelegramIdentity, type User, type TelegramIdentityStatus } from "@/lib/api"

function Field({ label, value }: { label: string; value: string }) {
  return <div className="space-y-1.5 rounded-lg border bg-card px-4 py-3">
    <p className="text-xs text-muted-foreground">{label}</p>
    <p className="break-all text-sm font-medium">{value || "未设置"}</p>
  </div>
}

export default function PersonalCenterPage() {
  const [user, setUser] = useState<User | null>(null)
  const [telegram, setTelegram] = useState<TelegramIdentityStatus | null>(null)
  const [error, setError] = useState("")
  useEffect(() => {
    let active = true
    Promise.all([currentUser(), getTelegramIdentity()])
      .then(([account, identity]) => { if (active) { setUser(account); setTelegram(identity) } })
      .catch((e: unknown) => { if (active) setError(e instanceof Error ? e.message : "账户信息加载失败") })
    return () => { active = false }
  }, [])
  const displayName = user?.display_name || "用户"
  const telegramName = telegram?.linked
    ? [telegram.telegram.first_name, telegram.telegram.last_name].filter(Boolean).join(" ") || telegram.telegram.username || "已关联账号"
    : "尚未关联"
  return <main className="mx-auto w-full max-w-6xl space-y-6 p-4 sm:p-6 lg:p-8">
    <div className="flex flex-wrap items-center justify-between gap-4">
      <div><h1 className="text-2xl font-semibold tracking-tight">个人中心</h1><p className="mt-1 text-sm text-muted-foreground">查看和管理你的账户信息</p></div>
      <Badge variant="outline">账户资料</Badge>
    </div>
    {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
    <Card>
      <CardContent className="flex flex-wrap items-center gap-4 p-5 sm:p-6">
        <div className="flex size-14 items-center justify-center rounded-full bg-primary/10 text-primary"><CircleUserRound className="size-7"/></div>
        <div className="min-w-0 flex-1"><p className="truncate text-lg font-semibold">{displayName}</p><p className="truncate text-sm text-muted-foreground">{user?.email || "正在加载账户..."}</p></div>
        <Badge variant={user?.role === "admin" ? "default" : "secondary"}>{user?.role === "admin" ? "管理员" : "普通用户"}</Badge>
      </CardContent>
    </Card>
    <div className="grid gap-5 lg:grid-cols-[1.35fr_1fr]">
      <Card>
        <CardHeader><CardTitle className="text-base">基本资料</CardTitle><CardDescription>当前账户的注册信息</CardDescription></CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-2">
          <Field label="用户 ID" value={user?.id || "加载中..."}/>
          <Field label="登录邮箱" value={user?.email || "加载中..."}/>
          <Field label="显示名称" value={user?.display_name || "加载中..."}/>
          <Field label="账户角色" value={user?.role === "admin" ? "管理员" : user ? "普通用户" : "加载中..."}/>
          <Field label="注册时间" value={user?.created_at ? new Date(user.created_at).toLocaleString() : "加载中..."}/>
          <Field label="所属租户" value={user?.tenant_id || "加载中..."}/>
        </CardContent>
      </Card>
      <div className="space-y-5">
        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2 text-base"><Link2 className="size-4"/>Telegram 账号</CardTitle><CardDescription>与 Telegram 身份关联后，可使用 Bot 管理入口</CardDescription></CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center justify-between gap-3"><span className="text-sm text-muted-foreground">关联状态</span><Badge variant={telegram?.linked ? "default" : "outline"}>{telegram?.linked ? "已关联" : telegram ? "未关联" : "加载中"}</Badge></div>
            <Field label="Telegram 用户" value={telegramName}/>
            <Field label="Telegram ID" value={telegram?.linked && telegram.telegram.id ? String(telegram.telegram.id) : "—"}/>
            {telegram?.linked && telegram.telegram.username && <Field label="用户名" value={"@" + telegram.telegram.username}/>}
            <Button asChild variant="outline" className="w-full"><a href="/dashboard/account/links">管理账号关联 <ArrowUpRight className="ml-2 size-4"/></a></Button>
          </CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2 text-base"><ShieldCheck className="size-4"/>账户安全</CardTitle><CardDescription>保护账户和登录设备</CardDescription></CardHeader>
          <CardContent className="grid gap-2">
            <Button asChild variant="outline" className="justify-between"><a href="/dashboard/security/password">修改登录密码 <ArrowUpRight className="size-4"/></a></Button>
            <Button asChild variant="outline" className="justify-between"><a href="/dashboard/security/sessions">管理登录设备 <ArrowUpRight className="size-4"/></a></Button>
            <Button asChild variant="outline" className="justify-between"><a href="/dashboard/security/logs">查看安全日志 <ArrowUpRight className="size-4"/></a></Button>
          </CardContent>
        </Card>
      </div>
    </div>
    <p className="flex items-center gap-2 text-xs text-muted-foreground"><CalendarDays className="size-3.5"/>资料来自当前登录账户；个人资料编辑功能将在后续接入保存接口后开放。</p>
  </main>
}
