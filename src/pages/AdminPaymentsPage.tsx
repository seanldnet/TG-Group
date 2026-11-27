import { useEffect, useState, type FormEvent } from "react"
import { AlertCircle, CreditCard, Save, ShieldCheck } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { getGMPaySupportedAssets, listAdminPaymentChannels, listAdminRechargeOrders, saveAdminPaymentChannel, type AdminRechargeOrder, type PaymentChannelStatus, type SupportedPaymentAsset } from "@/lib/api"

const initial = { name: "EPUSDT GMPay", base_url: "", merchant_pid: "", secret_key: "", currency: "usd", token: "usdt", network: "tron", enabled: false }

export default function AdminPaymentsPage() {
  const [form, setForm] = useState(initial)
  const [status, setStatus] = useState<PaymentChannelStatus | null>(null)
  const [supportedAssets, setSupportedAssets] = useState<SupportedPaymentAsset[]>([])
  const [rechargeOrders, setRechargeOrders] = useState<AdminRechargeOrder[]>([])
  const [rechargeTotal, setRechargeTotal] = useState(0)
  const [statusFilter, setStatusFilter] = useState("")
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState("")
  const [notice, setNotice] = useState("")

  async function load() {
    setLoading(true)
    setError("")
    try {
      const [result, assets, orders] = await Promise.all([
        listAdminPaymentChannels(),
        getGMPaySupportedAssets().catch(() => ({ supported_assets: [] as SupportedPaymentAsset[] })),
        listAdminRechargeOrders(statusFilter, 20, 0),
      ])
      setSupportedAssets(assets.supported_assets)
      setRechargeOrders(orders.items)
      setRechargeTotal(orders.total)
      const channel = result.items.find((item) => item.code === "epusdt_gmpay")
      setStatus(channel ?? null)
      if (channel) setForm((current) => ({ ...current, name: channel.name, base_url: channel.base_url, merchant_pid: channel.merchant_pid, currency: channel.currency, token: channel.token, network: channel.network, enabled: channel.enabled, secret_key: "" }))
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "读取支付渠道失败")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { void load() }, [])

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSaving(true)
    setError("")
    setNotice("")
    try {
      const result = await saveAdminPaymentChannel("epusdt_gmpay", form)
      const channel = result.items.find((item) => item.code === "epusdt_gmpay") ?? null
      setStatus(channel)
      setForm((current) => ({ ...current, secret_key: "" }))
      setNotice("支付渠道配置已加密保存。密钥不会回显。")
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "保存支付渠道失败")
    } finally {
      setSaving(false)
    }
  }

  function update<K extends keyof typeof form>(key: K, value: (typeof form)[K]) {
    setForm((current) => ({ ...current, [key]: value }))
  }

  return (
    <main className="mx-auto w-full max-w-5xl space-y-6 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="mb-2 flex gap-2"><Badge variant="secondary">平台管理</Badge><Badge variant="outline">支付渠道</Badge></div>
          <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">支付渠道配置</h1>
          <p className="mt-2 text-sm text-muted-foreground">配置 EPUSDT GMPay 商户凭证与收银台参数，密钥使用 AES-256-GCM 加密保存。</p>
        </div>
        <CreditCard className="size-8 text-muted-foreground" />
      </div>

      {error && <div role="alert" className="flex gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive"><AlertCircle className="size-4 shrink-0" />{error}</div>}
      {notice && <p role="status" className="rounded-lg border p-3 text-sm">{notice}</p>}

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between gap-3">
            <CardTitle className="text-base">EPUSDT GMPay</CardTitle>
            {status && <Badge variant={status.enabled ? "default" : "secondary"}>{status.enabled ? "已启用" : "已停用"}</Badge>}
          </div>
          <CardDescription>新接入使用 GMPay HMAC-SHA256 协议。支付网关必须使用 HTTPS；币种和网络需要与 EPUSDT 公共配置支持项一致。</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? <p className="py-6 text-center text-sm text-muted-foreground">正在读取配置…</p> : (
            <form className="space-y-5" onSubmit={submit}>
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2"><Label htmlFor="payment-name">渠道名称</Label><Input id="payment-name" value={form.name} onChange={(event) => update("name", event.target.value)} required /></div>
                <div className="space-y-2"><Label htmlFor="payment-base-url">EPUSDT 网关 HTTPS 地址</Label><Input id="payment-base-url" value={form.base_url} onChange={(event) => update("base_url", event.target.value)} placeholder="https://pay.example.com" required /></div>
                <div className="space-y-2"><Label htmlFor="payment-pid">商户 PID</Label><Input id="payment-pid" value={form.merchant_pid} onChange={(event) => update("merchant_pid", event.target.value)} required /></div>
                <div className="space-y-2"><Label htmlFor="payment-secret">商户 Secret Key</Label><Input id="payment-secret" type="password" autoComplete="new-password" value={form.secret_key} onChange={(event) => update("secret_key", event.target.value)} placeholder="保存后不会回显；更新时请重新输入" required /></div>
                <div className="space-y-2"><Label htmlFor="payment-currency">计价币种</Label><Input id="payment-currency" value={form.currency} onChange={(event) => update("currency", event.target.value)} required /></div>
                <div className="space-y-2"><Label htmlFor="payment-token">收款资产</Label><Input id="payment-token" value={form.token} onChange={(event) => update("token", event.target.value)} required /></div>
                <div className="space-y-2"><Label htmlFor="payment-network">收款网络</Label><Input id="payment-network" value={form.network} onChange={(event) => update("network", event.target.value)} required /></div>
              </div>
              <label className="flex items-center gap-3 rounded-lg border p-3 text-sm"><input type="checkbox" checked={form.enabled} onChange={(event) => update("enabled", event.target.checked)} className="size-4" /><span>启用此支付渠道</span></label>
              <div className="flex items-start gap-2 rounded-lg bg-muted/40 p-3 text-xs leading-5 text-muted-foreground"><ShieldCheck className="mt-0.5 size-4 shrink-0" />请同时设置服务端环境变量 PAYMENT_CREDENTIALS_ENCRYPTION_KEY（Base64 编码的 32 字节密钥）和 PAYMENT_PUBLIC_BASE_URL（网站 HTTPS 公网根地址），并确保该地址能接收 GMPay 异步回调。</div>
              <div className="rounded-lg border p-3">
                <p className="text-sm font-medium">网关当前支持的资产与网络</p>
                {supportedAssets.length ? <div className="mt-2 flex flex-wrap gap-2">{supportedAssets.map((asset) => <Badge key={asset.network} variant="secondary">{asset.display_name} · {asset.network} · {asset.tokens.join(", ")}</Badge>)}</div> : <p className="mt-1 text-xs text-muted-foreground">保存商户网关地址与凭证后，可读取 EPUSDT 公共配置；请只选择网关实际启用的网络。</p>}
              </div>
              <div className="flex justify-end"><Button type="submit" disabled={saving || !form.secret_key.trim()}><Save className="mr-2 size-4" />{saving ? "正在保存…" : "保存渠道配置"}</Button></div>
            </form>
          )}
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle className="text-base">最近充值订单</CardTitle>
          <CardDescription>平台共 {rechargeTotal} 笔充值订单。金额异常的订单进入待核查状态，不会自动入账。</CardDescription>
          <div className="flex flex-wrap items-center gap-2 pt-2">
            <Label htmlFor="recharge-status-filter">订单状态</Label>
            <select id="recharge-status-filter" value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)} className="h-9 rounded-md border bg-background px-3 text-sm">
              <option value="">全部状态</option>
              <option value="pending">待支付</option>
              <option value="paid">已到账</option>
              <option value="review">待核查</option>
              <option value="expired">已过期</option>
              <option value="failed">失败</option>
            </select>
            <Button size="sm" variant="outline" onClick={() => void load()} disabled={loading}>筛选</Button>
          </div>
        </CardHeader>
        <CardContent>
          {rechargeOrders.length ? (
            <div className="space-y-3">
              {rechargeOrders.map((order) => (
                <div key={order.id} className="space-y-2 rounded-lg border p-3">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0"><p className="font-medium">{order.order_no}</p><p className="mt-1 break-all text-xs text-muted-foreground">{order.email} · {order.channel_code} · {new Date(order.created_at).toLocaleString()}</p></div>
                    <Badge variant={order.status === "paid" ? "default" : order.status === "review" ? "destructive" : order.status === "pending" ? "secondary" : "outline"}>{order.status === "paid" ? "已到账" : order.status === "review" ? "待核查" : order.status === "pending" ? "待支付" : order.status === "expired" ? "已过期" : "失败"}</Badge>
                  </div>
                  <div className="flex flex-wrap gap-x-5 gap-y-1 text-xs text-muted-foreground">
                    <span>订单金额：{order.amount_usdt} USDT</span>
                    <span>实际到账：{order.actual_amount || "—"} {order.token || "USDT"}</span>
                    <span>网关交易：{order.provider_trade_id || "—"}</span>
                  </div>
                  {order.block_transaction_id && <p className="break-all text-xs text-muted-foreground">链上交易：{order.block_transaction_id}</p>}
                  {order.status === "review" && <p className="text-xs text-destructive">实际到账数量不是整数 USDT，未自动入账。请核对网关交易后再按财务流程处理。</p>}
                </div>
              ))}
            </div>
          ) : <p className="py-8 text-center text-sm text-muted-foreground">暂无充值订单。</p>}
        </CardContent>
      </Card>
    </main>
  )
}
