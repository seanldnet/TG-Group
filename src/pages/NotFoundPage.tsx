import { ArrowLeft, Home, SearchX } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export default function NotFoundPage() {
  return (
    <main className="flex min-h-svh items-center justify-center bg-background px-4 py-12">
      <Card className="w-full max-w-lg border-border/80 shadow-sm">
        <CardHeader className="items-center text-center">
          <span className="mb-3 flex size-14 items-center justify-center rounded-2xl bg-muted text-muted-foreground">
            <SearchX className="size-7" />
          </span>
          <Badge variant="outline">404 · 页面不存在</Badge>
          <CardTitle className="pt-2 text-2xl">找不到这个页面</CardTitle>
          <CardDescription className="max-w-sm leading-6">
            页面可能已移动、链接地址有误，或当前账号没有对应入口。你可以返回首页或进入管理中心继续操作。
          </CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col justify-center gap-2 sm:flex-row">
          <Button asChild variant="outline">
            <a href="/" aria-label="返回网站首页"><Home className="mr-2 size-4" />网站首页</a>
          </Button>
          <Button asChild>
            <a href="/dashboard"><ArrowLeft className="mr-2 size-4" />管理中心</a>
          </Button>
        </CardContent>
      </Card>
    </main>
  )
}
