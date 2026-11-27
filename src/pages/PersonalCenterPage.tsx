import { useEffect, useState, type FormEvent } from "react"
import { ArrowUpRight, CalendarDays, CircleUserRound, Link2, ShieldCheck } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { currentUser, getTelegramIdentity, updateProfile, type User, type TelegramIdentityStatus } from "@/lib/api"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

function Field({ label, value }: { label: string; value: string }) {
  return <div className="space-y-1.5 rounded-lg border bg-card px-4 py-3">
    <p className="text-xs text-muted-foreground">{label}</p>
    <p className="break-all text-sm font-medium">{value || "未设置"}</p>
  </div>
}

export default function PersonalCenterPage() {
  const [user, setUser] = useState<User | null>(null)
  const [telegram, setTelegram] = useState<TelegramIdentityStatus | null>(null)
  const [accountError, setAccountError] = useState("")
  const [telegramError, setTelegramError] = useState("")
  const [loading, setLoading] = useState(true)
  const [profileName, setProfileName] = useState("")
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState("")
  useEffect(() => {
    let active = true
    Promise.allSettled([currentUser(), getTelegramIdentity()])
      .then(([accountResult, telegramResult]) => {
        if (!active) return
        if (accountResult.status === "fulfilled") {
          setUser(accountResult.value)
          setProfileName(accountResult.value.display_name || "")
        } else {
          setAccountError(accountResult.reason instanceof Error ? accountResult.reason.message : "账户信息加载失败")
        }
        if (telegramResult.status === "fulfilled") setTelegram(telegramResult.value)
        else setTelegramError(telegramResult.reason instanceof Error ? telegramResult.reason.message : "Telegram 关联状态暂时无法读取")
        setLoading(false)
      })
    return () => { active = false }
  }, [])
  async function saveProfile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setAccountError("")
    setNotice("")
    setSaving(true)
    try {
      const updated = await updateProfile({ display_name: profileName.trim() })
      setUser(updated)
      setProfileName(updated.display_name)
      setNotice("个人资料已保存")
    } catch (cause) {
      setAccountError(cause instanceof Error ? cause.message : "保存个人资料失败")
    } finally {
      setSaving(false)
    }
  }
  const displayName = user?.display_name || "用户"
  const telegramName = telegram?.linked
    ? [telegram.telegram.first_name, telegram.telegram.last_name].filter(Boolean).join(" ") || telegram.telegram.username || "已关联账号"
    : "尚未关联"
  return <main className="mx-auto w-full max-w-6xl space-y-6 p-4 sm:p-6 lg:p-8">
    <div className="flex flex-wrap items-center justify-between gap-4">
      <div><h1 className="text-2xl font-semibold tracking-tight">个人中心</h1><p className="mt-1 text-sm text-muted-foreground">查看和管理你的账户信息</p></div>
      <Badge variant="outline">账户资料</Badge>
    </div>
    {accountError && <p role="alert" className="text-sm text-destructive">账户资料：{accountError}</p>}
    {telegramError && <p role="alert" className="text-sm text-destructive">Telegram 关联状态：{telegramError}</p>}
    <Card>
      <CardContent className="flex flex-wrap items-center gap-4 p-5 sm:p-6">
        <div className="flex size-14 items-center justify-center rounded-full bg-primary/10 text-primary"><CircleUserRound className="size-7"/></div>
        <div className="min-w-0 flex-1"><p className="truncate text-lg font-semibold">{displayName}</p><p className="truncate text-sm text-muted-foreground">{loading ? "正在加载账户..." : user?.email || "账户信息不可用"}</p></div>
        <Badge variant={user?.role === "admin" ? "default" : "secondary"}>{user?.role === "admin" ? "管理员" : user?.role === "agent" ? "代理" : "普通用户"}</Badge>
      </CardContent>
    </Card>
    <div className="grid gap-5 lg:grid-cols-[1.35fr_1fr]">
      <Card>
        <CardHeader><CardTitle className="text-base">基本资料</CardTitle><CardDescription>当前账户的注册信息</CardDescription></CardHeader>
        <CardContent className="grid gap-3 sm:grid-cols-2">
          <Field label="用户 ID" value={user?.id || (loading ? "加载中..." : "暂不可用")}/>
          <Field label="登录邮箱" value={user?.email || (loading ? "加载中..." : "暂不可用")}/>
          <Field label="显示名称" value={user?.display_name || (loading ? "加载中..." : "暂不可用")}/>
          <Field label="账户角色" value={user ? user.role === "admin" ? "管理员" : user.role === "agent" ? "代理" : "普通用户" : loading ? "加载中..." : "暂不可用"}/>
          <Field label="注册时间" value={user?.created_at ? new Date(user.created_at).toLocaleString() : loading ? "加载中..." : "暂不可用"}/>
          <Field label="所属租户" value={user?.tenant_id || (loading ? "加载中..." : "暂不可用")}/>
          <form className="space-y-3 rounded-lg border p-4 sm:col-span-2" onSubmit={saveProfile}>
            <div className="space-y-1"><Label htmlFor="profile-display-name">显示名称</Label><Input id="profile-display-name" value={profileName} onChange={event => setProfileName(event.target.value)} maxLength={40} placeholder="输入显示名称" disabled={!user || saving}/><p className="text-xs text-muted-foreground">最多 40 个字符，修改后会同步更新账户显示名称。</p></div>
            <div className="flex flex-wrap items-center justify-between gap-3"><span className="text-xs text-muted-foreground">{profileName.length}/40</span><Button type="submit" disabled={!user || saving || !profileName.trim() || profileName.trim() === user.display_name}>{saving ? "正在保存..." : "保存资料"}</Button></div>
          </form>
        </CardContent>
      </Card>
      <div className="space-y-5">
        <Card>
          <CardHeader><CardTitle className="flex items-center gap-2 text-base"><Link2 className="size-4"/>Telegram 账号</CardTitle><CardDescription>与 Telegram 身份关联后，可使用 Bot 管理入口</CardDescription></CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center justify-between gap-3"><span className="text-sm text-muted-foreground">关联状态</span><Badge variant={telegram?.linked ? "default" : "outline"}>{telegram?.linked ? "已关联" : telegram ? "未关联" : telegramError ? "读取失败" : "加载中"}</Badge></div>
            <Field label="Telegram 用户" value={telegram ? telegramName : telegramError ? "暂不可用" : "加载中..."}/>
            <Field label="Telegram ID" value={telegram?.linked && telegram.telegram.id ? String(telegram.telegram.id) : telegram ? "—" : telegramError ? "暂不可用" : "加载中..."}/>
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
    {notice && <p role="status" className="text-sm text-emerald-600">{notice}</p>}
    <p className="flex items-center gap-2 text-xs text-muted-foreground"><CalendarDays className="size-3.5"/>资料修改会记录到账户安全日志。</p>
  </main>
}
