import { useCallback, useEffect, useRef, useState } from "react"
import { AlertCircle, RefreshCw, Wallet, Crown, ArrowDownUp } from "lucide-react"
import { getAgentAccount, listAgentUpgradeOrders, listWalletLedger, upgradeAgentWithWallet, type AgentAccount, type AgentUpgradeOrder, type WalletLedgerEntry } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

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
    try {
      const [accountResult, ledgerResult] = await Promise.all([
        getAgentAccount(),
        listWalletLedger(20, 0),
      ])
      setAccount(accountResult)
      setLedger(ledgerResult.items)
      setLedgerTotal(ledgerResult.total)\n      setUpgradeOrders(orderResult.items)\n      setUpgradeOrderTotal(orderResult.total)
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载代理账户失败")
    } finally {
      setLoading(false)
    }
  }, [])

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
            当前已支持使用账户 USDT 余额完成升级，订单、扣款、资金流水和等级生效在同一数据库事务内提交；在线支付渠道仍在开发中。
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">代理升级订单</CardTitle>
          <CardDescription>最近 10 笔升级订单，共 {upgradeOrderTotal} 笔。订单状态以服务端记录为准。</CardDescription>
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
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between gap-3">
            <div>
              <CardTitle className="text-base">钱包资金流水</CardTitle>
              <CardDescription>当前账户最近 20 条资金记录，共 {ledgerTotal} 条。</CardDescription>
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
        </CardContent>
      </Card>
    </main>
  )
}
