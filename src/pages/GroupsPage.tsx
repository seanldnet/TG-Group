import { useCallback, useEffect, useMemo, useState } from "react"
import { Hash, RefreshCw, Search, UsersRound } from "lucide-react"

import { listGroups, type TelegramGroup } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
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

function groupStatusLabel(status: TelegramGroup["status"]) {
  if (status === "left") return "Bot 已离开"
  if (status === "kicked") return "Bot 已被移除"
  return "正常"
}

export default function GroupsPage() {
  const [items, setItems] = useState<TelegramGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState("")
  const [search, setSearch] = useState("")
  const [statusFilter, setStatusFilter] = useState<"all" | TelegramGroup["status"]>("all")

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

  const filteredItems = useMemo(() => {
    const query = search.trim().toLocaleLowerCase()
    return items.filter((group) => {
      const matchesStatus = statusFilter === "all" || group.status === statusFilter
      const matchesSearch = !query || [group.title, group.username, String(group.telegram_chat_id)]
        .some((value) => value.toLocaleLowerCase().includes(query))
      return matchesStatus && matchesSearch
    })
  }, [items, search, statusFilter])

  return (
    <main className="w-full">
      <div className="mx-auto w-full max-w-6xl space-y-6 p-4 sm:p-6 lg:p-8">
        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight sm:text-[28px]">群组管理</h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
              平台通过已启用 Bot 的 Telegram 长轮询更新自动发现群组、超级群组和频道。将 Bot 添加到目标群组后，收到相关更新即可登记。
            </p>
          </div>
          <Button variant="outline" onClick={() => void refresh(true)} disabled={refreshing}>
            <RefreshCw className={`mr-2 size-4 ${refreshing ? "animate-spin" : ""}`} />
            刷新
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
                <CardTitle>群组列表</CardTitle>
                <CardDescription>共 {items.length} 个群组，当前显示 {filteredItems.length} 个</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-3 sm:grid-cols-[minmax(220px,1fr)_180px]">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input className="pl-9" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索群组名称、用户名或 Chat ID" />
              </div>
              <Select value={statusFilter} onValueChange={(value) => setStatusFilter(value as typeof statusFilter)}>
                <SelectTrigger aria-label="群组状态筛选"><SelectValue placeholder="全部状态" /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部状态</SelectItem>
                  <SelectItem value="active">正常</SelectItem>
                  <SelectItem value="left">Bot 已离开</SelectItem>
                  <SelectItem value="kicked">Bot 已被移除</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {loading ? (
              <p className="py-10 text-center text-sm text-muted-foreground">正在加载群组列表...</p>
            ) : items.length === 0 ? (
              <div className="flex flex-col items-center gap-3 py-12 text-center">
                <Hash className="size-9 text-muted-foreground/60" />
                <div>
                  <p className="font-medium">暂无群组</p>
                  <p className="mt-1 max-w-lg text-sm text-muted-foreground">添加机器人并将其邀请至 Telegram 群组，即可开始管理。</p>
                </div>
                <Button asChild variant="outline"><a href="/dashboard/bots">添加机器人</a></Button>
              </div>
            ) : filteredItems.length === 0 ? (
              <p className="py-10 text-center text-sm text-muted-foreground">没有符合当前搜索和状态条件的群组。</p>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>群组 / 频道</TableHead>
                      <TableHead>群组 ID</TableHead>
                      <TableHead>类型</TableHead>
                      <TableHead>机器人</TableHead>
                      <TableHead>最近活动</TableHead>
                      <TableHead>状态</TableHead>
                      <TableHead>管理</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredItems.map((group) => (
                      <TableRow key={group.id}>
                        <TableCell>
                          <div className="font-medium">{group.title || "未命名群组"}</div>
                          <div className="text-xs text-muted-foreground">{group.username ? `@${group.username}` : "未设置用户名"}</div>
                        </TableCell>
                        <TableCell className="font-mono text-xs">{group.telegram_chat_id}</TableCell>
                        <TableCell><Badge variant="outline">{groupTypeLabel(group.chat_type)}</Badge>{group.is_forum && <Badge className="ml-1" variant="secondary">话题</Badge>}</TableCell>
                        <TableCell className="font-mono text-xs">{group.bot_id.slice(0, 8)}…</TableCell>
                        <TableCell className="text-xs text-muted-foreground">{new Date(group.last_seen_at).toLocaleString()}</TableCell>
                        <TableCell><Badge variant={group.status === "active" ? "default" : "secondary"}>{groupStatusLabel(group.status)}</Badge></TableCell>
                        <TableCell>
                          <div className="flex items-center gap-3">
                            <a className="text-sm text-primary hover:underline" href={`/dashboard/groups/${group.id}/members`}>成员</a>
                            <a className="text-sm text-primary hover:underline" href={`/dashboard/groups/${group.id}/events`}>消息日志</a>
                            <a className="text-sm text-primary hover:underline" href={`/dashboard/groups/${group.id}/rules`}>规则</a>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </main>
  )
}
