import { useCallback, useEffect, useState } from "react"
import { RefreshCw, ShieldCheck } from "lucide-react"
import { listAccountLicenses, type AccountLicense } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

function date(value:string){const parsed=new Date(value);return Number.isNaN(parsed.getTime())?"—":parsed.toLocaleString("zh-CN")}
export default function LicensesPage(){
 const [items,setItems]=useState<AccountLicense[]>([])
 const [loading,setLoading]=useState(true)
 const [error,setError]=useState("")
 const load=useCallback(async()=>{setLoading(true);setError("");try{const result=await listAccountLicenses();setItems(result.items)}catch(err){setError(err instanceof Error?err.message:"读取授权失败")}finally{setLoading(false)}},[])
 useEffect(()=>{void load()},[load])
 return <main className="mx-auto w-full max-w-[1440px] space-y-6 p-4 sm:p-6 lg:p-8">
  <div className="flex flex-wrap items-end justify-between gap-3"><div><Badge variant="secondary" className="mb-2">账户业务</Badge><h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">我的授权</h1><p className="mt-2 text-sm text-muted-foreground">查看已购买套餐的授权期限与资源额度。授权状态按服务端记录和到期时间展示。</p></div><Button variant="outline" onClick={()=>void load()} disabled={loading}><RefreshCw className={`mr-2 size-4 ${loading?"animate-spin":""}`}/>刷新</Button></div>
  {error&&<p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">{error}</p>}
  <Card><CardHeader><div className="flex items-center gap-3"><ShieldCheck className="size-5 text-primary"/><div><CardTitle className="text-base">授权列表</CardTitle><CardDescription>共 {items.length} 项授权</CardDescription></div></div></CardHeader><CardContent>
   {loading?<p className="py-10 text-center text-sm text-muted-foreground">正在读取授权…</p>:items.length?<div className="space-y-3">{items.map(item=><div key={item.id} className="rounded-lg border p-4"><div className="flex flex-wrap items-center justify-between gap-2"><div className="font-medium">{item.plan_name}</div><Badge variant={item.status==="active"?"default":item.status==="suspended"?"destructive":"secondary"}>{item.status==="active"?"生效中":item.status==="suspended"?"已暂停":"已到期"}</Badge></div><div className="mt-3 grid gap-3 text-sm sm:grid-cols-2"><div><p className="text-xs text-muted-foreground">生效时间</p><p className="mt-1">{date(item.starts_at)}</p></div><div><p className="text-xs text-muted-foreground">到期时间</p><p className="mt-1">{date(item.expires_at)}</p></div><div><p className="text-xs text-muted-foreground">Bot 额度</p><p className="mt-1">{item.bot_limit===0?"不限":item.bot_limit}</p></div><div><p className="text-xs text-muted-foreground">群组额度</p><p className="mt-1">{item.group_limit===0?"不限":item.group_limit}</p></div></div></div>)}</div>:<p className="py-12 text-center text-sm text-muted-foreground">暂无授权。购买套餐后，系统会在钱包扣款事务中自动开通。</p>}
  </CardContent></Card>
 </main>
}
