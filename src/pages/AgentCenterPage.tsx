import { useCallback, useEffect, useRef, useState } from "react"
import { AlertCircle, RefreshCw, Wallet, Crown, ArrowDownUp } from "lucide-react"
import { createWalletRecharge, getAgentAccount, listAgentUpgradeOrders, listWalletLedger, listWalletRecharges, refreshWalletRecharge, upgradeAgentWithWallet, type AgentAccount, type AgentUpgradeOrder, type WalletLedgerEntry, type WalletRechargeOrder } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

function formatUSDT(value: number) {
  return new Intl.NumberFormat("en-US", { maximumFractionDigits: 0 }).format(value)
}

function entryLabel(type: string) {
  const labels: Record<string, string> = {
    deposit: "余额充值",
    agent_upgrade: "代理等级升级",
    order_payment: "订单支付",
    refund: "退款",
    commission: "佣金",
    adjustment: "余额调整",
  }
  return labels[type] ?? type
}

export default function AgentCenterPage() {
  const [account, setAccount] = useState<AgentAccount | null>(null)
  const [ledger, setLedger] = useState<WalletLedgerEntry[]>([])
  const [rechargeOrders, setRechargeOrders] = useState<WalletRechargeOrder[]>([])
  const [rechargeLoadError, setRechargeLoadError] = useState("")
  const [rechargeTotal, setRechargeTotal] = useState(0)
  const [rechargePage, setRechargePage] = useState(0)
  const [upgradePage, setUpgradePage] = useState(0)
  const [ledgerPage, setLedgerPage] = useState(0)
  const [historyLoading, setHistoryLoading] = useState("")
  const [historyError, setHistoryError] = useState("")
  const [rechargeAmount, setRechargeAmount] = useState("100")
  const [recharging, setRecharging] = useState(false)
  const [refreshingRechargeID, setRefreshingRechargeID] = useState("")
  const [rechargeMessage, setRechargeMessage] = useState("")
  const [upgradeOrders, setUpgradeOrders] = useState<AgentUpgradeOrder[]>([])
  const [upgradeOrderTotal, setUpgradeOrderTotal] = useState(0)
  const [ledgerTotal, setLedgerTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")
  const [upgradingLevelID, setUpgradingLevelID] = useState("")
  const [upgradeMessage, setUpgradeMessage] = useState("")
  const upgradeKeys = useRef<Record<string, string>>({})

  const load = useCallback(async () => {
    setLoading(true)
    setError("")
    setRechargeLoadError("")
    try {
      const [accountResult, ledgerResult, orderResult] = await Promise.all([
        getAgentAccount(),
        listWalletLedger(20, 0),
        listAgentUpgradeOrders(10, 0),
      ])
      const rechargeResult = await listWalletRecharges(10, 0).then((result) => ({ result, error: "" })).catch((err) => ({ result: { items: [] as WalletRechargeOrder[], total: 0 }, error: err instanceof Error ? err.message : "充值订单加载失败" }))
      setAccount(accountResult)
      setLedger(ledgerResult.items)
      setLedgerTotal(ledgerResult.total)
      setUpgradeOrders(orderResult.items)
      setUpgradeOrderTotal(orderResult.total)
      setRechargeOrders(rechargeResult.result.items)
      setRechargeTotal(rechargeResult.result.total)
      setRechargeLoadError(rechargeResult.error)
      setRechargePage(0)
      setUpgradePage(0)
      setLedgerPage(0)
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载代理账户失败")
    } finally {
      setLoading(false)
    }
  }, [])

  async function loadHistory(kind: "recharge" | "upgrade" | "ledger", targetPage: number) {
    setHistoryLoading(kind)
    setHistoryError("")
    try {
      if (kind === "recharge") {
        const result = await listWalletRecharges(10, targetPage * 10)
        setRechargeOrders(result.items)
        setRechargeTotal(result.total)
        setRechargePage(targetPage)
        setRechargeLoadError("")
      } else if (kind === "upgrade") {
        const result = await listAgentUpgradeOrders(10, targetPage * 10)
        setUpgradeOrders(result.items)
        setUpgradeOrderTotal(result.total)
        setUpgradePage(targetPage)
      } else {
        const result = await listWalletLedger(20, targetPage * 20)
        setLedger(result.items)
        setLedgerTotal(result.total)
        setLedgerPage(targetPage)
      }
    } catch (err) {
      setHistoryError(err instanceof Error ? err.message : "读取账户历史记录失败")
    } finally {
      setHistoryLoading("")
    }
  }

  async function handleWalletUpgrade(levelID: string, levelName: string, price: number) {
    if (!account) return
    if (!window.confirm(`确认使用钱包余额 ${formatUSDT(price)} USDT 升级至“${levelName}”吗？支付成功后等级将立即生效。`)) return
    setUpgradingLevelID(levelID)
    setUpgradeMessage("")
    try {
      const idempotencyKey = upgradeKeys.current[levelID] ?? crypto.randomUUID()
      upgradeKeys.current[levelID] = idempotencyKey
      const order = await upgradeAgentWithWallet(levelID, idempotencyKey)
      delete upgradeKeys.current[levelID]
      setUpgradeMessage(`升级成功，订单号：${order.order_no}。代理等级已自动生效。`)
      await load()
    } catch (err) {
      setUpgradeMessage(err instanceof Error ? err.message : "代理升级失败")
    } finally {
      setUpgradingLevelID("")
    }
  }

  async function handleRefreshRecharge(orderID: string) {
    setRefreshingRechargeID(orderID)
    setRechargeMessage("")
    try {
      const updated = await refreshWalletRecharge(orderID)
      setRechargeOrders((current) => current.map((item) => item.id === updated.id ? updated : item))
      if (updated.provider_status === 2 && updated.status === "pending") {
        setRechargeMessage("网关已确认支付，正在等待签名回调完成钱包入账。")
      } else if (updated.provider_status === 5) {
        setRechargeMessage("网关已检测到转账，正在等待链上确认。")
      } else if (updated.status === "review") {
        setRechargeMessage("该订单进入待核查状态，暂未入账。")
      } else {
        setRechargeMessage("支付状态已刷新。")
      }
      await load()
    } catch (err) {
      setRechargeMessage(err instanceof Error ? err.message : "刷新支付状态失败")
    } finally {
      setRefreshingRechargeID("")
    }
  }

  async function handleRecharge() {
    const amount = Number(rechargeAmount)
    if (!/^\d+$/.test(rechargeAmount) || !Number.isSafeInteger(amount) || amount <= 0) {
      setRechargeMessage("请输入正整数 USDT 充值金额。")
      return
    }
    setRecharging(true)
    setRechargeMessage("")
    try {
      const order = await createWalletRecharge(amount)
      setRechargeOrders((current) => [order, ...current.filter((item) => item.id !== order.id)].slice(0, 10))
      setRechargeLoadError("")
      if (order.payment_url) {
        window.location.assign(order.payment_url)
        return
      }
      setRechargeMessage(`充值订单已创建：${order.order_no}`)
    } catch (err) {
      setRechargeMessage(err instanceof Error ? err.message : "创建充值订单失败")
    } finally {
      setRecharging(false)
    }
  }

  useEffect(() => { void load() }, [load])

  return (
    <main className="mx-auto w-full max-w-[1440px] space-y-6 p-4 sm:p-6 lg:p-8">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <div className="mb-2 flex gap-2">
            <Badge variant="secondary">账户中心</Badge>
            <Badge variant="outline">实时账户数据</Badge>
          </div>
          <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">代理中心</h1>
          <p className="mt-2 text-sm text-muted-foreground">查看代理等级、可用等级方案、USDT 余额与资金流水。</p>
        </div>
        <Button variant="outline" onClick={() => void load()} disabled={loading}>
          <RefreshCw className={`mr-2 size-4 ${loading ? "animate-spin" : ""}`} />
          刷新数据
        </Button>
      </div>

      {historyError && <div role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">读取历史记录失败：{historyError}</div>}
      {error && (
        <div className="flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive" role="alert">
          <AlertCircle className="mt-0.5 size-4 shrink-0" />
          <span>{error}</span>
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">USDT 钱包余额</CardTitle>
            <Wallet className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-semibold tabular-nums">{loading ? "—" : account ? formatUSDT(account.wallet_balance_usdt) : "—"} <span className="text-base font-medium text-muted-foreground">USDT</span></p>
            <p className="mt-1 text-xs text-muted-foreground">余额来自当前登录账户的钱包账户</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">当前代理等级</CardTitle>
            <Crown className="size-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-semibold">{loading ? "加载中…" : account?.current_level?.name ?? "尚未开通代理"}</p>
            <p className="mt-1 text-xs text-muted-foreground">
              {account?.is_agent ? (account.status === "active" ? "代理状态正常" : "代理状态受限") : "可查看下方已开放的等级目录"}
            </p>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">代理等级目录</CardTitle>
          <CardDescription>等级价格和权益由平台配置；金额以整数 USDT 展示。</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="py-8 text-center text-sm text-muted-foreground">正在读取等级目录…</p>
          ) : account?.levels.length ? (
            <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              {account.levels.map((level) => (
                <div key={level.id} className="rounded-lg border p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="font-medium">{level.name}</p>
                      <p className="mt-1 text-xs text-muted-foreground">{level.code}</p>
                    </div>
                    {account.current_level?.id === level.id && <Badge>当前等级</Badge>}
                  </div>
                  <p className="mt-4 text-2xl font-semibold tabular-nums">{formatUSDT(level.price_usdt)} <span className="text-sm font-medium text-muted-foreground">USDT</span></p>
                  {Object.keys(level.benefits ?? {}).length > 0 && (
                    <ul className="mt-3 space-y-1 text-xs leading-5 text-muted-foreground">
                      {Object.entries(level.benefits).map(([key, value]) => (
                        <li key={key}>{key}: {typeof value === "string" ? value : JSON.stringify(value)}</li>
                      ))}
                    </ul>
                  )}
                  {account.current_level?.id !== level.id && level.sort_order > (account.current_level?.sort_order ?? -1) && (
                    <Button
                      className="mt-4 w-full"
                      size="sm"
                      disabled={!!upgradingLevelID || account.wallet_balance_usdt < level.price_usdt}
                      onClick={() => void handleWalletUpgrade(level.id, level.name, level.price_usdt)}
                    >
                      {upgradingLevelID === level.id ? "正在升级…" : `余额升级 · ${formatUSDT(level.price_usdt)} USDT`}
                    </Button>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <p className="py-8 text-center text-sm text-muted-foreground">平台暂未开放代理等级。</p>
          )}
          {upgradeMessage && <p className="mt-4 rounded-lg border p-3 text-sm" role="status">{upgradeMessage}</p>}
          <p className="mt-4 rounded-lg bg-muted/40 p-3 text-xs leading-5 text-muted-foreground">
            当前已支持使用账户 USDT 余额完成升级，订单、扣款、资金流水和等级生效在同一数据库事务内提交；在线充值现已接入 GMPay 服务端流程；管理员完成渠道配置并部署 HTTPS 回调地址后可创建充值订单。
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">USDT 余额充值</CardTitle>
          <CardDescription>通过已配置的 EPUSDT GMPay 收银台充值。实际到账必须为整数 USDT，非整数到账不会自动入账。</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
            <div className="flex-1 space-y-2">
              <Label htmlFor="wallet-recharge-amount">充值金额（整数 USDT）</Label>
              <Input id="wallet-recharge-amount" inputMode="numeric" value={rechargeAmount} onChange={(event) => setRechargeAmount(event.target.value)} placeholder="例如 100" />
            </div>
            <Button onClick={() => void handleRecharge()} disabled={recharging}>{recharging ? "正在创建订单…" : "创建充值订单"}</Button>
          </div>
          {rechargeMessage && <p className="rounded-lg border p-3 text-sm" role="status">{rechargeMessage}</p>}
          {rechargeLoadError ? <p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">充值订单加载失败：{rechargeLoadError}</p> : rechargeOrders.length > 0 ? (
            <div className="space-y-2">
              <div className="flex flex-wrap items-center justify-between gap-2"><p className="text-sm font-medium">充值订单 · 共 {rechargeTotal} 笔</p><div className="flex items-center gap-2"><Button size="sm" variant="outline" disabled={historyLoading!==""||rechargePage===0} onClick={()=>void loadHistory("recharge",rechargePage-1)}>上一页</Button><span className="text-xs text-muted-foreground">第 {rechargePage+1} / {Math.max(1,Math.ceil(rechargeTotal/10))} 页</span><Button size="sm" variant="outline" disabled={historyLoading!==""||(rechargePage+1)*10>=rechargeTotal} onClick={()=>void loadHistory("recharge",rechargePage+1)}>下一页</Button></div></div>
              {rechargeOrders.map((order) => (
                <div key={order.id} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3">
                  <div><p className="text-sm font-medium">{order.order_no}</p><p className="mt-1 text-xs text-muted-foreground">{new Date(order.created_at).toLocaleString()}</p></div>
                  <div className="text-right"><p className="font-semibold tabular-nums">{formatUSDT(order.amount_usdt)} USDT</p><div className="flex flex-col items-end gap-2"><Badge variant={order.status === "paid" ? "default" : order.status === "pending" ? "secondary" : "destructive"}>{order.status === "paid" ? "已到账" : order.status === "pending" ? (order.provider_status === 5 ? "链上确认中" : order.provider_status === 2 ? "等待入账回调" : "待支付") : order.status === "review" ? "待核查" : order.status === "expired" ? "已过期" : "失败"}</Badge>{order.status === "pending" && order.provider_trade_id && <Button size="sm" variant="outline" disabled={refreshingRechargeID === order.id} onClick={() => void handleRefreshRecharge(order.id)}>{refreshingRechargeID === order.id ? "查询中…" : "刷新支付状态"}</Button>}</div></div>
                </div>
              ))}
            </div>
          ) : !loading ? <p className="py-4 text-center text-sm text-muted-foreground">暂无充值订单。</p> : null}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">代理升级订单</CardTitle>
          <CardDescription>共 {upgradeOrderTotal} 笔升级订单，按服务端分页读取。订单状态以服务端记录为准。</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? <p className="py-6 text-center text-sm text-muted-foreground">正在读取升级订单…</p> : upgradeOrders.length ? (
            <div className="space-y-3">
              {upgradeOrders.map((order) => (
                <div key={order.id} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3">
                  <div className="min-w-0">
                    <p className="font-medium">{order.order_no}</p>
                    <p className="mt-1 text-xs text-muted-foreground">{new Date(order.created_at).toLocaleString()} · {order.payment_method === "wallet" ? "钱包余额支付" : order.payment_method.toUpperCase()}</p>
                  </div>
                  <div className="text-right">
                    <p className="font-semibold tabular-nums">{formatUSDT(order.amount_usdt)} USDT</p>
                    <Badge variant={order.status === "paid" ? "default" : order.status === "pending" ? "secondary" : "destructive"}>{order.status === "paid" ? "已支付" : order.status === "pending" ? "待支付" : order.status === "failed" ? "支付失败" : "已取消"}</Badge>
                  </div>
                </div>
              ))}
            </div>
          ) : <p className="py-6 text-center text-sm text-muted-foreground">暂无代理升级订单。</p>}
          <div className="mt-4 flex flex-wrap items-center justify-between gap-2 border-t pt-4"><p className="text-xs text-muted-foreground">第 {upgradeOrderTotal===0?0:upgradePage+1} / {Math.max(1,Math.ceil(upgradeOrderTotal/10))} 页</p><div className="flex gap-2"><Button size="sm" variant="outline" disabled={historyLoading!==""||upgradePage===0} onClick={()=>void loadHistory("upgrade",upgradePage-1)}>上一页</Button><Button size="sm" variant="outline" disabled={historyLoading!==""||(upgradePage+1)*10>=upgradeOrderTotal} onClick={()=>void loadHistory("upgrade",upgradePage+1)}>下一页</Button></div></div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between gap-3">
            <div>
              <CardTitle className="text-base">钱包资金流水</CardTitle>
              <CardDescription>当前账户资金流水共 {ledgerTotal} 条，按服务端分页读取。</CardDescription>
            </div>
            <ArrowDownUp className="size-4 text-muted-foreground" />
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="py-8 text-center text-sm text-muted-foreground">正在读取资金流水…</p>
          ) : ledger.length ? (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[680px] text-sm">
                <thead><tr className="border-b text-left text-xs text-muted-foreground">
                  <th className="px-3 py-3 font-medium">发生时间</th>
                  <th className="px-3 py-3 font-medium">类型</th>
                  <th className="px-3 py-3 font-medium">说明</th>
                  <th className="px-3 py-3 text-right font-medium">变动（USDT）</th>
                  <th className="px-3 py-3 text-right font-medium">变动后余额</th>
                </tr></thead>
                <tbody>
                  {ledger.map((entry) => (
                    <tr key={entry.id} className="border-b last:border-0">
                      <td className="whitespace-nowrap px-3 py-3 text-muted-foreground">{new Date(entry.created_at).toLocaleString()}</td>
                      <td className="px-3 py-3">{entryLabel(entry.entry_type)}</td>
                      <td className="max-w-[240px] truncate px-3 py-3 text-muted-foreground">{entry.description || entry.reference_id || "—"}</td>
                      <td className={`px-3 py-3 text-right font-medium tabular-nums ${entry.amount_usdt > 0 ? "text-emerald-600" : "text-foreground"}`}>{entry.amount_usdt > 0 ? "+" : ""}{formatUSDT(entry.amount_usdt)}</td>
                      <td className="px-3 py-3 text-right tabular-nums">{formatUSDT(entry.balance_after_usdt)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p className="py-8 text-center text-sm text-muted-foreground">暂无资金流水记录。</p>
          )}
          <div className="mt-4 flex flex-wrap items-center justify-between gap-2 border-t pt-4"><p className="text-xs text-muted-foreground">第 {ledgerTotal===0?0:ledgerPage+1} / {Math.max(1,Math.ceil(ledgerTotal/20))} 页</p><div className="flex gap-2"><Button size="sm" variant="outline" disabled={historyLoading!==""||ledgerPage===0} onClick={()=>void loadHistory("ledger",ledgerPage-1)}>上一页</Button><Button size="sm" variant="outline" disabled={historyLoading!==""||(ledgerPage+1)*20>=ledgerTotal} onClick={()=>void loadHistory("ledger",ledgerPage+1)}>下一页</Button></div></div>
        </CardContent>
      </Card>
    </main>
  )
}
