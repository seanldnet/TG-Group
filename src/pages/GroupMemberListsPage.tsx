import { useCallback, useEffect, useState } from "react"
import { ArrowLeft, RefreshCw, ShieldCheck, Trash2 } from "lucide-react"

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
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const [userID, setUserID] = useState("")
  const [listType, setListType] = useState<TelegramGroupMemberListEntry["list_type"]>("blacklist")
  const [reason, setReason] = useState("")

  const refresh = useCallback(async () => {
    setError("")
    try {
      const response = await listGroupMemberLists(groupID)
      setItems(response.items)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "加载成员名单失败")
    } finally {
      setLoading(false)
    }
  }, [groupID])

  useEffect(() => { void refresh() }, [refresh])

  async function submit() {
    const id = Number(userID)
    if (!Number.isSafeInteger(id) || id <= 0) {
      setError("Telegram 用户 ID 必须是正整数")
      return
    }
    setSaving(true)
    setError("")
    try {
      await upsertGroupMemberList(groupID, { telegram_user_id: id, list_type: listType, reason: reason.trim() })
      setUserID("")
      setReason("")
      await refresh()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "保存成员名单失败")
    } finally {
      setSaving(false)
    }
  }

  async function remove(item: TelegramGroupMemberListEntry) {
    if (!window.confirm(`确认从“${listLabels[item.list_type]}”中移除用户 ${item.telegram_user_id}？`)) return
    setError("")
    try {
      await deleteGroupMemberList(groupID, item.telegram_user_id, item.list_type)
      await refresh()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "移除名单项目失败")
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
          <Button variant="outline" onClick={() => void refresh()}><RefreshCw className="mr-2 size-4" />刷新</Button>
        </div>
        {error && <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>}
        <div className="grid gap-6 lg:grid-cols-[0.85fr_1.15fr]">
          <Card>
            <CardHeader><CardTitle>添加名单成员</CardTitle><CardDescription>按 Telegram 用户 ID 管理名单，同一用户可以同时存在于不同名单。</CardDescription></CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2"><Label htmlFor="member-list-user-id">Telegram 用户 ID</Label><Input id="member-list-user-id" inputMode="numeric" value={userID} onChange={(event) => setUserID(event.target.value)} placeholder="例如 123456789" /></div>
              <div className="space-y-2"><Label>名单类型</Label><Select value={listType} onValueChange={(value) => setListType(value as TelegramGroupMemberListEntry["list_type"])}><SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectItem value="blacklist">黑名单</SelectItem><SelectItem value="whitelist">白名单</SelectItem><SelectItem value="exempt">验证豁免</SelectItem></SelectContent></Select></div>
              <div className="space-y-2"><Label htmlFor="member-list-reason">备注原因</Label><Input id="member-list-reason" value={reason} onChange={(event) => setReason(event.target.value)} maxLength={500} placeholder="可选，最多 500 字" /></div>
              <Button onClick={() => void submit()} disabled={saving || !userID.trim()}><ShieldCheck className="mr-2 size-4" />{saving ? "保存中..." : "添加到名单"}</Button>
            </CardContent>
          </Card>
          <Card>
            <CardHeader><CardTitle>当前名单</CardTitle><CardDescription>最多展示最近更新的 500 条名单记录。</CardDescription></CardHeader>
            <CardContent className="space-y-3">
              {loading ? <p className="py-8 text-center text-sm text-muted-foreground">正在加载...</p> : items.length === 0 ? <p className="py-8 text-center text-sm text-muted-foreground">当前群组没有名单记录。</p> : items.map((item) => (
                <div key={`${item.telegram_user_id}:${item.list_type}`} className="flex flex-col gap-3 rounded-lg border p-3 sm:flex-row sm:items-center">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2"><span className="font-mono text-sm">{item.telegram_user_id}</span><Badge variant={item.list_type === "blacklist" ? "destructive" : "secondary"}>{listLabels[item.list_type]}</Badge></div>
                    <p className="mt-1 text-xs text-muted-foreground">{item.reason || "未填写备注"} · {new Date(item.created_at).toLocaleString()}</p>
                  </div>
                  <Button size="sm" variant="outline" onClick={() => void remove(item)}><Trash2 className="mr-2 size-4" />移除</Button>
                </div>
              ))}
            </CardContent>
          </Card>
        </div>
      </div>
    </main>
  )
}
