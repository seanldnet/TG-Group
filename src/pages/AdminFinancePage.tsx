import { useCallback, useEffect, useState } from "react"
import { RefreshCw, Search, Wallet } from "lucide-react"
import { getAdminFinanceSummary, listAdminWalletLedger, type AdminFinanceSummary, type AdminWalletLedgerEntry } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

const limit=20
const types=[{value:"",label:"全部"},{value:"deposit",label:"充值入账"},{value:"order_payment",label:"套餐购买"},{value:"agent_upgrade",label:"代理升级"},{value:"refund",label:"退款"},{value:"commission",label:"佣金"},{value:"adjustment",label:"资金调整"}]
function money(value:number){return new Intl.NumberFormat("en-US").format(value)+" USDT"}
function date(value:string){const d=new Date(value);return Number.isNaN(d.getTime())?"—":d.toLocaleString("zh-CN")}
export default function AdminFinancePage(){
 const [summary,setSummary]=useState<AdminFinanceSummary|null>(null)
 const [items,setItems]=useState<AdminWalletLedgerEntry[]>([])
 const [total,setTotal]=useState(0)
 const [offset,setOffset]=useState(0)
 const [entryType,setEntryType]=useState("")
 const [query,setQuery]=useState("")
 const [appliedQuery,setAppliedQuery]=useState("")
 const [loading,setLoading]=useState(true)
 const [summaryError,setSummaryError]=useState("")
 const [ledgerError,setLedgerError]=useState("")
 const load=useCallback(async(next=0,nextType=entryType,nextQuery=appliedQuery)=>{
  setLoading(true);setLedgerError("")
  try{const result=await listAdminWalletLedger({entry_type:nextType,q:nextQuery,limit,offset:next});setItems(result.items);setTotal(result.total);setOffset(next)}
  catch(err){setLedgerError(err instanceof Error?err.message:"读取平台资金流水失败")}
  finally{setLoading(false)}
 },[entryType,appliedQuery])
 const loadSummary=useCallback(async()=>{
  setSummaryError("")
  try{setSummary(await getAdminFinanceSummary())}
  catch(err){setSummaryError(err instanceof Error?err.message:"读取财务概览失败")}
 },[])
 useEffect(()=>{void load(0,"","");void loadSummary()},[])
 function filterType(value:string){setEntryType(value);void load(0,value,appliedQuery)}
 function search(){const value=query.trim();setAppliedQuery(value);void load(0,entryType,value)}
 const pages=Math.max(1,Math.ceil(total/limit))
 return <main className="mx-auto w-full max-w-[1440px] space-y-6 p-4 sm:p-6 lg:p-8">
  <div className="flex flex-wrap items-end justify-between gap-3"><div><Badge variant="secondary" className="mb-2">平台管理</Badge><h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">平台财务</h1><p className="mt-2 text-sm text-muted-foreground">查看全站钱包负债余额与资金流水。累计入账/支出是流水方向统计，不等同于平台利润或会计收入。</p></div><Button variant="outline" onClick={()=>{void load();void loadSummary()}} disabled={loading}><RefreshCw className={`mr-2 size-4 ${loading?"animate-spin":""}`}/>刷新</Button></div>
  {summaryError&&<p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">{summaryError}</p>}
  <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
   <Card><CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2"><CardTitle className="text-sm text-muted-foreground">钱包账户数</CardTitle><Wallet className="size-4 text-primary"/></CardHeader><CardContent><p className="text-2xl font-semibold tabular-nums">{summary?summary.wallet_count.toLocaleString("en-US"):"—"}</p><p className="mt-1 text-xs text-muted-foreground">已建立的钱包账户</p></CardContent></Card>
   <Card><CardHeader><CardTitle className="text-sm text-muted-foreground">钱包余额合计</CardTitle></CardHeader><CardContent><p className="text-2xl font-semibold tabular-nums">{summary?money(summary.total_wallet_balance_usdt):"—"}</p><p className="mt-1 text-xs text-muted-foreground">当前钱包余额总和</p></CardContent></Card>
   <Card><CardHeader><CardTitle className="text-sm text-muted-foreground">累计正向流水</CardTitle></CardHeader><CardContent><p className="text-2xl font-semibold tabular-nums">{summary?money(summary.credit_total_usdt):"—"}</p><p className="mt-1 text-xs text-muted-foreground">所有正金额资金记录</p></CardContent></Card>
   <Card><CardHeader><CardTitle className="text-sm text-muted-foreground">累计负向流水</CardTitle></CardHeader><CardContent><p className="text-2xl font-semibold tabular-nums">{summary?money(summary.debit_total_usdt):"—"}</p><p className="mt-1 text-xs text-muted-foreground">所有负金额资金记录绝对值</p></CardContent></Card>
  </section>
  <Card><CardHeader><CardTitle className="text-base">全站资金流水</CardTitle><CardDescription>共 {total} 笔记录；只读查询，不提供人工余额修改入口。</CardDescription></CardHeader><CardContent className="space-y-4">
   {ledgerError&&<p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">{ledgerError}</p>}
   <form className="flex flex-col gap-2 sm:flex-row" onSubmit={e=>{e.preventDefault();search()}}><Input value={query} onChange={e=>setQuery(e.target.value)} placeholder="搜索邮箱、昵称、关联单号或说明"/><Button type="submit"><Search className="mr-2 size-4"/>搜索</Button></form>
   <div className="flex flex-wrap gap-2">{types.map(item=><Button key={item.value||"all"} size="sm" variant={entryType===item.value?"default":"outline"} onClick={()=>filterType(item.value)} disabled={loading}>{item.label}</Button>)}</div>
   {loading?<p className="py-10 text-center text-sm text-muted-foreground">正在读取资金流水…</p>:items.length?<div className="overflow-x-auto"><Table><TableHeader><TableRow><TableHead>用户</TableHead><TableHead>类型</TableHead><TableHead>说明</TableHead><TableHead>变动金额</TableHead><TableHead>变动后余额</TableHead><TableHead>关联单号</TableHead><TableHead>时间</TableHead></TableRow></TableHeader><TableBody>{items.map(item=><TableRow key={item.id}><TableCell><div className="font-medium">{item.display_name||"未设置昵称"}</div><div className="text-xs text-muted-foreground">{item.user_email}</div></TableCell><TableCell><Badge variant="outline">{item.entry_type}</Badge></TableCell><TableCell className="min-w-36">{item.description}</TableCell><TableCell className={item.amount_usdt>=0?"whitespace-nowrap font-medium text-emerald-600":"whitespace-nowrap font-medium text-destructive"}>{item.amount_usdt>0?"+":""}{money(item.amount_usdt)}</TableCell><TableCell className="whitespace-nowrap tabular-nums">{money(item.balance_after_usdt)}</TableCell><TableCell className="font-mono text-xs">{item.reference_id||"—"}</TableCell><TableCell className="whitespace-nowrap text-xs text-muted-foreground">{date(item.created_at)}</TableCell></TableRow>)}</TableBody></Table></div>:<p className="py-10 text-center text-sm text-muted-foreground">没有符合条件的资金流水。</p>}
   <div className="flex items-center justify-between border-t pt-3"><p className="text-xs text-muted-foreground">第 {Math.min(pages,Math.floor(offset/limit)+1)} / {pages} 页 · 共 {summary?.ledger_entry_count??"—"} 笔总流水</p><div className="flex gap-2"><Button size="sm" variant="outline" disabled={loading||offset===0} onClick={()=>void load(Math.max(0,offset-limit))}>上一页</Button><Button size="sm" variant="outline" disabled={loading||offset+limit>=total} onClick={()=>void load(offset+limit)}>下一页</Button></div></div>
  </CardContent></Card>
 </main>
}
