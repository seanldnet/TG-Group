import { useCallback, useEffect, useState } from "react"
import { ArrowDownLeft, ArrowUpRight, RefreshCw, Wallet } from "lucide-react"
import { getAgentAccount, listAccountOrders, listAccountLicenses, listWalletLedger, listWalletRecharges, type AccountOrder, type AccountLicense, type WalletLedgerEntry, type WalletRechargeOrder } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

function date(value:string){const d=new Date(value);return Number.isNaN(d.getTime())?"—":d.toLocaleString("zh-CN")}
function money(value:number){return new Intl.NumberFormat("en-US").format(value)+" USDT"}
export default function FinanceOverviewPage(){
 const [balance,setBalance]=useState<number|null>(null)
 const [ledger,setLedger]=useState<WalletLedgerEntry[]>([])
 const [recharges,setRecharges]=useState<WalletRechargeOrder[]>([])
 const [orders,setOrders]=useState<AccountOrder[]>([])
 const [licenses,setLicenses]=useState<AccountLicense[]>([])
 const [errors,setErrors]=useState<string[]>([])
 const [loading,setLoading]=useState(true)
 const [refreshing,setRefreshing]=useState(false)
 const load=useCallback(async()=>{
  setRefreshing(true)
  const results=await Promise.allSettled([getAgentAccount(),listWalletLedger(10,0),listWalletRecharges(5,0),listAccountOrders(5,0),listAccountLicenses()])
  const failures:string[]=[]
  if(results[0].status==="fulfilled")setBalance(results[0].value.wallet_balance_usdt);else failures.push("钱包余额")
  if(results[1].status==="fulfilled")setLedger(results[1].value.items);else failures.push("资金流水")
  if(results[2].status==="fulfilled")setRecharges(results[2].value.items);else failures.push("充值订单")
  if(results[3].status==="fulfilled")setOrders(results[3].value.items);else failures.push("套餐订单")
  if(results[4].status==="fulfilled")setLicenses(results[4].value.items);else failures.push("套餐授权")
  setErrors(failures);setLoading(false);setRefreshing(false)
 },[])
 useEffect(()=>{void load()},[load])
 return <main className="mx-auto w-full max-w-[1440px] space-y-6 p-4 sm:p-6 lg:p-8">
  <div className="flex flex-wrap items-end justify-between gap-3"><div><Badge variant="secondary" className="mb-2">账户业务</Badge><h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">财务概览</h1><p className="mt-2 text-sm text-muted-foreground">查看本人钱包余额、资金流水、充值记录、套餐订单与授权。数据均来自当前账号真实 API。</p></div><Button variant="outline" onClick={()=>void load()} disabled={refreshing}><RefreshCw className={`mr-2 size-4 ${refreshing?"animate-spin":""}`}/>刷新</Button></div>
  {errors.length>0&&<p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">部分数据暂时无法读取：{errors.join("、")}。其他已成功读取的数据仍可查看。</p>}
  <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
   <Card><CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2"><CardTitle className="text-sm text-muted-foreground">钱包余额</CardTitle><Wallet className="size-4 text-primary"/></CardHeader><CardContent><p className="text-2xl font-semibold tabular-nums">{balance===null?(loading?"读取中…":"暂不可用"):money(balance)}</p><p className="mt-1 text-xs text-muted-foreground">USDT 整数余额</p></CardContent></Card>
   <Card><CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2"><CardTitle className="text-sm text-muted-foreground">资金流水</CardTitle><ArrowUpRight className="size-4 text-primary"/></CardHeader><CardContent><p className="text-2xl font-semibold tabular-nums">{ledger.length}</p><p className="mt-1 text-xs text-muted-foreground">最近 10 笔记录</p></CardContent></Card>
   <Card><CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2"><CardTitle className="text-sm text-muted-foreground">充值订单</CardTitle><ArrowDownLeft className="size-4 text-primary"/></CardHeader><CardContent><p className="text-2xl font-semibold tabular-nums">{recharges.length}</p><p className="mt-1 text-xs text-muted-foreground">最近 5 笔记录</p></CardContent></Card>
   <Card><CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2"><CardTitle className="text-sm text-muted-foreground">套餐授权</CardTitle><Wallet className="size-4 text-primary"/></CardHeader><CardContent><p className="text-2xl font-semibold tabular-nums">{licenses.filter(x=>x.status==="active").length}</p><p className="mt-1 text-xs text-muted-foreground">当前生效授权</p></CardContent></Card>
  </section>
  <Card><CardHeader><CardTitle className="text-base">最近资金流水</CardTitle><CardDescription>展示最近 10 笔本人钱包变动</CardDescription></CardHeader><CardContent>
   {loading?<p className="py-8 text-center text-sm text-muted-foreground">正在读取…</p>:ledger.length?<div className="overflow-x-auto"><Table><TableHeader><TableRow><TableHead>类型</TableHead><TableHead>说明</TableHead><TableHead>变动金额</TableHead><TableHead>变动后余额</TableHead><TableHead>时间</TableHead></TableRow></TableHeader><TableBody>{ledger.map(item=><TableRow key={item.id}><TableCell><Badge variant="outline">{item.entry_type}</Badge></TableCell><TableCell className="min-w-40">{item.description||item.reference_type}</TableCell><TableCell className={item.amount_usdt>=0?"font-medium text-emerald-600":"font-medium text-destructive"}>{item.amount_usdt>0?"+":""}{money(item.amount_usdt)}</TableCell><TableCell className="tabular-nums">{money(item.balance_after_usdt)}</TableCell><TableCell className="whitespace-nowrap text-xs text-muted-foreground">{date(item.created_at)}</TableCell></TableRow>)}</TableBody></Table></div>:<p className="py-8 text-center text-sm text-muted-foreground">暂无资金流水。</p>}
  </CardContent></Card>
  <div className="grid gap-4 xl:grid-cols-2">
   <Card><CardHeader><CardTitle className="text-base">最近充值订单</CardTitle><CardDescription>钱包充值的网关订单状态</CardDescription></CardHeader><CardContent>{loading?<p className="py-6 text-center text-sm text-muted-foreground">正在读取…</p>:recharges.length?<div className="space-y-2">{recharges.map(item=><div key={item.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border p-3"><div><p className="font-mono text-xs">{item.order_no}</p><p className="mt-1 text-xs text-muted-foreground">{date(item.created_at)}</p></div><div className="text-right"><p className="text-sm font-medium">{money(item.amount_usdt)}</p><Badge variant={item.status==="paid"?"default":item.status==="pending"?"secondary":item.status==="review"?"destructive":"outline"}>{item.status==="paid"?"已到账":item.status==="pending"?"待支付":item.status==="review"?"待核查":item.status==="expired"?"已过期":"失败"}</Badge></div></div>)}</div>:<p className="py-6 text-center text-sm text-muted-foreground">暂无充值订单。</p>}</CardContent></Card>
   <Card><CardHeader><CardTitle className="text-base">最近套餐订单</CardTitle><CardDescription>套餐购买与钱包扣款记录</CardDescription></CardHeader><CardContent>{loading?<p className="py-6 text-center text-sm text-muted-foreground">正在读取…</p>:orders.length?<div className="space-y-2">{orders.map(item=><div key={item.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border p-3"><div><p className="font-medium">{item.plan_name}</p><p className="mt-1 font-mono text-xs text-muted-foreground">{item.order_no}</p></div><div className="text-right"><p className="text-sm font-medium">{money(item.amount_usdt)}</p><Badge variant={item.status==="paid"?"default":"secondary"}>{item.status==="paid"?"已支付":item.status}</Badge></div></div>)}</div>:<p className="py-6 text-center text-sm text-muted-foreground">暂无套餐订单。</p>}</CardContent></Card>
  </div>
  <Card><CardHeader><CardTitle className="text-base">套餐授权概览</CardTitle><CardDescription>授权期限与当前状态</CardDescription></CardHeader><CardContent>{loading?<p className="py-6 text-center text-sm text-muted-foreground">正在读取…</p>:licenses.length?<div className="space-y-2">{licenses.map(item=><div key={item.id} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3"><div><p className="font-medium">{item.plan_name}</p><p className="mt-1 text-xs text-muted-foreground">到期：{date(item.expires_at)}</p></div><Badge variant={item.status==="active"?"default":item.status==="suspended"?"destructive":"secondary"}>{item.status==="active"?"生效中":item.status==="suspended"?"已暂停":"已到期"}</Badge></div>)}</div>:<p className="py-6 text-center text-sm text-muted-foreground">暂无套餐授权。</p>}</CardContent></Card>
 </main>
}
