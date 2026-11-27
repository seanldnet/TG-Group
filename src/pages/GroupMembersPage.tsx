import { useCallback, useEffect, useState } from "react"
import { AlertTriangle, ArrowLeft, RefreshCw, UsersRound } from "lucide-react"

import { addGroupMemberWarning, clearGroupMemberWarnings, getGroupBotPermissions, listGroupMemberWarnings, listGroupMembers, moderateGroupMember, listWarningEscalations, retryWarningEscalation, type TelegramWarningEscalationExecution, type TelegramGroupBotPermissions, type TelegramGroupMember, type TelegramGroupMemberWarning } from "@/lib/api"
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

function memberStatusLabel(status: TelegramGroupMember["status"]) {
  const labels: Record<TelegramGroupMember["status"], string> = {
    member: "普通成员",
    administrator: "管理员",
    creator: "群主",
    restricted: "受限成员",
    left: "已离开",
    kicked: "已封禁",
  }
  return labels[status]
}

export default function GroupMembersPage({ groupID }: { groupID: string }) {
  const [items, setItems] = useState<TelegramGroupMember[]>([])
  const [page, setPage] = useState(1)
  const [memberTotal, setMemberTotal] = useState(0)
  const pageSize = 20
  const pageCount = Math.max(1, Math.ceil(memberTotal / pageSize))
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState("")
  const [permissions, setPermissions] = useState<TelegramGroupBotPermissions | null>(null)
  const [actingUserID, setActingUserID] = useState<number | null>(null)
  const [warningUserID, setWarningUserID] = useState<number | null>(null)
  const [warnings, setWarnings] = useState<TelegramGroupMemberWarning[]>([])
  const [warningTotal, setWarningTotal] = useState(0)
  const [warningPage, setWarningPage] = useState(1)
  const warningPageSize = 10
  const warningPageCount = Math.max(1, Math.ceil(warningTotal / warningPageSize))
  const [warningBusy, setWarningBusy] = useState(false)
  const [escalations, setEscalations] = useState<TelegramWarningEscalationExecution[]>([])
  const [escalationTotal, setEscalationTotal] = useState(0)
  const [escalationPage, setEscalationPage] = useState(1)
  const escalationPageSize = 10
  const escalationPageCount = Math.max(1, Math.ceil(escalationTotal / escalationPageSize))

  const refresh = useCallback(async (manual = false, targetPage = 1) => {
    setError("")
    if (manual) setRefreshing(true)
    try {
      const [response, permissionResult] = await Promise.all([listGroupMembers(groupID, pageSize, (targetPage - 1) * pageSize), getGroupBotPermissions(groupID)])
      setItems(response.items)
      setMemberTotal(response.total)
      setPage(targetPage)
      setPermissions(permissionResult)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "加载群成员失败")
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [groupID])

  useEffect(() => {
    void refresh()
  }, [refresh])

  const openWarnings = async (userID: number) => {
    setWarningUserID(userID)
    setWarningBusy(true)
    setError("")
    try {
      const [response, executions] = await Promise.all([listGroupMemberWarnings(groupID, userID, warningPageSize, 0), listWarningEscalations(groupID, userID, escalationPageSize, 0)])
      setWarnings(response.items)
      setWarningTotal(response.total)
      setWarningPage(1)
      setEscalations(executions.items)
      setEscalationTotal(executions.total)
      setEscalationTotal(executions.total)
      setEscalationPage(1)
      setEscalationTotal(executions.total)
      setEscalationPage(1)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "加载警告记录失败")
      setWarnings([])
    } finally {
      setWarningBusy(false)
    }
  }

  const addWarning = async () => {
    if (warningUserID === null) return
    const reason = window.prompt("请输入本次警告原因", "违反群规")
    if (reason === null) return
    if (!reason.trim()) { setError("警告原因不能为空"); return }
    setWarningBusy(true)
    setError("")
    try {
      await addGroupMemberWarning(groupID, warningUserID, reason.trim())
      const [response, executions] = await Promise.all([listGroupMemberWarnings(groupID, warningUserID, warningPageSize, 0), listWarningEscalations(groupID, warningUserID, escalationPageSize, 0)])
      setWarnings(response.items)
      setWarningTotal(response.total)
      setWarningPage(1)
      setEscalations(executions.items)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "添加警告失败")
    } finally {
      setWarningBusy(false)
    }
  }

  const retryEscalation = async (execution: TelegramWarningEscalationExecution) => {
    if (warningUserID === null || !window.confirm("确认重新执行这条失败的自动处罚？服务端会再次检查 Bot 权限及目标成员身份。")) return
    setWarningBusy(true)
    setError("")
    try {
      await retryWarningEscalation(groupID, warningUserID, execution.id)
      const [response, executions] = await Promise.all([listGroupMemberWarnings(groupID, warningUserID), listWarningEscalations(groupID, warningUserID, escalationPageSize, (escalationPage - 1) * escalationPageSize)])
      setWarnings(response.items)
      setWarningTotal(response.total)
      setEscalations(executions.items)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "重试自动处罚失败")
      try { const executions = await listWarningEscalations(groupID, warningUserID, escalationPageSize, (escalationPage - 1) * escalationPageSize); setEscalations(executions.items); setEscalationTotal(executions.total) } catch { /* preserve original error */ }
    } finally { setWarningBusy(false) }
  }

  const clearWarnings = async () => {
    if (warningUserID === null || !window.confirm("确认清除此成员在当前群组的全部警告记录？此操作不可恢复。")) return
    setWarningBusy(true)
    setError("")
    try {
      await clearGroupMemberWarnings(groupID, warningUserID)
      setWarnings([])
      setWarningTotal(0)
      setWarningPage(1)
      setEscalations([])
      setEscalationTotal(0)
      setEscalationPage(1)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "清零警告失败")
    } finally {
      setWarningBusy(false)
    }
  }

  const performAction = async (member: TelegramGroupMember, action: "ban" | "unban" | "mute" | "unmute") => {
    const labels = { ban: "封禁", unban: "解封", mute: "禁言", unmute: "解除禁言" }
    const name = [member.first_name, member.last_name].filter(Boolean).join(" ") || String(member.telegram_user_id)
    if (!window.confirm("确认对 " + name + " 执行" + labels[action] + "操作？")) return
    let duration = 0
    if (action === "mute") {
      const raw = window.prompt("请输入禁言时长（分钟，1-525600）", "60")
      if (raw === null) return
      duration = Number(raw)
      if (!Number.isInteger(duration) || duration < 1 || duration > 525600) { setError("禁言时长必须是 1 到 525600 之间的整数分钟"); return }
    }
    setActingUserID(member.telegram_user_id)
    setError("")
    try { await moderateGroupMember(groupID, member.telegram_user_id, action, duration); await refresh() }
    catch (cause) { setError(cause instanceof Error ? cause.message : "成员操作失败") }
    finally { setActingUserID(null) }
  }
  return (
    <main className="w-full"><a href="/dashboard/groups" className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"><ArrowLeft className="size-4"/>返回群组列表</a>
      <div className="mx-auto w-full max-w-6xl space-y-6 p-4 sm:p-6 lg:p-8">
        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
          <div>
            <Badge variant="secondary" className="mb-3">成员管理</Badge>
            <h1 className="text-3xl font-semibold tracking-tight">群成员</h1>
            <p className="mt-2 text-sm text-muted-foreground">群组 ID：<span className="font-mono">{groupID}</span></p>
          </div>
          <Button variant="outline" asChild><a href={`/dashboard/groups/${groupID}/member-lists`}>黑白名单</a></Button>
          <Button variant="outline" onClick={() => void refresh(true)} disabled={refreshing}>
            <RefreshCw className={`mr-2 size-4 ${refreshing ? "animate-spin" : ""}`} />
            刷新成员
          </Button>
        </div>

        {permissions && (
          <Card>
            <CardHeader><CardTitle className="text-base">Bot 群组权限（实时检测）</CardTitle><CardDescription>Bot 当前身份：{permissions.bot_status}</CardDescription></CardHeader>
            <CardContent className="flex flex-wrap gap-2">
              <Badge variant={permissions.can_restrict_members ? "default" : "destructive"}>成员限制：{permissions.can_restrict_members ? "具备" : "缺失"}</Badge>
              <Badge variant={permissions.can_delete_messages ? "secondary" : "outline"}>删除消息：{permissions.can_delete_messages ? "具备" : "缺失"}</Badge>
              <Badge variant={permissions.can_invite_users ? "secondary" : "outline"}>邀请成员：{permissions.can_invite_users ? "具备" : "缺失"}</Badge>
              <Badge variant={permissions.can_manage_topics ? "secondary" : "outline"}>管理话题：{permissions.can_manage_topics ? "具备" : "缺失"}</Badge>
            </CardContent>
          </Card>
        )}
        {permissions && (!["administrator", "creator"].includes(permissions.bot_status) || !permissions.can_restrict_members) && <p className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">Bot 需要群管理员身份及限制成员权限才能执行处罚。</p>}
        {error && <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>}

        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="flex size-10 items-center justify-center rounded-xl bg-primary/8 text-primary">
                <UsersRound className="size-5" />
              </div>
              <div>
                <CardTitle>已记录成员</CardTitle>
                <CardDescription>按最近活动排序，通过服务端分页读取已采集成员，每页 20 位。状态来自 Telegram 成员更新；Telegram Bot API 不提供任意群组全量成员枚举接口。</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {loading ? (
              <p className="py-10 text-center text-sm text-muted-foreground">正在加载群成员...</p>
            ) : items.length === 0 ? (
              <div className="py-12 text-center">
                <p className="font-medium">暂未收到成员资料</p>
                <p className="mt-1 text-sm text-muted-foreground">请确认 Bot 已加入群组并具有接收成员状态更新所需的管理员权限。</p>
              </div>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>成员</TableHead>
                    <TableHead>Telegram 用户 ID</TableHead>
                    <TableHead>身份</TableHead>
                    <TableHead>权限</TableHead>
                    <TableHead>加入时间</TableHead>
                    <TableHead>最近活动</TableHead>
                    <TableHead>管理操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((member) => {
                    const name = [member.first_name, member.last_name].filter(Boolean).join(" ") || "未设置名称"
                    const memberPermissions = [
                      member.can_delete_messages && "删消息",
                      member.can_restrict_members && "限制成员",
                      member.can_invite_users && "邀请成员",
                      member.can_manage_topics && "管理话题",
                    ].filter(Boolean)
                    return (
                      <TableRow key={member.id}>
                        <TableCell>
                          <div className="font-medium">{name}</div>
                          <div className="text-xs text-muted-foreground">{member.username ? `@${member.username}` : member.is_bot ? "机器人账号" : "无用户名"}</div>
                        </TableCell>
                        <TableCell className="font-mono text-xs">{member.telegram_user_id}</TableCell>
                        <TableCell><Badge variant={member.status === "administrator" || member.status === "creator" ? "default" : "outline"}>{memberStatusLabel(member.status)}</Badge></TableCell>
                        <TableCell className="text-xs text-muted-foreground">{memberPermissions.length ? memberPermissions.join(" · ") : "—"}</TableCell>
                        <TableCell className="whitespace-nowrap text-xs text-muted-foreground">{member.joined_at ? new Date(member.joined_at).toLocaleString() : "—"}</TableCell>
                        <TableCell className="whitespace-nowrap text-xs text-muted-foreground">{new Date(member.last_seen_at).toLocaleString()}</TableCell>
                        <TableCell><div className="flex flex-wrap gap-1"><Button size="sm" variant="outline" onClick={() => void openWarnings(member.telegram_user_id)}><AlertTriangle className="mr-1 size-3" />警告记录</Button>{member.status === "kicked" ? <Button size="sm" variant="outline" disabled={actingUserID === member.telegram_user_id || !permissions?.can_restrict_members} onClick={() => void performAction(member, "unban")}>解封</Button> : member.status === "restricted" ? <Button size="sm" variant="outline" disabled={actingUserID === member.telegram_user_id || !permissions?.can_restrict_members} onClick={() => void performAction(member, "unmute")}>解除禁言</Button> : member.status === "member" ? <><Button size="sm" variant="outline" disabled={actingUserID === member.telegram_user_id || !permissions?.can_restrict_members} onClick={() => void performAction(member, "mute")}>禁言</Button><Button size="sm" variant="destructive" disabled={actingUserID === member.telegram_user_id || !permissions?.can_restrict_members} onClick={() => void performAction(member, "ban")}>封禁</Button></> : <span className="text-xs text-muted-foreground">不可操作</span>}</div></TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            )}
            {!loading && items.length > 0 && (
              <div className="mt-4 flex flex-col gap-3 border-t pt-4 sm:flex-row sm:items-center sm:justify-between">
                <p className="text-sm text-muted-foreground">
                  共 {memberTotal} 位已采集成员 · 第 {page} / {pageCount} 页
                </p>
                <div className="flex items-center gap-2">
                  <Button size="sm" variant="outline" disabled={page <= 1} onClick={() => void refresh(false, page - 1)}>上一页</Button>
                  <Button size="sm" variant="outline" disabled={page >= pageCount} onClick={() => void refresh(false, page + 1)}>下一页</Button>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
        {warningUserID !== null && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">成员警告记录</CardTitle>
              <CardDescription>Telegram 用户 ID：{warningUserID}。累计 {warningTotal} 次警告，当前展示最近 {warnings.length} 条。达到已启用递进策略阈值时会自动执行处罚；失败记录可在此手动重试。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap gap-2">
                <Button size="sm" onClick={() => void addWarning()} disabled={warningBusy}>添加警告</Button>
                <Button size="sm" variant="destructive" onClick={() => void clearWarnings()} disabled={warningBusy || warnings.length === 0}>清零全部警告</Button>
                <Button size="sm" variant="outline" onClick={() => setWarningUserID(null)}>关闭</Button>
              </div>
              <div className="space-y-2">
                <p className="text-sm font-medium">递进处罚执行记录</p>
                {escalations.length === 0 ? <p className="text-sm text-muted-foreground">暂无自动处罚执行记录。</p> : escalations.map((execution) => (
                  <div key={execution.id} className="flex flex-col gap-2 rounded-lg border p-3 sm:flex-row sm:items-center sm:justify-between">
                    <div className="min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <Badge variant={execution.status === "succeeded" ? "default" : execution.status === "failed" ? "destructive" : "secondary"}>{execution.status === "succeeded" ? "成功" : execution.status === "failed" ? "失败" : execution.status === "processing" ? "处理中" : "已跳过"}</Badge>
                        <span className="text-sm font-medium">{execution.action === "mute" ? `禁言 ${execution.duration_minutes} 分钟` : "封禁"} · 第 {execution.warning_count} 次警告</span>
                      </div>
                      <p className="mt-1 break-words text-xs text-muted-foreground">{execution.detail || "—"}</p>
                      <time className="text-xs text-muted-foreground">{new Date(execution.updated_at).toLocaleString()}</time>
                    </div>
                    {execution.status === "failed" && <Button size="sm" variant="outline" disabled={warningBusy || !permissions?.can_restrict_members} onClick={() => void retryEscalation(execution)}>重试处罚</Button>}
                  </div>
                ))}
                {escalationTotal > 0 && (
                  <div className="flex items-center justify-between gap-3 text-sm text-muted-foreground">
                    <span>执行记录第 {escalationPage} / {escalationPageCount} 页，共 {escalationTotal} 条</span>
                    <div className="flex gap-2">
                      <Button size="sm" variant="outline" disabled={warningBusy || escalationPage <= 1} onClick={async () => {
                        if (warningUserID === null) return
                        const nextPage = escalationPage - 1
                        setWarningBusy(true)
                        try { const response = await listWarningEscalations(groupID, warningUserID, escalationPageSize, (nextPage - 1) * escalationPageSize); setEscalations(response.items); setEscalationTotal(response.total); setEscalationPage(nextPage) }
                        catch (cause) { setError(cause instanceof Error ? cause.message : "读取处罚执行记录失败") }
                        finally { setWarningBusy(false) }
                      }}>上一页</Button>
                      <Button size="sm" variant="outline" disabled={warningBusy || escalationPage >= escalationPageCount} onClick={async () => {
                        if (warningUserID === null) return
                        const nextPage = escalationPage + 1
                        setWarningBusy(true)
                        try { const response = await listWarningEscalations(groupID, warningUserID, escalationPageSize, (nextPage - 1) * escalationPageSize); setEscalations(response.items); setEscalationTotal(response.total); setEscalationPage(nextPage) }
                        catch (cause) { setError(cause instanceof Error ? cause.message : "读取处罚执行记录失败") }
                        finally { setWarningBusy(false) }
                      }}>下一页</Button>
                    </div>
                  </div>
                )}
              </div>
              {warnings.length > 0 && (
                <div className="flex items-center justify-between gap-3 text-sm text-muted-foreground">
                  <span>警告记录第 {warningPage} / {warningPageCount} 页，共 {warningTotal} 条</span>
                  <div className="flex gap-2">
                    <Button size="sm" variant="outline" disabled={warningBusy || warningPage <= 1} onClick={async () => {
                      if (warningUserID === null) return
                      const nextPage = warningPage - 1
                      setWarningBusy(true)
                      try { const response = await listGroupMemberWarnings(groupID, warningUserID, warningPageSize, (nextPage - 1) * warningPageSize); setWarnings(response.items); setWarningPage(nextPage) }
                      catch (cause) { setError(cause instanceof Error ? cause.message : "读取警告记录失败") }
                      finally { setWarningBusy(false) }
                    }}>上一页</Button>
                    <Button size="sm" variant="outline" disabled={warningBusy || warningPage >= warningPageCount} onClick={async () => {
                      if (warningUserID === null) return
                      const nextPage = warningPage + 1
                      setWarningBusy(true)
                      try { const response = await listGroupMemberWarnings(groupID, warningUserID, warningPageSize, (nextPage - 1) * warningPageSize); setWarnings(response.items); setWarningPage(nextPage) }
                      catch (cause) { setError(cause instanceof Error ? cause.message : "读取警告记录失败") }
                      finally { setWarningBusy(false) }
                    }}>下一页</Button>
                  </div>
                </div>
              )}
              {warningBusy ? <p className="text-sm text-muted-foreground">正在处理...</p> : warnings.length === 0 ? (
                <p className="text-sm text-muted-foreground">该成员在当前群组暂无警告记录。</p>
              ) : (
                <div className="space-y-2">
                  {warnings.map((warning) => (
                    <div key={warning.id} className="flex flex-col gap-1 rounded-lg border p-3 sm:flex-row sm:items-center sm:justify-between">
                      <div><p className="text-sm font-medium">{warning.reason}</p><p className="text-xs text-muted-foreground">记录 #{warning.id} · {warning.source === "automatic" ? "自动记录" : "管理员记录"}</p></div>
                      <time className="text-xs text-muted-foreground">{new Date(warning.created_at).toLocaleString()}</time>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        )}
      </div>
    </main>
  )
}
