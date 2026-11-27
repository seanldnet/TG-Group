import { useCallback, useEffect, useMemo, useState, type FormEvent } from "react"
import { Bot, KeyRound, Plus, RefreshCw, Search, Trash2 } from "lucide-react"

import {
  addBot,
  listBots,
  removeBot,
  rotateBotToken,
  setBotStatus,
  type TelegramBot,
} from "@/lib/api"
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

export default function BotsPage() {
  const [items, setItems] = useState<TelegramBot[]>([])
  const [loading, setLoading] = useState(true)
  const [busyID, setBusyID] = useState("")
  const [error, setError] = useState("")
  const [notice, setNotice] = useState("")
  const [dialogOpen, setDialogOpen] = useState(false)
  const [dialogMode, setDialogMode] = useState<"add" | "rotate">("add")
  const [selectedBot, setSelectedBot] = useState<TelegramBot | null>(null)
  const [token, setToken] = useState("")
  const [saving, setSaving] = useState(false)
  const [search, setSearch] = useState("")
  const [statusFilter, setStatusFilter] = useState<"all" | TelegramBot["status"]>("all")

  const refresh = useCallback(async () => {
    setError("")
    try {
      const response = await listBots()
      setItems(response.items)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "加载 Bot 列表失败")
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const filteredItems = useMemo(() => {
    const query = search.trim().toLocaleLowerCase()
    return items.filter((bot) => {
      const matchesStatus = statusFilter === "all" || bot.status === statusFilter
      const matchesSearch = !query || [bot.username, bot.first_name, String(bot.telegram_bot_id)]
        .some((value) => value.toLocaleLowerCase().includes(query))
      return matchesStatus && matchesSearch
    })
  }, [items, search, statusFilter])

  function openAddDialog() {
    setDialogMode("add")
    setSelectedBot(null)
    setToken("")
    setDialogOpen(true)
  }

  function openRotateDialog(bot: TelegramBot) {
    setDialogMode("rotate")
    setSelectedBot(bot)
    setToken("")
    setDialogOpen(true)
  }

  async function submitToken(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!token.trim()) return
    setSaving(true)
    setError("")
    setNotice("")
    try {
      if (dialogMode === "add") {
        await addBot(token.trim())
        setNotice("Bot 已验证并添加")
      } else if (selectedBot) {
        await rotateBotToken(selectedBot.id, token.trim())
        setNotice("Bot Token 已轮换")
      }
      setToken("")
      setDialogOpen(false)
      await refresh()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "操作失败")
    } finally {
      setSaving(false)
    }
  }

  async function toggleStatus(bot: TelegramBot) {
    setBusyID(bot.id)
    setError("")
    setNotice("")
    try {
      await setBotStatus(bot.id, bot.status === "active" ? "disabled" : "active")
      setNotice(bot.status === "active" ? "Bot 已停用" : "Bot 已启用")
      await refresh()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "更新 Bot 状态失败")
    } finally {
      setBusyID("")
    }
  }

  async function unlinkBot(bot: TelegramBot) {
    const label = bot.username ? `@${bot.username}` : bot.first_name
    if (!window.confirm(`确定解绑 ${label}？解绑会删除平台保存的 Token。`)) return
    setBusyID(bot.id)
    setError("")
    setNotice("")
    try {
      await removeBot(bot.id)
      setNotice("Bot 已解绑")
      await refresh()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "解绑 Bot 失败")
    } finally {
      setBusyID("")
    }
  }

  return (
    <main className="w-full">
      <div className="mx-auto w-full max-w-6xl space-y-6 p-4 sm:p-6 lg:p-8">
        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
          <div>
            <Badge variant="secondary" className="mb-3">资源管理</Badge>
            <h1 className="text-3xl font-semibold tracking-tight">Telegram Bot</h1>
            <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">
              添加 BotFather Token 后，平台会向 Telegram 验证身份，并加密保存 Token。启用的 Bot 会由后台长轮询 worker 接收更新，并自动发现其所在群组。
            </p>
          </div>
          <div className="flex gap-2"><Button variant="outline" onClick={() => void refresh()} disabled={loading}><RefreshCw className={`mr-2 size-4 ${loading ? "animate-spin" : ""}`}/>刷新</Button><Button onClick={openAddDialog}>
            <Plus className="mr-2 size-4" />
            添加 Bot
          </Button>
        </div>

        {error && <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>}
        {notice && <p role="status" className="rounded-lg border border-border bg-background px-4 py-3 text-sm text-foreground">{notice}</p>}

        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="flex size-10 items-center justify-center rounded-xl bg-primary/8 text-primary">
                <Bot className="size-5" />
              </div>
              <div>
                <CardTitle>已接入 Bot</CardTitle>
                <CardDescription>共登记 {items.length} 个 Bot，当前显示 {filteredItems.length} 个；Token 不会在列表中展示。</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-3 sm:grid-cols-[minmax(220px,1fr)_180px]">
              <div className="relative"><Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"/><Input className="pl-9" value={search} onChange={event=>setSearch(event.target.value)} placeholder="搜索 Bot 名称、用户名或 Telegram ID"/></div>
              <Select value={statusFilter} onValueChange={value=>setStatusFilter(value as typeof statusFilter)}>
                <SelectTrigger aria-label="Bot 状态筛选"><SelectValue/></SelectTrigger>
                <SelectContent><SelectItem value="all">全部状态</SelectItem><SelectItem value="active">已启用</SelectItem><SelectItem value="disabled">已停用</SelectItem></SelectContent>
              </Select>
            </div>
            {loading ? (
              <p className="py-10 text-center text-sm text-muted-foreground">正在加载 Bot 列表...</p>
            ) : items.length === 0 ? (
              <div className="flex flex-col items-center gap-3 py-12 text-center">
                <Bot className="size-9 text-muted-foreground/60" />
                <div>
                  <p className="font-medium">还没有添加 Bot</p>
                  <p className="mt-1 text-sm text-muted-foreground">从 BotFather 获取 Token，点击“添加 Bot”开始接入。</p>
                </div>
                <Button variant="outline" onClick={openAddDialog}>添加第一个 Bot</Button>
              </div>
            ) : filteredItems.length === 0 ? (
              <p className="py-10 text-center text-sm text-muted-foreground">没有符合当前搜索和状态条件的 Bot。</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Bot</TableHead>
                    <TableHead>Telegram ID</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead>验证时间</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredItems.map((bot) => (
                    <TableRow key={bot.id}>
                      <TableCell>
                        <div className="font-medium">{bot.first_name || "Telegram Bot"}</div>
                        <div className="text-xs text-muted-foreground">{bot.username ? `@${bot.username}` : "未设置用户名"}</div>
                      </TableCell>
                      <TableCell className="font-mono text-xs">{bot.telegram_bot_id}</TableCell>
                      <TableCell>
                        <Badge variant={bot.status === "active" ? "default" : "secondary"}>
                          {bot.status === "active" ? "已启用" : "已停用"}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {new Date(bot.verified_at).toLocaleString()}
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end gap-1">
                          <Button size="sm" variant="outline" disabled={busyID === bot.id} onClick={() => void toggleStatus(bot)}>
                            {bot.status === "active" ? "停用" : "启用"}
                          </Button>
                          <Button size="icon-sm" variant="ghost" aria-label="轮换 Token" title="轮换 Token" disabled={busyID === bot.id} onClick={() => openRotateDialog(bot)}>
                            <RefreshCw className="size-4" />
                          </Button>
                          <Button size="icon-sm" variant="ghost" aria-label="解绑 Bot" title="解绑 Bot" className="text-destructive hover:text-destructive" disabled={busyID === bot.id} onClick={() => void unlinkBot(bot)}>
                            <Trash2 className="size-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>

      <Dialog open={dialogOpen} onOpenChange={(open) => { setDialogOpen(open); if (!open) setToken("") }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{dialogMode === "add" ? "添加 Telegram Bot" : "轮换 Bot Token"}</DialogTitle>
            <DialogDescription>
              {dialogMode === "add" ? "粘贴从 @BotFather 获取的 Bot Token。平台会先调用 Telegram getMe 验证。" : `为 ${selectedBot?.username ? `@${selectedBot.username}` : "此 Bot"} 设置新 Token；新 Token 必须属于同一个 Bot。`}
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={submitToken} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="telegram-bot-token">Bot Token</Label>
              <Input
                id="telegram-bot-token"
                type="password"
                autoComplete="off"
                spellCheck={false}
                placeholder="123456789:AA..."
                value={token}
                onChange={(event) => setToken(event.target.value)}
                required
              />
              <p className="text-xs text-muted-foreground">Token 只在提交时发送给服务端，不会显示在 Bot 列表或操作结果中。</p>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>取消</Button>
              <Button type="submit" disabled={saving || !token.trim()}>
                <KeyRound className="mr-2 size-4" />
                {saving ? "正在验证..." : dialogMode === "add" ? "验证并添加" : "验证并轮换"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </main>
  )
}
