import { useCallback, useEffect, useState, type FormEvent } from "react"
import { AlertCircle, ArrowUpRight, CheckCircle2, Clock3, RefreshCw, Wallet } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { createWalletRecharge, listWalletRecharges, refreshWalletRecharge, type WalletRechargeOrder } from "@/lib/api"

const PAGE_SIZE = 10

function formatUSDT(value: number) {
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(value)
}

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function statusLabel(status: WalletRechargeOrder["status"]) {
  const labels: Record<WalletRechargeOrder["status"], string> = {
    pending: "待支付",
    paid: "已到账",
    expired: "已过期",
    failed: "失败",
    review: "待核查",
  }
  return labels[status]
}

function statusVariant(status: WalletRechargeOrder["status"]): "default" | "secondary" | "destructive" | "outline" {
  if (status === "paid") return "default"
  if (status === "failed") return "destructive"
  if (status === "pending") return "outline"
  return "secondary"
}

export default function WalletPaymentsPage() {
  const [orders, setOrders] = useState<WalletRechargeOrder[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(0)
  const [amount, setAmount] = useState("100")
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [refreshingID, setRefreshingID] = useState("")
  const [error, setError] = useState("")
  const [notice, setNotice] = useState("")

  const load = useCallback(async (targetPage = page) => {
    setLoading(true)
    setError("")
    try {
      const result = await listWalletRecharges(PAGE_SIZE, targetPage * PAGE_SIZE)
      setOrders(result.items)
      setTotal(result.total)
      setPage(targetPage)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "读取充值订单失败")
    } finally {
      setLoading(false)
    }
  }, [page])

  useEffect(() => { void load(0) }, [])

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError("")
    setNotice("")
    if (!/^\\d+$/.test(amount) || !Number.isSafeInteger(Number(amount)) || Number(amount) <= 0) {
      setError("充值金额必须是大于 0 的整数 USDT。")
      return
    }
    setCreating(true)
    try {
      const order = await createWalletRecharge(Number(amount))
      setNotice(`充值订单 ${order.order_no} 已创建。请通过订单收银台完成支付，到账以服务端回调核验为准。`)
      setAmount("100")
      await load(0)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "创建充值订单失败")
    } finally {
      setCreating(false)
    }
  }

  async function refreshOrder(orderID: string) {
    setRefreshingID(orderID)
    setError("")
    setNotice("")
    try {
      const updated = await refreshWalletRecharge(orderID)
      setNotice(`订单 ${updated.order_no} 状态已刷新：${statusLabel(updated.status)}。公开查单不会直接给钱包入账，到账仍以验签回调为准。`)
      await load(page)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "刷新充值订单失败")
    } finally {
      setRefreshingID("")
    }
  }

  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <main className="mx-auto w-full max-w-[1440px] space-y-6 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <div className="mb-2 flex gap-2"><Badge variant="secondary">账户中心</Badge><Badge variant="outline">真实 API</Badge></div>
          <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">USDT 充值与支付</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-muted-foreground">创建钱包充值订单、跳转网关收银台并追踪订单状态。充值金额按整数 USDT 处理；只有通过服务端签名验证的支付回调才会入账。</p>
        </div>
        <Button variant="outline" onClick={() => void load(page)} disabled={loading}>
          <RefreshCw className={`mr-2 size-4 ${loading ? "animate-spin" : ""}`} />刷新订单
        </Button>
      </div>

      {error && <div className="flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive" role="alert"><AlertCircle className="mt-0.5 size-4 shrink-0" />{error}</div>}
      {notice && <div className="flex items-start gap-2 rounded-lg border border-emerald-600/20 bg-emerald-600/5 p-3 text-sm" role="status"><CheckCircle2 className="mt-0.5 size-4 shrink-0" />{notice}</div>}

      <div className="grid gap-4 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)]">
        <Card>
          <CardHeader>
            <div className="flex size-10 items-center justify-center rounded-xl bg-primary/10 text-primary"><Wallet className="size-5" /></div>
            <CardTitle className="text-base">创建充值订单</CardTitle>
            <CardDescription>支付渠道需由平台管理员预先配置并启用。</CardDescription>
          </CardHeader>
          <CardContent>
            <form className="space-y-4" onSubmit={(event) => void submit(event)}>
              <div className="space-y-2">
                <Label htmlFor="wallet-recharge-amount">充值金额（USDT）</Label>
                <Input id="wallet-recharge-amount" inputMode="numeric" value={amount} onChange={(event) => setAmount(event.target.value)} placeholder="请输入整数金额" required />
                <p className="text-xs text-muted-foreground">仅接受正整数，不会对小数金额进行四舍五入或截断。</p>
              </div>
              <Button type="submit" className="w-full" disabled={creating}>
                {creating ? "正在创建订单…" : "创建充值订单"}
              </Button>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">充值安全说明</CardTitle>
            <CardDescription>订单状态与资金入账由服务端支付流程控制。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm text-muted-foreground">
            <div className="flex gap-3"><CheckCircle2 className="mt-0.5 size-4 shrink-0 text-primary" /><p>创建订单后通过 HTTPS 收银台完成支付，收银台地址由服务端返回。</p></div>
            <div className="flex gap-3"><ShieldCheckIcon /><p>网关回调需通过签名、商户、订单和金额核验，并以事务方式入账。</p></div>
            <div className="flex gap-3"><Clock3 className="mt-0.5 size-4 shrink-0 text-primary" /><p>主动刷新只同步网关状态，不会单独触发钱包加款。</p></div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div><CardTitle className="text-base">充值订单</CardTitle><CardDescription>共 {total} 笔，按创建时间倒序显示。</CardDescription></div>
            <Badge variant="outline">{page + 1} / {pageCount} 页</Badge>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? <p className="py-10 text-center text-sm text-muted-foreground">正在读取充值订单…</p> : orders.length === 0 ? (
            <div className="py-10 text-center"><p className="text-sm font-medium">暂无充值订单</p><p className="mt-1 text-xs text-muted-foreground">创建充值订单后，会在这里显示支付状态和收银台入口。</p></div>
          ) : (
            <div className="space-y-3">
              {orders.map((order) => (
                <div key={order.id} className="flex flex-col gap-3 rounded-xl border p-4 lg:flex-row lg:items-center">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2"><p className="font-medium">{order.order_no}</p><Badge variant={statusVariant(order.status)}>{statusLabel(order.status)}</Badge></div>
                    <p className="mt-1 text-xs text-muted-foreground">创建于 {formatDate(order.created_at)}</p>
                    <p className="mt-2 text-sm font-semibold tabular-nums">{formatUSDT(order.amount_usdt)} USDT</p>
                    {order.actual_amount && <p className="mt-1 text-xs text-muted-foreground">实际到账：{order.actual_amount} {order.token ?? "USDT"} {order.network ? `· ${order.network}` : ""}</p>}
                    {order.receive_address && <p className="mt-1 break-all text-xs text-muted-foreground">收款地址：{order.receive_address}</p>}
                    {order.expires_at && <p className="mt-1 text-xs text-muted-foreground">支付截止：{formatDate(order.expires_at)}</p>}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {order.payment_url && order.status === "pending" && <Button asChild size="sm"><a href={order.payment_url} target="_blank" rel="noreferrer">前往支付<ArrowUpRight className="ml-2 size-3.5" /></a></Button>}
                    {(order.status === "pending" || order.status === "review") && <Button size="sm" variant="outline" onClick={() => void refreshOrder(order.id)} disabled={refreshingID === order.id}><RefreshCw className={`mr-2 size-3.5 ${refreshingID === order.id ? "animate-spin" : ""}`} />刷新状态</Button>}
                  </div>
                </div>
              ))}
            </div>
          )}
          <div className="mt-5 flex items-center justify-between border-t pt-4">
            <p className="text-xs text-muted-foreground">每页 {PAGE_SIZE} 笔</p>
            <div className="flex gap-2">
              <Button size="sm" variant="outline" onClick={() => void load(page - 1)} disabled={loading || page <= 0}>上一页</Button>
              <Button size="sm" variant="outline" onClick={() => void load(page + 1)} disabled={loading || page + 1 >= pageCount}>下一页</Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </main>
  )
}

function ShieldCheckIcon() {
  return <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-primary" />
}
