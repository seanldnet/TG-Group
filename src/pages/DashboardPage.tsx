import { useEffect, useState } from "react"
import { Bot, LogOut, ShieldCheck, UsersRound } from "lucide-react"

import { clearToken, currentUser, logout, getTelegramIdentity, createTelegramLink, unlinkTelegramIdentity, changePassword, saveToken, type TelegramIdentityStatus, type User } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

export default function DashboardPage() {
  const [user, setUser] = useState<User | null>(null)
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(true)
  const [signingOut, setSigningOut] = useState(false)
  const [telegram, setTelegram] = useState<TelegramIdentityStatus | null>(null)
  const [telegramBusy, setTelegramBusy] = useState(false)
  const [telegramError, setTelegramError] = useState("")
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [passwordBusy, setPasswordBusy] = useState(false)
  const [passwordMessage, setPasswordMessage] = useState("")
  const [passwordError, setPasswordError] = useState("")

  useEffect(() => {
    let active = true
    Promise.all([currentUser(), getTelegramIdentity()])
      .then(([value, telegramValue]) => {
        if (active) { setUser(value); setTelegram(telegramValue) }
      })
      .catch((cause: unknown) => {
        clearToken()
        if (active) {
          setError(cause instanceof Error ? cause.message : "登录状态已失效")
          window.location.replace("/login")
        }
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [])

  async function handleTelegramLink() {
    setTelegramBusy(true)
    setTelegramError("")
    try {
      const challenge = await createTelegramLink()
      window.location.assign(challenge.url)
    } catch (cause) {
      setTelegramError(cause instanceof Error ? cause.message : "创建 Telegram 绑定链接失败")
      setTelegramBusy(false)
    }
  }

  async function handleTelegramUnlink() {
    if (!window.confirm("确定解除当前账号的 Telegram 绑定吗？")) return
    setTelegramBusy(true)
    setTelegramError("")
    try {
      await unlinkTelegramIdentity()
      setTelegram(await getTelegramIdentity())
    } catch (cause) {
      setTelegramError(cause instanceof Error ? cause.message : "解除 Telegram 绑定失败")
    } finally {
      setTelegramBusy(false)
    }
  }

  async function handlePasswordChange() {
    setPasswordMessage("")
    setPasswordError("")
    if (newPassword.length < 10 || newPassword.length > 72) {
      setPasswordError("新密码长度必须为 10-72 个字符")
      return
    }
    if (newPassword !== confirmPassword) {
      setPasswordError("两次输入的新密码不一致")
      return
    }
    setPasswordBusy(true)
    try {
      const session = await changePassword({ current_password: currentPassword, new_password: newPassword })
      saveToken(session.token, session.refresh_token)
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
      setPasswordMessage("密码修改成功，其他设备的登录会话已全部退出。")
    } catch (cause) {
      setPasswordError(cause instanceof Error ? cause.message : "修改密码失败")
    } finally {
      setPasswordBusy(false)
    }
  }

  async function handleLogout() {
    setSigningOut(true)
    try {
      await logout()
    } finally {
      clearToken()
      window.location.replace("/login")
    }
  }

  if (loading) {
    return (
      <main className="flex min-h-svh items-center justify-center bg-muted/30 px-4">
        <p className="text-sm text-muted-foreground">正在加载账号信息...</p>
      </main>
    )
  }

  if (!user) {
    return (
      <main className="flex min-h-svh items-center justify-center bg-muted/30 px-4">
        <p role="alert" className="text-sm text-destructive">{error || "正在返回登录页面..."}</p>
      </main>
    )
  }

  return (
    <main className="min-h-svh bg-muted/25">
      <header className="border-b border-border/70 bg-background">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-5 sm:px-8">
          <a href="/" className="flex items-center gap-3">
            <span className="flex size-9 items-center justify-center rounded-xl bg-primary text-primary-foreground">
              <ShieldCheck className="size-5" />
            </span>
            <span className="text-base font-semibold">TG-Group</span>
          </a>
          <Button variant="outline" size="sm" onClick={handleLogout} disabled={signingOut}>
            <LogOut className="mr-2 size-4" />
            {signingOut ? "正在退出..." : "退出登录"}
          </Button>
        </div>
      </header>

      <div className="mx-auto max-w-6xl px-5 py-10 sm:px-8 sm:py-14">
        <div className="mb-8">
          <Badge variant="secondary" className="mb-3">账号工作台</Badge>
          <h1 className="text-3xl font-semibold tracking-tight">
            欢迎回来，{user.display_name || user.email}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            账号已通过身份认证。可在工作台管理 Bot，并查看自动发现的 Telegram 群组。
          </p>
        </div>

        <div className="mb-4">
          <Card>
            <CardHeader>
              <CardTitle>Telegram 身份绑定</CardTitle>
              <CardDescription>通过平台官方 Telegram Bot 自动验证身份，不需要手动填写 Telegram ID。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {telegram?.linked ? (
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="text-sm">
                    <p className="font-medium">{telegram.telegram.username ? "@"+telegram.telegram.username : [telegram.telegram.first_name,telegram.telegram.last_name].filter(Boolean).join(" ") || "Telegram 用户"}</p>
                    <p className="text-xs text-muted-foreground">Telegram ID：{telegram.telegram.id}</p>
                  </div>
                  <Button variant="outline" size="sm" disabled={telegramBusy} onClick={handleTelegramUnlink}>解除绑定</Button>
                </div>
              ) : (
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <p className="text-sm text-muted-foreground">尚未绑定 Telegram。点击后跳转官方 Bot，按提示点击 Start 即可自动完成绑定。</p>
                  <Button size="sm" disabled={telegramBusy} onClick={handleTelegramLink}>{telegramBusy ? "正在创建链接..." : "绑定 Telegram"}</Button>
                </div>
              )}
              {telegramError && <p role="alert" className="text-sm text-destructive">{telegramError}</p>}
            </CardContent>
          </Card>
        </div>

        <div className="mb-4">
          <Card>
            <CardHeader>
              <CardTitle>修改登录密码</CardTitle>
              <CardDescription>修改后会撤销所有旧登录会话，当前设备将自动换发新会话。</CardDescription>
            </CardHeader>
            <CardContent className="max-w-xl space-y-4">
              <div className="space-y-2">
                <Label htmlFor="current-password">当前密码</Label>
                <Input id="current-password" type="password" autoComplete="current-password" value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-password">新密码（10-72 个字符）</Label>
                <Input id="new-password" type="password" autoComplete="new-password" value={newPassword} onChange={(event) => setNewPassword(event.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm-password">确认新密码</Label>
                <Input id="confirm-password" type="password" autoComplete="new-password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} />
              </div>
              {passwordError && <p role="alert" className="text-sm text-destructive">{passwordError}</p>}
              {passwordMessage && <p role="status" className="text-sm text-emerald-600">{passwordMessage}</p>}
              <Button onClick={handlePasswordChange} disabled={passwordBusy || !currentPassword || !newPassword || !confirmPassword}>
                {passwordBusy ? "正在修改..." : "确认修改密码"}
              </Button>
            </CardContent>
          </Card>
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <div className="flex size-10 items-center justify-center rounded-xl bg-primary/8 text-primary">
                <ShieldCheck className="size-5" />
              </div>
              <CardTitle>账号信息</CardTitle>
              <CardDescription>当前登录身份与租户空间</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              <div className="flex items-center justify-between gap-4 border-b border-border/70 pb-3">
                <span className="text-muted-foreground">邮箱</span>
                <span className="font-medium">{user.email}</span>
              </div>
              <div className="flex items-center justify-between gap-4 border-b border-border/70 pb-3">
                <span className="text-muted-foreground">账号角色</span>
                <Badge variant="outline">{user.role}</Badge>
              </div>
              <div className="space-y-1">
                <span className="text-muted-foreground">租户 ID</span>
                <p className="break-all font-mono text-xs">{user.tenant_id}</p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <div className="flex size-10 items-center justify-center rounded-xl bg-primary/8 text-primary">
                <Bot className="size-5" />
              </div>
              <CardTitle>社群资源</CardTitle>
              <CardDescription>管理已接入的 Bot，并继续配置 Telegram 群组</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-3 rounded-lg border border-border/70 p-3">
                <Bot className="size-5 text-muted-foreground" />
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">Telegram Bot</p>
                  <p className="text-xs text-muted-foreground">验证 Token、启停、轮换与解绑 Bot</p>
                </div>
              </div>
              <Button asChild variant="outline" className="w-full">
                <a href="/dashboard/bots">进入 Bot 管理</a>
              </Button>
              <div className="flex items-center gap-3 rounded-lg border border-border/70 p-3">
                <UsersRound className="size-5 text-muted-foreground" />
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium">Telegram 群组</p>
                  <p className="text-xs text-muted-foreground">查看 Bot 自动发现的群组、超级群组和频道</p>
                </div>
              </div>
              <Button asChild variant="outline" className="w-full">
                <a href="/dashboard/groups">查看群组列表</a>
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </main>
  )
}
