import { useCallback, useEffect, useState } from "react"
import { ArrowLeft, Hash, RefreshCw, UsersRound } from "lucide-react"

import { listGroups, type TelegramGroup } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

function groupTypeLabel(type: TelegramGroup["chat_type"]) {
  if (type === "supergroup") return "超级群组"
  if (type === "channel") return "频道"
  return "普通群组"
}

export default function GroupsPage() {
  const [items, setItems] = useState<TelegramGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState("")

  const refresh = useCallback(async (manual = false) => {
    setError("")
    if (manual) setRefreshing(true)
    try {
      const response = await listGroups()
      setItems(response.items)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "加载群组列表失败")
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  return (
    <main className="min-h-svh bg-muted/25">
      <header className="border-b border-border/70 bg-background">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-5 sm:px-8">
          <a href="/dashboard" className="inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground">
            <ArrowLeft className="size-4" />
            返回工作台
          </a>
          <div className="flex items-center gap-2 text-sm font-semibold">
            <UsersRound className="size-5 text-primary" />
            TG-Group
          </div>
        </div>
      </header>

      <div className="mx-auto max-w-6xl space-y-6 px-5 py-9 sm:px-8 sm:py-12">
        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
          <div>
            <Badge variant="secondary" className="mb-3">资源管理</Badge>
            <h1 className="text-3xl font-semibold tracking-tight">Telegram 群组</h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
              平台通过已启用 Bot 的 Telegram 长轮询更新自动发现群组、超级群组和频道。将 Bot 添加到目标群组后，收到相关更新即可登记。
            </p>
          </div>
          <Button variant="outline" onClick={() => void refresh(true)} disabled={refreshing}>
            <RefreshCw className={`mr-2 size-4 ${refreshing ? "animate-spin" : ""}`} />
            刷新列表
          </Button>
        </div>

        {error && <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>}

        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="flex size-10 items-center justify-center rounded-xl bg-primary/8 text-primary">
                <UsersRound className="size-5" />
              </div>
              <div>
                <CardTitle>已发现群组</CardTitle>
                <CardDescription>当前租户共发现 {items.length} 个群组、超级群组或频道。</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {loading ? (
              <p className="py-10 text-center text-sm text-muted-foreground">正在加载群组列表...</p>
            ) : items.length === 0 ? (
              <div className="flex flex-col items-center gap-3 py-12 text-center">
                <Hash className="size-9 text-muted-foreground/60" />
                <div>
                  <p className="font-medium">暂未发现群组</p>
                  <p className="mt-1 max-w-lg text-sm text-muted-foreground">
                    先在 Bot 管理中添加并启用 Bot，再将它添加到 Telegram 群组。群组产生消息或 Bot 成员状态更新后，平台会自动发现。
                  </p>
                </div>
                <Button asChild variant="outline"><a href="/dashboard/bots">前往 Bot 管理</a></Button>
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>群组 / 频道</TableHead>
                    <TableHead>Telegram Chat ID</TableHead>
                    <TableHead>类型</TableHead>
                    <TableHead>来源 Bot</TableHead>
                    <TableHead>最近发现</TableHead>
                    <TableHead>操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((group) => (
                    <TableRow key={group.id}>
                      <TableCell>
                        <div className="font-medium">{group.title || "未命名群组"}</div>
                        <div className="text-xs text-muted-foreground">{group.username ? `@${group.username}` : "无公开用户名"}</div>
                      </TableCell>
                      <TableCell className="font-mono text-xs">{group.telegram_chat_id}</TableCell>
                      <TableCell><Badge variant="outline">{groupTypeLabel(group.chat_type)}</Badge>{group.is_forum && <Badge className="ml-1" variant="secondary">话题</Badge>}</TableCell>
                      <TableCell className="font-mono text-xs">{group.bot_id.slice(0, 8)}…</TableCell>
                      <TableCell className="text-xs text-muted-foreground">{new Date(group.last_seen_at).toLocaleString()}</TableCell>
                      <TableCell><div className="flex items-center gap-3"><a className="text-sm text-primary hover:underline" href={`/dashboard/groups/${group.id}/members`}>成员</a><a className="text-sm text-primary hover:underline" href={`/dashboard/groups/${group.id}/events`}>消息日志</a></div></TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </main>
  )
}
