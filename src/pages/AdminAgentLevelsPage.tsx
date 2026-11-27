import { useCallback, useEffect, useState, type FormEvent } from "react"
import { AlertCircle, Check, Pencil, Plus, RefreshCw } from "lucide-react"
import { createAdminAgentLevel, listAdminAgentLevels, updateAdminAgentLevel, type AgentLevel } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"

const emptyForm = { code: "", name: "", price: "0", sortOrder: "0", benefits: "{\n  \"description\": \"\"\n}", enabled: true }

export default function AdminAgentLevelsPage() {
  const [items, setItems] = useState<AgentLevel[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const [notice, setNotice] = useState("")
  const [editingID, setEditingID] = useState("")
  const [form, setForm] = useState(emptyForm)

  const load = useCallback(async () => {
    setLoading(true)
    setError("")
    try {
      const result = await listAdminAgentLevels()
      setItems(result.items)
    } catch (err) {
      setError(err instanceof Error ? err.message : "读取代理等级失败")
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void load() }, [load])

  function startEdit(item: AgentLevel) {
    setEditingID(item.id)
    setForm({
      code: item.code,
      name: item.name,
      price: String(item.price_usdt),
      sortOrder: String(item.sort_order),
      benefits: JSON.stringify(item.benefits ?? {}, null, 2),
      enabled: item.enabled,
    })
    setNotice("")
  }

  function resetForm(options: { clearNotice?: boolean } = {}) {
    setEditingID("")
    setForm(emptyForm)
    if (options.clearNotice !== false) setNotice("")
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError("")
    setNotice("")
    if (!/^\d+$/.test(form.price) || !/^\d+$/.test(form.sortOrder) || !Number.isSafeInteger(Number(form.price)) || !Number.isSafeInteger(Number(form.sortOrder))) {
      setError("价格和排序必须填写非负整数；价格单位为 USDT。")
      return
    }
    let benefits: Record<string, unknown>
    try {
      const parsed: unknown = JSON.parse(form.benefits || "{}")
      if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error()
      benefits = parsed as Record<string, unknown>
    } catch {
      setError("权益配置必须是有效的 JSON 对象。")
      return
    }
    setSaving(true)
    try {
      if (editingID) {
        await updateAdminAgentLevel(editingID, {
          name: form.name,
          price_usdt: Number(form.price),
          sort_order: Number(form.sortOrder),
          benefits,
          enabled: form.enabled,
        })
        setNotice("代理等级配置已保存。")
      } else {
        await createAdminAgentLevel({
          code: form.code,
          name: form.name,
          price_usdt: Number(form.price),
          sort_order: Number(form.sortOrder),
          benefits,
        })
        setNotice("代理等级已创建。")
      }
      resetForm({ clearNotice: false })
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存代理等级失败")
    } finally {
      setSaving(false)
    }
  }

  return (
    <main className="mx-auto w-full max-w-[1440px] space-y-6 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <div className="mb-2 flex gap-2"><Badge variant="secondary">平台管理</Badge><Badge variant="outline">真实配置</Badge></div>
          <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">代理等级配置</h1>
          <p className="mt-2 text-sm text-muted-foreground">管理员配置等级标识、展示名称、整数 USDT 升级价格、排序和权益。用户在代理中心自主升级，余额支付后自动生效。</p>
        </div>
        <Button variant="outline" onClick={() => void load()} disabled={loading}><RefreshCw className={`mr-2 size-4 ${loading ? "animate-spin" : ""}`}/>刷新</Button>
      </div>

      {error && <div className="flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive" role="alert"><AlertCircle className="mt-0.5 size-4 shrink-0"/>{error}</div>}
      {notice && <p className="rounded-lg border border-emerald-600/20 bg-emerald-600/5 p-3 text-sm" role="status">{notice}</p>}

      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.25fr)_minmax(320px,0.75fr)]">
        <Card>
          <CardHeader><CardTitle className="text-base">等级目录</CardTitle><CardDescription>已配置 {items.length} 个等级；已停用等级不会出现在用户可选目录。</CardDescription></CardHeader>
          <CardContent>
            {loading ? <p className="py-8 text-center text-sm text-muted-foreground">正在读取…</p> : items.length ? (
              <div className="space-y-3">
                {items.map((item) => (
                  <div key={item.id} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-4">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2"><p className="font-medium">{item.name}</p><Badge variant={item.enabled ? "default" : "secondary"}>{item.enabled ? "启用" : "停用"}</Badge></div>
                      <p className="mt-1 text-xs text-muted-foreground">{item.code} · 排序 {item.sort_order}</p>
                      <p className="mt-2 text-lg font-semibold tabular-nums">{new Intl.NumberFormat("en-US").format(item.price_usdt)} <span className="text-xs font-medium text-muted-foreground">USDT</span></p>
                    </div>
                    <Button size="sm" variant="outline" onClick={() => startEdit(item)}><Pencil className="mr-2 size-3.5"/>编辑</Button>
                  </div>
                ))}
              </div>
            ) : <p className="py-8 text-center text-sm text-muted-foreground">还没有代理等级。</p>}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center justify-between gap-3"><div><CardTitle className="text-base">{editingID ? "编辑代理等级" : "新增代理等级"}</CardTitle><CardDescription>价格使用整数 USDT，不接受小数。</CardDescription></div>{editingID ? <Button variant="ghost" size="sm" onClick={() => resetForm()}>取消编辑</Button> : <Plus className="size-4 text-muted-foreground"/>}</div>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={(event) => void submit(event)}>
              <div className="space-y-2"><Label htmlFor="agent-level-code">等级标识</Label><Input id="agent-level-code" value={form.code} disabled={!!editingID} onChange={(e) => setForm((v) => ({...v, code:e.target.value}))} placeholder="例如 silver" required maxLength={40}/></div>
              <div className="space-y-2"><Label htmlFor="agent-level-name">等级名称</Label><Input id="agent-level-name" value={form.name} onChange={(e) => setForm((v) => ({...v, name:e.target.value}))} placeholder="例如白银代理" required maxLength={80}/></div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2"><Label htmlFor="agent-level-price">价格（USDT）</Label><Input id="agent-level-price" inputMode="numeric" value={form.price} onChange={(e) => setForm((v) => ({...v, price:e.target.value}))} required/></div>
                <div className="space-y-2"><Label htmlFor="agent-level-sort">排序</Label><Input id="agent-level-sort" inputMode="numeric" value={form.sortOrder} onChange={(e) => setForm((v) => ({...v, sortOrder:e.target.value}))} required/></div>
              </div>
              <div className="space-y-2"><Label htmlFor="agent-level-benefits">等级权益（JSON 对象）</Label><Textarea id="agent-level-benefits" rows={7} value={form.benefits} onChange={(e) => setForm((v) => ({...v, benefits:e.target.value}))} className="font-mono text-xs"/></div>
              {editingID && <div className="flex items-center justify-between rounded-lg border p-3"><Label htmlFor="agent-level-enabled">开放用户升级选择</Label><Switch id="agent-level-enabled" checked={form.enabled} onCheckedChange={(enabled) => setForm((v) => ({...v, enabled}))}/></div>}
              <Button type="submit" className="w-full" disabled={saving}>{saving ? "保存中…" : editingID ? <><Check className="mr-2 size-4"/>保存配置</> : <><Plus className="mr-2 size-4"/>创建等级</>}</Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </main>
  )
}
