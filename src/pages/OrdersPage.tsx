import { useCallback, useEffect, useState } from "react"
import { RefreshCw, Receipt } from "lucide-react"
import { listAccountOrders, type AccountOrder } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

const limit=10
function date(value:string){const parsed=new Date(value);return Number.isNaN(parsed.getTime())?"—":parsed.toLocaleString("zh-CN")}
export default function OrdersPage(){
 const [items,setItems]=useState<AccountOrder[]>([])
 const [total,setTotal]=useState(0)
 const [offset,setOffset]=useState(0)
 const [loading,setLoading]=useState(true)
 const [error,setError]=useState("")
 const load=useCallback(async(next=offset)=>{
  setLoading(true);setError("")
  try{const result=await listAccountOrders(limit,next);setItems(result.items);setTotal(result.total);setOffset(next)}
  catch(err){setError(err instanceof Error?err.message:"读取订单失败")}
  finally{setLoading(false)}
 },[offset])
 useEffect(()=>{void load(0)},[])
 const pages=Math.max(1,Math.ceil(total/limit))
 return <main className="mx-auto w-full max-w-[1440px] space-y-6 p-4 sm:p-6 lg:p-8">
  <div className="flex flex-wrap items-end justify-between gap-3"><div><Badge variant="secondary" className="mb-2">账户业务</Badge><h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">我的订单</h1><p className="mt-2 text-sm text-muted-foreground">查看套餐购买记录、支付方式和订单状态。订单按当前登录账号隔离。</p></div><Button variant="outline" onClick={()=>void load()} disabled={loading}><RefreshCw className={`mr-2 size-4 ${loading?"animate-spin":""}`}/>刷新</Button></div>
  {error&&<p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">{error}</p>}
  <Card><CardHeader><div className="flex items-center gap-3"><Receipt className="size-5 text-primary"/><div><CardTitle className="text-base">订单记录</CardTitle><CardDescription>共 {total} 笔订单</CardDescription></div></div></CardHeader><CardContent className="space-y-3">
   {loading?<p className="py-10 text-center text-sm text-muted-foreground">正在读取订单…</p>:items.length?<div className="space-y-2">{items.map(item=><div key={item.id} className="flex flex-wrap items-center justify-between gap-3 rounded-lg border p-4"><div className="min-w-0"><p className="font-medium">{item.plan_name}</p><p className="mt-1 break-all font-mono text-xs text-muted-foreground">{item.order_no}</p><p className="mt-1 text-xs text-muted-foreground">{date(item.created_at)} · {item.payment_method==="wallet"?"钱包余额":"其他渠道"}</p></div><div className="text-right"><p className="font-semibold tabular-nums">{item.amount_usdt.toLocaleString("en-US")} USDT</p><Badge variant={item.status==="paid"?"default":item.status==="pending"?"secondary":"destructive"}>{item.status==="paid"?"已支付":item.status==="pending"?"待支付":item.status==="cancelled"?"已关闭":"失败"}</Badge></div></div>)}</div>:<div className="py-12 text-center text-sm text-muted-foreground">暂无订单。可以先到套餐目录选择套餐。</div>}
   <div className="flex items-center justify-between border-t pt-3"><p className="text-xs text-muted-foreground">第 {Math.min(pages,Math.floor(offset/limit)+1)} / {pages} 页</p><div className="flex gap-2"><Button size="sm" variant="outline" disabled={loading||offset===0} onClick={()=>void load(Math.max(0,offset-limit))}>上一页</Button><Button size="sm" variant="outline" disabled={loading||offset+limit>=total} onClick={()=>void load(offset+limit)}>下一页</Button></div></div>
  </CardContent></Card>
 </main>
}
