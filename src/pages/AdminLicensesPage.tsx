import { useCallback, useEffect, useState } from "react"
import { RefreshCw, Search } from "lucide-react"
import { listAdminLicenses, updateAdminLicenseStatus, type AdminLicense } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

const limit=20
const statuses=[{value:"",label:"全部"},{value:"active",label:"生效中"},{value:"expired",label:"已到期"},{value:"suspended",label:"已暂停"}]
function date(value:string){const d=new Date(value);return Number.isNaN(d.getTime())?"—":d.toLocaleString("zh-CN")}
export default function AdminLicensesPage(){
 const [items,setItems]=useState<AdminLicense[]>([])
 const [total,setTotal]=useState(0)
 const [offset,setOffset]=useState(0)
 const [status,setStatus]=useState("")
 const [query,setQuery]=useState("")
 const [appliedQuery,setAppliedQuery]=useState("")
 const [loading,setLoading]=useState(true)
 const [busyID,setBusyID]=useState("")
 const [error,setError]=useState("")
 const [notice,setNotice]=useState("")
 const load=useCallback(async(next=0,nextStatus=status,nextQuery=appliedQuery)=>{
  setLoading(true);setError("")
  try{const result=await listAdminLicenses({status:nextStatus,q:nextQuery,limit,offset:next});setItems(result.items);setTotal(result.total);setOffset(next)}
  catch(err){setError(err instanceof Error?err.message:"读取全站授权失败")}
  finally{setLoading(false)}
 },[status,appliedQuery])
 useEffect(()=>{void load(0,"","")},[])
 function filterStatus(value:string){setStatus(value);void load(0,value,appliedQuery)}
 function search(){const value=query.trim();setAppliedQuery(value);void load(0,status,value)}
 async function changeStatus(item:AdminLicense){
  const next=item.status==="suspended"?"active":"suspended"
  if(!window.confirm(`确认将 ${item.plan_name}（${item.user_email}）授权设为${next==="active"?"生效":"暂停"}？`))return
  setBusyID(item.id);setError("");setNotice("")
  try{await updateAdminLicenseStatus(item.id,next);setNotice("授权状态已更新。");await load()}
  catch(err){setError(err instanceof Error?err.message:"更新授权状态失败")}
  finally{setBusyID("")}
 }
 const pages=Math.max(1,Math.ceil(total/limit))
 return <main className="mx-auto w-full max-w-[1440px] space-y-6 p-4 sm:p-6 lg:p-8">
  <div className="flex flex-wrap items-end justify-between gap-3"><div><Badge variant="secondary" className="mb-2">平台管理</Badge><h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">全站授权</h1><p className="mt-2 text-sm text-muted-foreground">查看用户套餐授权期限与额度。管理员可暂停或恢复未到期授权；到期授权必须通过购买续期，不能直接恢复。</p></div><Button variant="outline" onClick={()=>void load()} disabled={loading}><RefreshCw className={`mr-2 size-4 ${loading?"animate-spin":""}`}/>刷新</Button></div>
  {error&&<p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">{error}</p>}
  {notice&&<p role="status" className="rounded-lg border border-emerald-600/20 bg-emerald-600/5 p-3 text-sm">{notice}</p>}
  <Card><CardHeader><CardTitle className="text-base">授权记录</CardTitle><CardDescription>共 {total} 项授权</CardDescription></CardHeader><CardContent className="space-y-4">
   <form className="flex flex-col gap-2 sm:flex-row" onSubmit={e=>{e.preventDefault();search()}}><Input value={query} onChange={e=>setQuery(e.target.value)} placeholder="搜索用户邮箱、昵称或套餐名称"/><Button type="submit"><Search className="mr-2 size-4"/>搜索</Button></form>
   <div className="flex flex-wrap gap-2">{statuses.map(item=><Button key={item.value||"all"} size="sm" variant={status===item.value?"default":"outline"} onClick={()=>filterStatus(item.value)} disabled={loading}>{item.label}</Button>)}</div>
   {loading?<p className="py-10 text-center text-sm text-muted-foreground">正在读取授权…</p>:items.length?<div className="overflow-x-auto"><Table><TableHeader><TableRow><TableHead>用户</TableHead><TableHead>套餐</TableHead><TableHead>授权期限</TableHead><TableHead>资源额度</TableHead><TableHead>状态</TableHead><TableHead>操作</TableHead></TableRow></TableHeader><TableBody>{items.map(item=><TableRow key={item.id}><TableCell><div className="font-medium">{item.display_name||"未设置昵称"}</div><div className="text-xs text-muted-foreground">{item.user_email}</div><div className="font-mono text-[10px] text-muted-foreground">{item.user_id}</div></TableCell><TableCell>{item.plan_name}</TableCell><TableCell className="whitespace-nowrap text-xs"><div>开始：{date(item.starts_at)}</div><div className="mt-1">到期：{date(item.expires_at)}</div></TableCell><TableCell className="whitespace-nowrap text-xs">Bot：{item.bot_limit===0?"不限":item.bot_limit}<br/>群组：{item.group_limit===0?"不限":item.group_limit}</TableCell><TableCell><Badge variant={item.status==="active"?"default":item.status==="suspended"?"destructive":"secondary"}>{item.status==="active"?"生效中":item.status==="suspended"?"已暂停":"已到期"}</Badge></TableCell><TableCell>{item.status==="expired"?<span className="text-xs text-muted-foreground">需购买续期</span>:<Button size="sm" variant="outline" disabled={busyID===item.id} onClick={()=>void changeStatus(item)}>{busyID===item.id?"处理中…":item.status==="suspended"?"恢复":"暂停"}</Button>}</TableCell></TableRow>)}</TableBody></Table></div>:<p className="py-10 text-center text-sm text-muted-foreground">没有符合条件的授权。</p>}
   <div className="flex items-center justify-between border-t pt-3"><p className="text-xs text-muted-foreground">第 {Math.min(pages,Math.floor(offset/limit)+1)} / {pages} 页</p><div className="flex gap-2"><Button size="sm" variant="outline" disabled={loading||offset===0} onClick={()=>void load(Math.max(0,offset-limit))}>上一页</Button><Button size="sm" variant="outline" disabled={loading||offset+limit>=total} onClick={()=>void load(offset+limit)}>下一页</Button></div></div>
  </CardContent></Card>
 </main>
}
