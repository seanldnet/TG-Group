import { useCallback, useEffect, useState } from "react"
import { RefreshCw, Search, Users } from "lucide-react"

import { listAdminUsers, setAdminUserStatus, type AdminUser } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"

export default function AdminUsersPage() {
  const [items,setItems]=useState<AdminUser[]>([])
  const [search,setSearch]=useState("")
  const [appliedSearch,setAppliedSearch]=useState("")
  const [total,setTotal]=useState(0)
  const [offset,setOffset]=useState(0)
  const [loading,setLoading]=useState(true)
  const [busyID,setBusyID]=useState("")
  const [error,setError]=useState("")
  const limit=25

  const refresh=useCallback(async(nextOffset=offset,nextSearch=appliedSearch)=>{
    setLoading(true);setError("")
    try { const result=await listAdminUsers(nextSearch,limit,nextOffset);setItems(result.items);setTotal(result.total);setOffset(nextOffset) }
    catch(cause){setError(cause instanceof Error?cause.message:"读取用户列表失败")}
    finally{setLoading(false)}
  },[offset,appliedSearch])

  useEffect(()=>{void refresh(0,"")},[]) // load once on entry

  async function searchUsers(){const value=search.trim();setAppliedSearch(value);setOffset(0);await refresh(0,value)}
  async function changeStatus(item:AdminUser){
    const next=item.status==="active"?"locked":"active"
    if(!window.confirm(\`确认将 \${item.email} 的账号状态改为 \${next==="locked"?"锁定":"正常"}？\`))return
    setBusyID(item.id);setError("")
    try{await setAdminUserStatus(item.id,next);await refresh()}
    catch(cause){setError(cause instanceof Error?cause.message:"更新用户状态失败")}
    finally{setBusyID("")}
  }

  return <main className="mx-auto w-full max-w-7xl space-y-6 p-4 sm:p-6 lg:p-8">
    <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
      <div><Badge variant="secondary" className="mb-3">平台管理</Badge><h1 className="text-3xl font-semibold tracking-tight">用户管理</h1><p className="mt-2 text-sm text-muted-foreground">查看平台注册账号、Telegram 绑定、租户和 Bot 资产；用户密码及凭证不会展示。</p></div>
      <Button variant="outline" onClick={()=>void refresh()} disabled={loading}><RefreshCw className={\`mr-2 size-4 \${loading?"animate-spin":""}\`}/>刷新</Button>
    </div>
    {error&&<p role="alert" className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}</p>}
    <Card>
      <CardHeader><div className="flex items-center gap-3"><Users className="size-5 text-primary"/><div><CardTitle>平台用户</CardTitle><CardDescription>共 {total} 个账号，按注册时间倒序排列</CardDescription></div></div></CardHeader>
      <CardContent className="space-y-4">
        <form className="flex flex-col gap-2 sm:flex-row" onSubmit={e=>{e.preventDefault();void searchUsers()}}>
          <Input value={search} onChange={e=>setSearch(e.target.value)} placeholder="搜索邮箱、昵称、用户 ID、Telegram ID 或用户名"/>
          <Button type="submit"><Search className="mr-2 size-4"/>搜索</Button>
        </form>
        {loading?<p className="py-10 text-center text-sm text-muted-foreground">正在加载用户...</p>:items.length===0?<p className="py-10 text-center text-sm text-muted-foreground">没有找到符合条件的用户</p>:
          <div className="overflow-x-auto"><Table><TableHeader><TableRow><TableHead>用户</TableHead><TableHead>平台角色</TableHead><TableHead>Telegram</TableHead><TableHead>租户</TableHead><TableHead>Bot 数量</TableHead><TableHead>注册时间</TableHead><TableHead>状态</TableHead><TableHead>操作</TableHead></TableRow></TableHeader>
          <TableBody>{items.map(item=><TableRow key={item.id}>
            <TableCell><div className="font-medium">{item.display_name||"未设置昵称"}</div><div className="text-xs text-muted-foreground">{item.email}</div><div className="font-mono text-[10px] text-muted-foreground">{item.id}</div></TableCell>
            <TableCell><Badge variant={item.role==="admin"?"default":item.role==="agent"?"secondary":"outline"}>{item.role==="admin"?"管理员":item.role==="agent"?"代理":"用户"}</Badge></TableCell>
            <TableCell><div>{item.telegram_username?\`@\${item.telegram_username}\`:"未绑定"}</div><div className="font-mono text-xs text-muted-foreground">{item.telegram_id??"—"}</div></TableCell>
            <TableCell><div>{item.tenant_name}</div><div className="text-xs text-muted-foreground">{item.tenant_status}</div></TableCell>
            <TableCell>{item.bot_count}</TableCell>
            <TableCell className="whitespace-nowrap text-xs text-muted-foreground">{new Date(item.created_at).toLocaleString()}</TableCell>
            <TableCell><Badge variant={item.status==="active"?"default":"destructive"}>{item.status==="active"?"正常":item.status==="locked"?"已锁定":"已删除"}</Badge></TableCell>
            <TableCell>{item.role==="admin"||item.status==="deleted"?<span className="text-xs text-muted-foreground">不可操作</span>:<Button size="sm" variant="outline" disabled={busyID===item.id} onClick={()=>void changeStatus(item)}>{item.status==="active"?"锁定":"解锁"}</Button>}</TableCell>
          </TableRow>)}</TableBody></Table></div>}
        <div className="flex items-center justify-between border-t pt-4 text-sm text-muted-foreground"><span>显示 {items.length?offset+1:0}–{offset+items.length} / {total}</span><div className="flex gap-2"><Button size="sm" variant="outline" disabled={offset===0||loading} onClick={()=>void refresh(Math.max(0,offset-limit))}>上一页</Button><Button size="sm" variant="outline" disabled={offset+limit>=total||loading} onClick={()=>void refresh(offset+limit)}>下一页</Button></div></div>
      </CardContent>
    </Card>
  </main>
}
