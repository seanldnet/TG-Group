import { useCallback, useEffect, useMemo, useState } from "react"
import { ArrowLeft, Plus, RefreshCw, Save, Trash2 } from "lucide-react"

import {
  createGroupRule,
  deleteGroupRule,
  listGroupRules,
  updateGroupRule,
  type TelegramGroupRule,
  type TelegramGroupRuleInput,
} from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"

const ruleTypes = [
  { value: "keyword_filter", label: "关键词过滤", config: { words: ["广告"], action: "delete" } },
  { value: "sensitive_word", label: "敏感词过滤", config: { words: ["敏感词"], action: "delete" } },
  { value: "anti_spam", label: "反刷屏", config: { max_messages: 5, window_seconds: 10, duplicate_count: 3, levels: [], action: "delete" } },
  { value: "anti_ad", label: "反广告", config: { patterns: ["t.me/"], exceptions: [], blocked_domains: [], allowed_domains: [], block_invite_links: false, action: "delete" } },
  { value: "anti_bot", label: "群组保护（防机器人）", config: { allowed_usernames: [], action: "ban" } },
  { value: "anti_invite_flood", label: "群组保护（批量邀请防护）", config: { max_members: 8, window_seconds: 60, action: "mute", duration_minutes: 30, allowed_inviter_ids: [] } },
  { value: "media_restriction", label: "消息与媒体限制", config: { types: ["photo", "video", "animation", "sticker", "voice", "audio", "document", "video_note", "poll", "contact", "location", "venue"], action: "delete" } },
  { value: "welcome", label: "入群欢迎", config: { text: "欢迎加入本群！" } },
  { value: "leave_notice", label: "离群通知", config: { text: "有成员离开了群组" } },
  { value: "qa", label: "自动问答", config: { triggers: ["群规"], match_mode: "contains", response: "请查看群公告中的群规。", case_sensitive: false } },
  { value: "join_verification", label: "入群验证（数学题）", config: { challenge_type: "math", timeout_seconds: 120, failure_action: "kick", required_channels: [] } },
  { value: "warning_escalation", label: "警告递进处罚", config: { levels: [{ threshold: 3, action: "mute", duration_minutes: 30 }, { threshold: 5, action: "ban" }] } },
] as const

type RuleType = TelegramGroupRule["rule_type"]

function defaultConfig(type: RuleType) {
  return ruleTypes.find((item) => item.value === type)?.config ?? {}
}

function ruleLabel(type: RuleType) {
  return ruleTypes.find((item) => item.value === type)?.label ?? type
}

export default function GroupRulesPage({ groupID }: { groupID: string }) {
  const [items, setItems] = useState<TelegramGroupRule[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const [editingID, setEditingID] = useState("")
  const [ruleType, setRuleType] = useState<RuleType>("keyword_filter")
  const [name, setName] = useState("关键词过滤")
  const [priority, setPriority] = useState("100")
  const [enabled, setEnabled] = useState(true)
  const [configText, setConfigText] = useState(JSON.stringify(defaultConfig("keyword_filter"), null, 2))

  const refresh = useCallback(async () => {
    setError("")
    try {
      const response = await listGroupRules(groupID)
      setItems(response.items)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "加载规则失败")
    } finally {
      setLoading(false)
    }
  }, [groupID])

  useEffect(() => { void refresh() }, [refresh])

  const editing = useMemo(() => items.find((item) => item.id === editingID), [items, editingID])

  function resetForm() {
    setEditingID("")
    setRuleType("keyword_filter")
    setName("关键词过滤")
    setPriority("100")
    setEnabled(true)
    setConfigText(JSON.stringify(defaultConfig("keyword_filter"), null, 2))
  }

  function startEdit(item: TelegramGroupRule) {
    setEditingID(item.id)
    setRuleType(item.rule_type)
    setName(item.name)
    setPriority(String(item.priority))
    setEnabled(item.enabled)
    setConfigText(JSON.stringify(item.config, null, 2))
  }

  function changeType(value: string) {
    const next = value as RuleType
    setRuleType(next)
    setName(ruleLabel(next))
    setConfigText(JSON.stringify(defaultConfig(next), null, 2))
  }

  async function submit() {
    setError("")
    let config: Record<string, unknown>
    try {
      config = JSON.parse(configText) as Record<string, unknown>
      if (!config || Array.isArray(config) || typeof config !== "object") throw new Error("配置必须是 JSON 对象")
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "规则配置 JSON 格式错误")
      return
    }
    const input: TelegramGroupRuleInput = {
      rule_type: ruleType,
      name: name.trim(),
      enabled,
      priority: Number(priority),
      config,
    }
    setSaving(true)
    try {
      if (editingID) await updateGroupRule(groupID, editingID, input)
      else await createGroupRule(groupID, input)
      resetForm()
      await refresh()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "保存规则失败")
    } finally {
      setSaving(false)
    }
  }

  async function removeRule(item: TelegramGroupRule) {
    if (!window.confirm(`确定删除规则“${item.name}”吗？`)) return
    try {
      await deleteGroupRule(groupID, item.id)
      if (editingID === item.id) resetForm()
      await refresh()
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "删除规则失败")
    }
  }

  return (
    <main className="w-full"><a href="/dashboard/groups" className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"><ArrowLeft className="size-4"/>返回群组列表</a>
      <div className="mx-auto w-full max-w-6xl space-y-6 p-4 sm:p-6 lg:p-8">
        <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-end">
          <div>
            
            <h1 className="text-2xl font-semibold tracking-tight sm:text-[28px]">群组规则</h1>
            
          </div>
          <div className="flex flex-wrap gap-2"><Button asChild variant="outline"><a href={`/dashboard/groups/${groupID}/rule-events`}>执行记录</a></Button><Button variant="outline" onClick={() => void refresh()}><RefreshCw className="mr-2 size-4" />刷新规则</Button></div>
        </div>

        {error && <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>}

        <div className="grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
          <Card>
            <CardHeader>
              <CardTitle>已配置规则</CardTitle>
              <CardDescription>共 {items.length} 条规则</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {loading ? <p className="py-8 text-center text-sm text-muted-foreground">正在加载...</p> : items.length === 0 ? (
                <p className="py-8 text-center text-sm text-muted-foreground">还没有规则，使用右侧表单创建第一条规则。</p>
              ) : items.map((item) => (
                <div key={item.id} className="flex flex-col gap-3 rounded-lg border p-4 sm:flex-row sm:items-center">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium">{item.name}</span>
                      <Badge variant="outline">{ruleLabel(item.rule_type)}</Badge>
                      <Badge variant={item.enabled ? "secondary" : "outline"}>{item.enabled ? "已启用" : "已停用"}</Badge>
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">优先级：{item.priority} · 更新于 {new Date(item.updated_at).toLocaleString()}</p>
                  </div>
                  <div className="flex shrink-0 gap-2">
                    <Button size="sm" variant="outline" onClick={() => startEdit(item)}>编辑</Button>
                    <Button size="sm" variant="ghost" onClick={() => void removeRule(item)}><Trash2 className="size-4" /></Button>
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{editing ? "编辑规则" : "创建规则"}</CardTitle>
              
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label>规则类型</Label>
                <Select value={ruleType} onValueChange={changeType} disabled={Boolean(editingID)}>
                  <SelectTrigger><SelectValue placeholder="选择规则类型" /></SelectTrigger>
                  <SelectContent>{ruleTypes.map((item) => <SelectItem key={item.value} value={item.value}>{item.label}</SelectItem>)}</SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="rule-name">规则名称</Label>
                <Input id="rule-name" value={name} onChange={(event) => setName(event.target.value)} maxLength={80} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="rule-priority">优先级（0–1000）</Label>
                <Input id="rule-priority" type="number" min={0} max={1000} value={priority} onChange={(event) => setPriority(event.target.value)} />
              </div>
              <div className="flex items-center justify-between rounded-lg border px-3 py-2">
                <Label htmlFor="rule-enabled">启用规则</Label>
                <Switch id="rule-enabled" checked={enabled} onCheckedChange={setEnabled} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="rule-config">规则内容</Label>
                <Textarea id="rule-config" value={configText} onChange={(event) => setConfigText(event.target.value)} rows={10} className="font-mono text-xs" />
                <p className="text-xs text-muted-foreground">填写规则内容并保存即可生效。</p>
              </div>
              <div className="flex gap-2">
                <Button onClick={() => void submit()} disabled={saving || !name.trim()}><Save className="mr-2 size-4" />{saving ? "保存中..." : editing ? "保存修改" : <><Plus className="mr-2 size-4" />创建规则</>}</Button>
                {editing && <Button variant="outline" onClick={resetForm}>取消编辑</Button>}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </main>
  )
}
