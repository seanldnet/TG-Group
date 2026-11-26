import { useEffect, useState, type FormEvent } from "react"
import { ArrowLeft, LoaderCircle, ShieldCheck } from "lucide-react"

import { currentUser, getToken, login, saveToken } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { CaptchaField } from "@/components/CaptchaField"

export default function LoginPage() {
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)
  const [captchaToken, setCaptchaToken] = useState("")
  const [captchaReset, setCaptchaReset] = useState(0)

  useEffect(() => {
    if (!getToken()) return
    let active = true
    currentUser().then(() => {
      if (active) window.location.replace("/dashboard")
    }).catch(() => {
      // Invalid/expired credentials are handled by the API client's refresh flow.
    })
    return () => { active = false }
  }, [])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError("")
    setLoading(true)
    try {
      const session = await login({ email, password, captcha_token: captchaToken })
      saveToken(session.token, session.refresh_token)
      window.location.assign("/dashboard")
    } catch (cause) {
      setCaptchaToken("")
      setCaptchaReset((value) => value + 1)
      setError(cause instanceof Error ? cause.message : "登录失败，请稍后重试")
    } finally {
      setLoading(false)
    }
  }

  return (
    <main className="flex min-h-svh items-center justify-center bg-muted/30 px-4 py-10">
      <div className="w-full max-w-md">
        <a href="/" className="mb-7 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
          <ArrowLeft className="size-4" />
          返回首页
        </a>
        <Card className="border-border/80 shadow-lg shadow-black/5">
          <CardHeader className="space-y-4">
            <div className="flex size-11 items-center justify-center rounded-xl bg-primary text-primary-foreground">
              <ShieldCheck className="size-5" />
            </div>
            <div>
              <Badge variant="secondary" className="mb-3">TG-Group 控制台</Badge>
              <CardTitle className="text-2xl">登录账号</CardTitle>
              <CardDescription className="mt-2 leading-6">
                登录后进入你的社群管理工作台。
              </CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            <form className="space-y-5" onSubmit={handleSubmit}>
              <div className="space-y-2">
                <Label htmlFor="email">邮箱或管理员账号</Label>
                <Input
                  id="email"
                  type="text"
                  autoComplete="username"
                  placeholder="admin / name@example.com"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="password">密码</Label>
                <Input
                  id="password"
                  type="password"
                  autoComplete="current-password"
                  placeholder="请输入密码"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  required
                />
              </div>
              <CaptchaField key={captchaReset} onToken={setCaptchaToken} />
              {error && (
                <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive">
                  {error}
                </p>
              )}
              <Button className="w-full" type="submit" disabled={loading}>
                {loading && <LoaderCircle className="size-4 animate-spin" />}
                {loading ? "正在登录..." : "登录"}
              </Button>
            </form>
            <p className="mt-6 text-center text-sm text-muted-foreground">
              还没有账号？{" "}
              <a href="/register" className="font-medium text-primary hover:underline">立即注册</a>
            </p>
          </CardContent>
        </Card>
        <p className="mt-6 text-center text-xs text-muted-foreground">
          登录即表示你将遵守平台使用规则。
        </p>
      </div>
    </main>
  )
}
