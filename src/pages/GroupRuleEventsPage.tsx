import { useCallback, useEffect, useState } from "react"
import { ArrowLeft, RefreshCw } from "lucide-react"

import { listGroupRuleEvents, type TelegramGroupRuleEvent } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

const actionLabels: Record<string, string> = {
  delete_message: "删除消息",
  welcome: "入群欢迎",
  leave_notice: "离群通知",
}

function statusVariant(status: TelegramGroupRuleEvent["status"]) {
  if (status === "succeeded") return "secondary" as const
  if (status === "failed") return "destructive" as const
  return "outline" as const
}

function statusLabel(status: TelegramGroupRuleEvent["status"]) {
  if (status === "succeeded") return "成功"
  if (status === "failed") return "失败"
  return "已跳过"
}

export default function GroupRuleEventsPage({ groupID }: { groupID: string }) {
  const [items, setItems] = useState<TelegramGroupRuleEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")

  const refresh = useCallback(async () => {
    setError("")
    try {
      const response = await listGroupRuleEvents(groupID, 100)
      setItems(response.items)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "加载规则执行记录失败")
    } finally {
      setLoading(false)
    }
  }, [groupID])

  useEffect(() => { void refresh() }, [refresh])

  return (
    <main className="w-full"><a href={`/dashboard/groups/${groupID}/rules`} className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"><ArrowLeft className="size-4"/>返回规则管理</a>
      <div className="mx-auto w-full max-w-6xl space-y-6 p-4 sm:p-6 lg:p-8">
        <div className="flex flex-col justify-between gap-3 sm:flex-row sm:items-end">
          <div>
            <Badge variant="secondary" className="mb-3">自动化审计</Badge>
            <h1 className="text-3xl font-semibold tracking-tight">规则执行记录</h1>
            <p className="mt-2 text-sm text-muted-foreground">展示当前群组最近 100 条规则动作及 Telegram 执行结果。</p>
          </div>
          <Button variant="outline" onClick={() => void refresh()}><RefreshCw className="mr-2 size-4" />刷新记录</Button>
        </div>

        {error && <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>}

        <Card>
          <CardHeader>
            <CardTitle>执行历史</CardTitle>
            <CardDescription>包含删除消息、入群欢迎和离群通知；失败或跳过时会显示原因。</CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? <p className="py-10 text-center text-sm text-muted-foreground">正在加载...</p> : items.length === 0 ? (
              <p className="py-10 text-center text-sm text-muted-foreground">暂时没有规则执行记录。请确认规则已启用，且 Bot 能收到群组更新。</p>
            ) : (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>执行时间</TableHead>
                      <TableHead>动作</TableHead>
                      <TableHead>状态</TableHead>
                      <TableHead>Telegram 用户 ID</TableHead>
                      <TableHead>消息 ID</TableHead>
                      <TableHead>执行详情</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {items.map((item) => (
                      <TableRow key={item.id}>
                        <TableCell className="whitespace-nowrap text-xs">{new Date(item.created_at).toLocaleString()}</TableCell>
                        <TableCell className="whitespace-nowrap">{actionLabels[item.action] ?? item.action}</TableCell>
                        <TableCell><Badge variant={statusVariant(item.status)}>{statusLabel(item.status)}</Badge></TableCell>
                        <TableCell className="font-mono text-xs">{item.telegram_user_id || "—"}</TableCell>
                        <TableCell className="font-mono text-xs">{item.telegram_message_id || "—"}</TableCell>
                        <TableCell className="min-w-56 text-xs text-muted-foreground">{item.detail || "—"}</TableCell>
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
