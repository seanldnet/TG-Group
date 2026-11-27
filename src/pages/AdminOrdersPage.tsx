import { useCallback, useEffect, useState } from "react"
import { RefreshCw, Search } from "lucide-react"
import { listAdminOrders, type AdminOrder } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

const limit=20
const statuses=[{value:"",label:"全部"},{value:"paid",label:"已支付"},{value:"pending",label:"待支付"},{value:"cancelled",label:"已关闭"},{value:"failed",label:"失败"}]
function date(value:string){const d=new Date(value);return Number.isNaN(d.getTime())?"—":d.toLocaleString("zh-CN")}
export default function AdminOrdersPage(){
 const [items,setItems]=useState<AdminOrder[]>([])
 const [total,setTotal]=useState(0)
 const [offset,setOffset]=useState(0)
 const [status,setStatus]=useState("")
 const [query,setQuery]=useState("")
 const [appliedQuery,setAppliedQuery]=useState("")
 const [loading,setLoading]=useState(true)
 const [error,setError]=useState("")
 const load=useCallback(async(next=0,nextStatus=status,nextQuery=appliedQuery)=>{
  setLoading(true);setError("")
  try{const result=await listAdminOrders({status:nextStatus,q:nextQuery,limit,offset:next});setItems(result.items);setTotal(result.total);setOffset(next)}
  catch(err){setError(err instanceof Error?err.message:"读取平台订单失败")}
  finally{setLoading(false)}
 },[status,appliedQuery])
 useEffect(()=>{void load(0,"","")},[])
 function filterStatus(value:string){setStatus(value);void load(0,value,appliedQuery)}
 function search(){const value=query.trim();setAppliedQuery(value);void load(0,status,value)}
 const pages=Math.max(1,Math.ceil(total/limit))
 return <main className="mx-auto w-full max-w-[1440px] space-y-6 p-4 sm:p-6 lg:p-8">
  <div className="flex flex-wrap items-end justify-between gap-3"><div><Badge variant="secondary" className="mb-2">平台管理</Badge><h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">平台订单</h1><p className="mt-2 text-sm text-muted-foreground">查询全站套餐订单、用户、金额和支付状态。订单数据仅供核查，不提供人工标记支付成功操作。</p></div><Button variant="outline" onClick={()=>void load()} disabled={loading}><RefreshCw className={`mr-2 size-4 ${loading?"animate-spin":""}`}/>刷新</Button></div>
  {error&&<p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">{error}</p>}
  <Card><CardHeader><CardTitle className="text-base">订单记录</CardTitle><CardDescription>共 {total} 笔订单</CardDescription></CardHeader><CardContent className="space-y-4">
   <form className="flex flex-col gap-2 sm:flex-row" onSubmit={e=>{e.preventDefault();search()}}><Input value={query} onChange={e=>setQuery(e.target.value)} placeholder="搜索订单号、用户邮箱或昵称"/><Button type="submit"><Search className="mr-2 size-4"/>搜索</Button></form>
   <div className="flex flex-wrap gap-2">{statuses.map(item=><Button key={item.value||"all"} size="sm" variant={status===item.value?"default":"outline"} onClick={()=>filterStatus(item.value)} disabled={loading}>{item.label}</Button>)}</div>
   {loading?<p className="py-10 text-center text-sm text-muted-foreground">正在读取订单…</p>:items.length?<div className="overflow-x-auto"><Table><TableHeader><TableRow><TableHead>订单号</TableHead><TableHead>用户</TableHead><TableHead>套餐</TableHead><TableHead>金额</TableHead><TableHead>渠道</TableHead><TableHead>状态</TableHead><TableHead>创建时间</TableHead></TableRow></TableHeader><TableBody>{items.map(item=><TableRow key={item.id}><TableCell className="font-mono text-xs">{item.order_no}</TableCell><TableCell><div className="font-medium">{item.display_name||"未设置昵称"}</div><div className="text-xs text-muted-foreground">{item.user_email}</div></TableCell><TableCell>{item.plan_name}</TableCell><TableCell className="whitespace-nowrap tabular-nums">{item.amount_usdt.toLocaleString("en-US")} USDT</TableCell><TableCell>{item.payment_method==="wallet"?"钱包余额":item.payment_method}</TableCell><TableCell><Badge variant={item.status==="paid"?"default":item.status==="pending"?"secondary":"destructive"}>{item.status==="paid"?"已支付":item.status==="pending"?"待支付":item.status==="cancelled"?"已关闭":"失败"}</Badge></TableCell><TableCell className="whitespace-nowrap text-xs text-muted-foreground">{date(item.created_at)}</TableCell></TableRow>)}</TableBody></Table></div>:<p className="py-10 text-center text-sm text-muted-foreground">没有符合条件的订单。</p>}
   <div className="flex items-center justify-between border-t pt-3"><p className="text-xs text-muted-foreground">第 {Math.min(pages,Math.floor(offset/limit)+1)} / {pages} 页</p><div className="flex gap-2"><Button size="sm" variant="outline" disabled={loading||offset===0} onClick={()=>void load(Math.max(0,offset-limit))}>上一页</Button><Button size="sm" variant="outline" disabled={loading||offset+limit>=total} onClick={()=>void load(offset+limit)}>下一页</Button></div></div>
  </CardContent></Card>
 </main>
}
