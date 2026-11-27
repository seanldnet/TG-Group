import { useCallback, useEffect, useMemo, useState } from "react"
import { ArrowLeft, RefreshCw, Search, ShieldCheck, Trash2 } from "lucide-react"

import { deleteGroupMemberList, listGroupMemberLists, upsertGroupMemberList, type TelegramGroupMemberListEntry } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

const listLabels: Record<TelegramGroupMemberListEntry["list_type"], string> = {
  blacklist: "黑名单",
  whitelist: "白名单",
  exempt: "验证豁免",
}

export default function GroupMemberListsPage({ groupID }: { groupID: string }) {
  const [items, setItems] = useState<TelegramGroupMemberListEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [deletingKey, setDeletingKey] = useState("")
  const [error, setError] = useState("")
  const [notice, setNotice] = useState("")
  const [userID, setUserID] = useState("")
  const [listType, setListType] = useState<TelegramGroupMemberListEntry["list_type"]>("blacklist")
  const [reason, setReason] = useState("")
  const [search, setSearch] = useState("")
  const [filterType, setFilterType] = useState<"all" | TelegramGroupMemberListEntry["list_type"]>("all")

  const refresh = useCallback(async (manual = false) => {
    setError("")
    if (manual) setRefreshing(true)
    try {
      const response = await listGroupMemberLists(groupID)
      setItems(response.items)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "加载成员名单失败")
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [groupID])

  useEffect(() => { void refresh() }, [refresh])

  const filteredItems = useMemo(() => {
    const query = search.trim().toLowerCase()
    return items.filter((item) => {
      if (filterType !== "all" && item.list_type !== filterType) return false
      if (!query) return true
      return String(item.telegram_user_id).includes(query) || item.reason.toLowerCase().includes(query)
    })
  }, [items, search, filterType])

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const id = Number(userID)
    if (!/^\d+$/.test(userID.trim()) || !Number.isSafeInteger(id) || id <= 0) {
      setError("Telegram 用户 ID 必须是正整数")
      return
    }
    setSaving(true)
    setError("")
    setNotice("")
    try {
      await upsertGroupMemberList(groupID, { telegram_user_id: id, list_type: listType, reason: reason.trim() })
      setUserID("")
      setReason("")
      await refresh()
      setNotice("成员名单已保存")
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "保存成员名单失败")
    } finally {
      setSaving(false)
    }
  }

  async function remove(item: TelegramGroupMemberListEntry) {
    const key = `${item.telegram_user_id}:${item.list_type}`
    if (deletingKey || !window.confirm(`确认从“${listLabels[item.list_type]}”中移除用户 ${item.telegram_user_id}？`)) return
    setDeletingKey(key)
    setError("")
    setNotice("")
    try {
      await deleteGroupMemberList(groupID, item.telegram_user_id, item.list_type)
      await refresh()
      setNotice(`已从${listLabels[item.list_type]}移除用户 ${item.telegram_user_id}`)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "移除名单项目失败")
    } finally {
      setDeletingKey("")
    }
  }

  return (
    <main className="w-full">
      <a href={`/dashboard/groups/${groupID}/members`} className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"><ArrowLeft className="size-4" />返回群成员</a>
      <div className="mx-auto w-full max-w-6xl space-y-6 p-4 sm:p-6 lg:p-8">
        <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-end">
          <div>
            <Badge variant="secondary" className="mb-3">群组安全</Badge>
            <h1 className="text-3xl font-semibold tracking-tight">成员名单</h1>
            <p className="mt-2 text-sm text-muted-foreground">黑名单成员会在入群时被移除；白名单和验证豁免成员跳过入群验证。黑名单优先级最高。</p>
          </div>
          <Button variant="outline" onClick={() => void refresh(true)} disabled={refreshing}><RefreshCw className={`mr-2 size-4 ${refreshing ? "animate-spin" : ""}`} />刷新</Button>
        </div>
        {error && <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>}
        {notice && <p role="status" className="rounded-lg border px-4 py-3 text-sm text-muted-foreground">{notice}</p>}
        <div className="grid gap-6 lg:grid-cols-[0.85fr_1.15fr]">
          <Card>
            <CardHeader><CardTitle>添加名单成员</CardTitle><CardDescription>按 Telegram 用户 ID 管理名单，同一用户可以同时存在于不同名单。</CardDescription></CardHeader>
            <CardContent>
              <form className="space-y-4" onSubmit={(event) => void submit(event)}>
                <div className="space-y-2"><Label htmlFor="member-list-user-id">Telegram 用户 ID</Label><Input id="member-list-user-id" inputMode="numeric" pattern="[0-9]*" value={userID} onChange={(event) => setUserID(event.target.value)} placeholder="例如 123456789" required /></div>
                <div className="space-y-2"><Label>名单类型</Label><Select value={listType} onValueChange={(value) => setListType(value as TelegramGroupMemberListEntry["list_type"])}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="blacklist">黑名单</SelectItem><SelectItem value="whitelist">白名单</SelectItem><SelectItem value="exempt">验证豁免</SelectItem></SelectContent></Select></div>
                <div className="space-y-2"><Label htmlFor="member-list-reason">备注原因</Label><Input id="member-list-reason" value={reason} onChange={(event) => setReason(event.target.value)} maxLength={500} placeholder="可选，最多 500 字" /></div>
                <Button type="submit" disabled={saving || !userID.trim()}><ShieldCheck className="mr-2 size-4" />{saving ? "保存中..." : "添加到名单"}</Button>
              </form>
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>当前名单</CardTitle><CardDescription>最多展示最近更新的 500 条名单记录；搜索和类型筛选作用于当前已加载记录。</CardDescription></CardHeader>
            <CardContent className="space-y-3">
              <div className="flex flex-col gap-2 sm:flex-row">
                <div className="relative min-w-0 flex-1"><Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" /><Input className="pl-9" value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索 Telegram 用户 ID 或备注" /></div>
                <Select value={filterType} onValueChange={(value) => setFilterType(value as typeof filterType)}><SelectTrigger className="sm:w-40"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="all">全部类型</SelectItem><SelectItem value="blacklist">黑名单</SelectItem><SelectItem value="whitelist">白名单</SelectItem><SelectItem value="exempt">验证豁免</SelectItem></SelectContent></Select>
              </div>
              {loading ? <p className="py-8 text-center text-sm text-muted-foreground">正在加载...</p> : filteredItems.length === 0 ? <p className="py-8 text-center text-sm text-muted-foreground">{items.length ? "没有符合条件的名单记录。" : "当前群组没有名单记录。"}</p> : filteredItems.map((item) => {
                const key = `${item.telegram_user_id}:${item.list_type}`
                return <div key={key} className="flex flex-col gap-3 rounded-lg border p-3 sm:flex-row sm:items-center">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2"><span className="font-mono text-sm">{item.telegram_user_id}</span><Badge variant={item.list_type === "blacklist" ? "destructive" : "secondary"}>{listLabels[item.list_type]}</Badge></div>
                    <p className="mt-1 text-xs text-muted-foreground">{item.reason || "未填写备注"} · {new Date(item.created_at).toLocaleString()}</p>
                  </div>
                  <Button size="sm" variant="outline" disabled={Boolean(deletingKey)} onClick={() => void remove(item)}><Trash2 className="mr-2 size-4" />{deletingKey === key ? "移除中..." : "移除"}</Button>
                </div>
              })}
            </CardContent>
          </Card>
        </div>
      </div>
    </main>
  )
}
