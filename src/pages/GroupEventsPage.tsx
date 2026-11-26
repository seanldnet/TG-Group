import { useCallback, useEffect, useState } from "react"
import { ArrowLeft, RefreshCw } from "lucide-react"

import { listGroupEvents, type TelegramGroupMessageEvent } from "@/lib/api"
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

function messageTypeLabel(type: string) {
  const labels: Record<string, string> = {
    text: "文字",
    photo: "图片",
    document: "文件",
    video: "视频",
    sticker: "贴纸",
    animation: "动画",
    voice: "语音",
    audio: "音频",
    other: "其他消息",
  }
  return labels[type] ?? type
}

export default function GroupEventsPage({ groupID }: { groupID: string }) {
  const [items, setItems] = useState<TelegramGroupMessageEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState("")

  const refresh = useCallback(async (manual = false) => {
    setError("")
    if (manual) setRefreshing(true)
    try {
      const response = await listGroupEvents(groupID, 100)
      setItems(response.items)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "加载消息日志失败")
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [groupID])

  useEffect(() => {
    void refresh()
  }, [refresh])

  return (
    <main className="w-full"><a href="/dashboard/groups" className="inline-flex items-center gap-2 text-sm text-muted-foreground hover:text-foreground"><ArrowLeft className="size-4"/>返回群组列表</a>
      <div className="mx-auto w-full max-w-6xl space-y-6 p-4 sm:p-6 lg:p-8">
        <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
          <div>
            <Badge variant="secondary" className="mb-3">消息审计</Badge>
            <h1 className="text-3xl font-semibold tracking-tight">群消息日志</h1>
            <p className="mt-2 text-sm text-muted-foreground">群组 ID：<span className="font-mono">{groupID}</span></p>
          </div>
          <Button variant="outline" onClick={() => void refresh(true)} disabled={refreshing}>
            <RefreshCw className={`mr-2 size-4 ${refreshing ? "animate-spin" : ""}`} />
            刷新日志
          </Button>
        </div>

        {error && <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>}

        <Card>
          <CardHeader>
            <CardTitle>最近消息</CardTitle>
            <CardDescription>最多显示最近 100 条消息事件，按接收时间倒序排列。</CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? (
              <p className="py-10 text-center text-sm text-muted-foreground">正在加载消息日志...</p>
            ) : items.length === 0 ? (
              <p className="py-12 text-center text-sm text-muted-foreground">暂无消息记录。Bot 收到群组消息后会自动写入日志。</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>发送者</TableHead>
                    <TableHead>Telegram 用户 ID</TableHead>
                    <TableHead>类型</TableHead>
                    <TableHead className="min-w-64">消息内容</TableHead>
                    <TableHead>发送时间</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {items.map((event) => (
                    <TableRow key={event.id}>
                      <TableCell>
                        <div className="font-medium">{event.sender_display_name || "频道 / 匿名发送者"}</div>
                        <div className="text-xs text-muted-foreground">{event.sender_username ? `@${event.sender_username}` : "无用户名"}</div>
                      </TableCell>
                      <TableCell className="font-mono text-xs">{event.telegram_user_id || "—"}</TableCell>
                      <TableCell><Badge variant="outline">{messageTypeLabel(event.message_type)}</Badge></TableCell>
                      <TableCell className="max-w-md whitespace-pre-wrap break-words text-sm">{event.message_text || "（无文字内容）"}</TableCell>
                      <TableCell className="whitespace-nowrap text-xs text-muted-foreground">{event.telegram_sent_at ? new Date(event.telegram_sent_at).toLocaleString() : new Date(event.received_at).toLocaleString()}</TableCell>
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
