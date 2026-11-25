import { useCallback, useEffect, useState } from "react"
import { ArrowLeft, RefreshCw, UsersRound } from "lucide-react"

import { listGroupMembers, type TelegramGroupMember } from "@/lib/api"
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
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState("")

  const refresh = useCallback(async (manual = false) => {
    setError("")
    if (manual) setRefreshing(true)
    try {
      const response = await listGroupMembers(groupID, 200)
      setItems(response.items)
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

  return (
    <main className="min-h-svh bg-muted/25">
      <header className="border-b border-border/70 bg-background">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-5 sm:px-8">
          <a href="/dashboard/groups" className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground">
            <ArrowLeft className="size-4" />
            返回群组列表
          </a>
          <span className="text-sm font-semibold">TG-Group</span>
        </div>
      </header>

      <div className="mx-auto max-w-6xl space-y-6 px-5 py-9 sm:px-8 sm:py-12">
        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
          <div>
            <Badge variant="secondary" className="mb-3">成员管理</Badge>
            <h1 className="text-3xl font-semibold tracking-tight">群成员</h1>
            <p className="mt-2 text-sm text-muted-foreground">群组 ID：<span className="font-mono">{groupID}</span></p>
          </div>
          <Button variant="outline" onClick={() => void refresh(true)} disabled={refreshing}>
            <RefreshCw className={`mr-2 size-4 ${refreshing ? "animate-spin" : ""}`} />
            刷新成员
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
                <CardTitle>已记录成员</CardTitle>
                <CardDescription>当前展示最近活动的最多 200 位成员；状态来自 Telegram 成员更新。</CardDescription>
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
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((member) => {
                    const name = [member.first_name, member.last_name].filter(Boolean).join(" ") || "未设置名称"
                    const permissions = [
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
                        <TableCell className="text-xs text-muted-foreground">{permissions.length ? permissions.join(" · ") : "—"}</TableCell>
                        <TableCell className="whitespace-nowrap text-xs text-muted-foreground">{member.joined_at ? new Date(member.joined_at).toLocaleString() : "—"}</TableCell>
                        <TableCell className="whitespace-nowrap text-xs text-muted-foreground">{new Date(member.last_seen_at).toLocaleString()}</TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            )}
          </CardContent>
        </Card>
      </div>
    </main>
  )
}
