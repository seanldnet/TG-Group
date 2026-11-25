import { useEffect, useState } from "react"
import { Bot, LogOut, ShieldCheck, UsersRound } from "lucide-react"

import { clearToken, currentUser, logout, type User } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
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

  useEffect(() => {
    let active = true
    currentUser()
      .then((value) => {
        if (active) setUser(value)
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
            账号已通过身份认证。Bot 与群组管理模块正在持续开发。
          </p>
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
                <div>
                  <p className="text-sm font-medium">群组管理</p>
                  <p className="text-xs text-muted-foreground">群组发现与规则配置开发中</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </main>
  )
}
