import { useCallback, useEffect, useState, type FormEvent } from "react"
import { AlertCircle, Check, Pencil, Plus, RefreshCw } from "lucide-react"
import { createAdminPlan, getAgentAccount, listAdminPlans, listPlans, purchasePlanWithWallet, updateAdminPlan, type Plan } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"

const emptyForm = { code: "", name: "", description: "", price: "0", days: "30", bots: "1", groups: "1", sort: "0", features: "{}", enabled: true }

export default function PlanCatalogPage({ admin = false }: { admin?: boolean }) {
  const [items,setItems]=useState<Plan[]>([])
  const [loading,setLoading]=useState(true)
  const [saving,setSaving]=useState(false)
  const [error,setError]=useState("")
  const [notice,setNotice]=useState("")
  const [editingID,setEditingID]=useState("")
  const [walletBalance,setWalletBalance]=useState<number|null>(null)
  const [purchasingID,setPurchasingID]=useState("")
  const [form,setForm]=useState(emptyForm)

  const load=useCallback(async()=>{
    setLoading(true);setError("")
    try { if(admin){const result=await listAdminPlans();setItems(result.items)}else{const [result,account]=await Promise.all([listPlans(),getAgentAccount()]);setItems(result.items);setWalletBalance(account.wallet_balance_usdt)} }
    catch(err){setError(err instanceof Error?err.message:"读取套餐目录失败")}
    finally{setLoading(false)}
  },[admin])
  useEffect(()=>{void load()},[load])

  async function buy(item:Plan){
    setPurchasingID(item.id);setError("");setNotice("")
    try{
      const order=await purchasePlanWithWallet(item.id,crypto.randomUUID())
      setNotice(`购买成功，订单 ${order.order_no} 已支付，套餐授权已自动开通/续期。`)
      await load()
    }catch(err){setError(err instanceof Error?err.message:"套餐购买失败")}
    finally{setPurchasingID("")}
  }

  function edit(item:Plan){
    setEditingID(item.id)
    setForm({code:item.code,name:item.name,description:item.description,price:String(item.price_usdt),days:String(item.duration_days),bots:String(item.bot_limit),groups:String(item.group_limit),sort:String(item.sort_order),features:JSON.stringify(item.features??{},null,2),enabled:item.enabled})
    setNotice("")
  }
  function reset(){setEditingID("");setForm(emptyForm);setNotice("")}

  async function submit(event:FormEvent<HTMLFormElement>){
    event.preventDefault();setError("");setNotice("")
    const values=[form.price,form.days,form.bots,form.groups,form.sort]
    if(values.some(value=>!/^\d+$/.test(value)||!Number.isSafeInteger(Number(value)))||Number(form.days)<1){
      setError("价格、期限、额度和排序必须是有效整数；期限至少为 1 天。");return
    }
    let features:Record<string,unknown>
    try{const parsed:unknown=JSON.parse(form.features||"{}");if(!parsed||typeof parsed!=="object"||Array.isArray(parsed))throw new Error();features=parsed as Record<string,unknown>}
    catch{setError("功能配置必须是有效的 JSON 对象。");return}
    setSaving(true)
    try{
      const input={name:form.name,description:form.description,price_usdt:Number(form.price),duration_days:Number(form.days),bot_limit:Number(form.bots),group_limit:Number(form.groups),features,sort_order:Number(form.sort),enabled:form.enabled}
      if(editingID){await updateAdminPlan(editingID,input);setNotice("套餐配置已保存。")}
      else{await createAdminPlan({code:form.code,...input});setNotice("套餐已创建。")}
      reset();setNotice(editingID?"套餐配置已保存。":"套餐已创建。");await load()
    }catch(err){setError(err instanceof Error?err.message:"保存套餐失败")}
    finally{setSaving(false)}
  }

  return <main className="mx-auto w-full max-w-[1440px] space-y-6 p-4 sm:p-6 lg:p-8">
    <div className="flex flex-wrap items-end justify-between gap-3">
      <div><div className="mb-2 flex gap-2"><Badge variant="secondary">{admin?"平台管理":"业务管理"}</Badge><Badge variant="outline">真实套餐目录</Badge></div><h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">{admin?"平台套餐配置":"套餐目录"}</h1><p className="mt-2 text-sm text-muted-foreground">{admin?"配置全站销售套餐、整数 USDT 价格、有效期和 Bot/群组额度。停用套餐会从用户目录隐藏。":`查看当前开放销售的套餐。钱包余额：${walletBalance===null?"读取中…":new Intl.NumberFormat("en-US").format(walletBalance)+" USDT"}。余额购买成功后会自动开通或续期授权。`}</p></div>
      <Button variant="outline" onClick={()=>void load()} disabled={loading}><RefreshCw className={`mr-2 size-4 ${loading?"animate-spin":""}`}/>刷新</Button>
    </div>
    {error&&<div role="alert" className="flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive"><AlertCircle className="mt-0.5 size-4 shrink-0"/>{error}</div>}
    {notice&&<p role="status" className="rounded-lg border border-emerald-600/20 bg-emerald-600/5 p-3 text-sm">{notice}</p>}
    <div className={admin?"grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)]":""}>
      <Card><CardHeader><CardTitle className="text-base">可用套餐</CardTitle><CardDescription>{admin?`共 ${items.length} 个套餐配置；停用项仅管理员可见。`:`当前开放 ${items.length} 个套餐。`}</CardDescription></CardHeader><CardContent>
        {loading?<p className="py-8 text-center text-sm text-muted-foreground">正在读取套餐…</p>:items.length?<div className="grid gap-3 sm:grid-cols-2">{items.map(item=><div key={item.id} className="rounded-xl border p-4">
          <div className="flex items-start justify-between gap-2"><div><p className="font-semibold">{item.name}</p><p className="mt-1 text-xs text-muted-foreground">{item.code}</p></div><Badge variant={item.enabled?"default":"secondary"}>{item.enabled?"销售中":"已停用"}</Badge></div>
          <p className="mt-4 text-2xl font-semibold tabular-nums">{new Intl.NumberFormat("en-US").format(item.price_usdt)} <span className="text-xs font-medium text-muted-foreground">USDT</span></p>
          <p className="mt-1 text-xs text-muted-foreground">有效期 {item.duration_days} 天 · Bot {item.bot_limit===0?"不限":item.bot_limit} 个 · 群组 {item.group_limit===0?"不限":item.group_limit} 个</p>
          {item.description&&<p className="mt-3 text-sm leading-6 text-muted-foreground">{item.description}</p>}
          {admin?<Button size="sm" variant="outline" className="mt-4" onClick={()=>edit(item)}><Pencil className="mr-2 size-3.5"/>编辑套餐</Button>:<Button size="sm" className="mt-4" disabled={purchasingID===item.id||walletBalance===null} onClick={()=>void buy(item)}>{purchasingID===item.id?"购买处理中…":`余额购买 · ${item.price_usdt} USDT`}</Button>}
        </div>)}</div>:<p className="py-8 text-center text-sm text-muted-foreground">{error?"套餐目录读取失败，请刷新重试。":"暂无开放套餐。"}</p>}
      </CardContent></Card>
      {admin&&<Card><CardHeader><div className="flex items-center justify-between gap-3"><div><CardTitle className="text-base">{editingID?"编辑套餐":"新增套餐"}</CardTitle><CardDescription>金额为整数 USDT；额度 0 表示不限。</CardDescription></div>{editingID?<Button variant="ghost" size="sm" onClick={reset}>取消编辑</Button>:<Plus className="size-4 text-muted-foreground"/>}</div></CardHeader><CardContent>
        <form className="space-y-3" onSubmit={event=>void submit(event)}>
          <div className="space-y-1.5"><Label htmlFor="plan-code">套餐标识</Label><Input id="plan-code" value={form.code} disabled={!!editingID} onChange={e=>setForm(v=>({...v,code:e.target.value}))} required maxLength={40} placeholder="例如 starter"/></div>
          <div className="space-y-1.5"><Label htmlFor="plan-name">套餐名称</Label><Input id="plan-name" value={form.name} onChange={e=>setForm(v=>({...v,name:e.target.value}))} required maxLength={80}/></div>
          <div className="space-y-1.5"><Label htmlFor="plan-description">套餐说明</Label><Textarea id="plan-description" value={form.description} onChange={e=>setForm(v=>({...v,description:e.target.value}))} maxLength={2000} rows={2}/></div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5"><Label htmlFor="plan-price">价格（USDT）</Label><Input id="plan-price" inputMode="numeric" value={form.price} onChange={e=>setForm(v=>({...v,price:e.target.value}))}/></div>
            <div className="space-y-1.5"><Label htmlFor="plan-days">有效期（天）</Label><Input id="plan-days" inputMode="numeric" value={form.days} onChange={e=>setForm(v=>({...v,days:e.target.value}))}/></div>
            <div className="space-y-1.5"><Label htmlFor="plan-bots">Bot 数量（0 不限）</Label><Input id="plan-bots" inputMode="numeric" value={form.bots} onChange={e=>setForm(v=>({...v,bots:e.target.value}))}/></div>
            <div className="space-y-1.5"><Label htmlFor="plan-groups">群组数量（0 不限）</Label><Input id="plan-groups" inputMode="numeric" value={form.groups} onChange={e=>setForm(v=>({...v,groups:e.target.value}))}/></div>
            <div className="space-y-1.5"><Label htmlFor="plan-sort">排序</Label><Input id="plan-sort" inputMode="numeric" value={form.sort} onChange={e=>setForm(v=>({...v,sort:e.target.value}))}/></div>
          </div>
          <div className="space-y-1.5"><Label htmlFor="plan-features">功能配置（JSON 对象）</Label><Textarea id="plan-features" rows={5} value={form.features} onChange={e=>setForm(v=>({...v,features:e.target.value}))} className="font-mono text-xs"/></div>
          {editingID&&<div className="flex items-center justify-between rounded-lg border p-3"><Label htmlFor="plan-enabled">开放销售</Label><Switch id="plan-enabled" checked={form.enabled} onCheckedChange={enabled=>setForm(v=>({...v,enabled}))}/></div>}
          <Button className="w-full" type="submit" disabled={saving}>{saving?"保存中…":editingID?<><Check className="mr-2 size-4"/>保存配置</>:<><Plus className="mr-2 size-4"/>创建套餐</>}</Button>
        </form>
      </CardContent></Card>}
    </div>
  </main>
}
