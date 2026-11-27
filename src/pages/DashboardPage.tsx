import { useCallback,useEffect,useMemo,useState } from "react"
import { Activity,Bot,Clock3,MessagesSquare,RefreshCw,ShieldCheck,UsersRound,ArrowUpRight,Radio,Wifi } from "lucide-react"
import { currentUser,listBots,listGroups,listGroupEvents,type User,type TelegramBot,type TelegramGroup,type TelegramGroupMessageEvent } from "@/lib/api"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card,CardContent,CardDescription,CardHeader,CardTitle } from "@/components/ui/card"

type ActivityItem = TelegramGroupMessageEvent & { groupTitle: string }
const hourLabels = Array.from({length:12},(_,i)=>i)

function formatTime(value:string|null|undefined){
 if(!value)return "时间未知"
 const date=new Date(value)
 if(Number.isNaN(date.getTime()))return "时间未知"
 return new Intl.DateTimeFormat("zh-CN",{month:"2-digit",day:"2-digit",hour:"2-digit",minute:"2-digit"}).format(date)
}
function relativeTime(value:string|null|undefined){
 if(!value)return "暂无活动"
 const time=new Date(value).getTime()
 if(!Number.isFinite(time))return "暂无活动"
 const minutes=Math.max(0,Math.floor((Date.now()-time)/60000))
 if(minutes<1)return "刚刚活跃"
 if(minutes<60)return minutes+" 分钟前"
 const hours=Math.floor(minutes/60)
 if(hours<24)return hours+" 小时前"
 return Math.floor(hours/24)+" 天前"
}
function eventSummary(event:TelegramGroupMessageEvent){
 if(event.message_text?.trim())return event.message_text.replace(/\s+/g," ").slice(0,90)
 const labels:Record<string,string>={new_members:"成员加入",photo:"发送了图片",video:"发送了视频",sticker:"发送了贴纸",voice:"发送了语音",document:"发送了文件"}
 return labels[event.message_type]||("产生了"+(event.message_type||"新")+"事件")
}
function MetricCard({title,value,detail,icon:Icon,index}:{title:string;value:string|number;detail:string;icon:typeof Bot;index:number}){
 return <Card className="dashboard-enter dashboard-metric overflow-hidden border-border/70 bg-card/90 shadow-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-lg" style={{animationDelay:(index*75)+"ms"}}>
  <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
   <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
   <span className="flex size-9 items-center justify-center rounded-xl bg-primary/8 text-primary"><Icon className="size-[18px]"/></span>
  </CardHeader>
  <CardContent><p className="dashboard-number text-3xl font-semibold tracking-tight">{value}</p><p className="mt-1 text-xs text-muted-foreground">{detail}</p></CardContent>
 </Card>
}
export default function DashboardPage(){
 const [user,setUser]=useState<User|null>(null)
 const [bots,setBots]=useState<TelegramBot[]>([])
 const [groups,setGroups]=useState<TelegramGroup[]>([])
 const [events,setEvents]=useState<ActivityItem[]>([])
 const [error,setError]=useState("")
 const [loading,setLoading]=useState(true)
 const [refreshing,setRefreshing]=useState(false)
 const [lastUpdated,setLastUpdated]=useState<Date|null>(null)
 const refresh=useCallback(async()=>{
  setRefreshing(true)
  try{
   const [u,b,g]=await Promise.all([currentUser(),listBots(),listGroups()])
   setUser(u);setBots(b.items||[]);setGroups(g.items||[])
   const recentGroups=(g.items||[]).filter(group=>group.status==="active").slice(0,6)
   const activity=await Promise.allSettled(recentGroups.map(group=>listGroupEvents(group.id,30)))
   const collected:ActivityItem[]=[]
   activity.forEach((result,index)=>{
    if(result.status==="fulfilled")for(const event of result.value.items||[])collected.push({...event,groupTitle:recentGroups[index].title||"未命名群组"})
   })
   collected.sort((a,b)=>new Date(b.received_at).getTime()-new Date(a.received_at).getTime())
   setEvents(collected.slice(0,12));setError("");setLastUpdated(new Date())
  }catch(e){setError(e instanceof Error?e.message:"仪表盘加载失败")}
  finally{setLoading(false);setRefreshing(false)}
 },[])
 useEffect(()=>{
  void refresh()
  const timer=window.setInterval(()=>{if(document.visibilityState==="visible")void refresh()},30000)
  return()=>window.clearInterval(timer)
 },[refresh])
 const activeBots=bots.filter(bot=>bot.status==="active").length
 const activeGroups=groups.filter(group=>group.status==="active").length
 const chart=useMemo(()=>{
  const now=Date.now()
  const buckets=hourLabels.map((_,index)=>{
   const start=now-(11-index)*60*60*1000
   return {label:new Date(start).toLocaleTimeString("zh-CN",{hour:"2-digit",minute:undefined}),count:0}
  })
  for(const event of events){
   const age=now-new Date(event.received_at).getTime()
   const index=11-Math.floor(age/3600000)
   if(index>=0&&index<12)buckets[index].count++
  }
  const max=Math.max(1,...buckets.map(bucket=>bucket.count))
  return buckets.map(bucket=>({...bucket,height:Math.max(7,Math.round(bucket.count/max*100))}))
 },[events,lastUpdated])
 const recentGroups=useMemo(()=>[...groups].sort((a,b)=>new Date(b.last_seen_at||0).getTime()-new Date(a.last_seen_at||0).getTime()).slice(0,5),[groups])
 return <main className="mx-auto w-full max-w-[1440px] space-y-6 p-4 sm:p-6 lg:p-8">
  <section className="dashboard-hero dashboard-enter relative isolate overflow-hidden rounded-2xl border border-primary/15 bg-gradient-to-br from-primary/[0.10] via-card to-card p-5 sm:p-7">
   <div className="dashboard-orb pointer-events-none absolute -right-16 -top-24 -z-10 size-72 rounded-full bg-primary/10 blur-3xl"/>
   <div className="dashboard-orb dashboard-orb-delay pointer-events-none absolute bottom-[-7rem] right-[28%] -z-10 size-56 rounded-full bg-sky-400/10 blur-3xl"/>
   <div className="flex flex-col gap-5 sm:flex-row sm:items-center sm:justify-between">
    <div className="min-w-0">
     <div className="mb-3 flex flex-wrap items-center gap-2"><Badge variant="secondary" className="gap-1.5"><span className="dashboard-live-dot size-1.5 rounded-full bg-emerald-500"/>实时管理中心</Badge><span className="text-xs text-muted-foreground">每 30 秒自动同步</span></div>
     <h1 className="text-2xl font-semibold tracking-tight sm:text-3xl">欢迎回来，{user?.display_name||user?.email||"用户"}</h1>
     <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">集中查看 Bot 接入、群组状态与最新群内动态。所有统计均来自当前账号实际接入的数据。</p>
    </div>
    <div className="flex shrink-0 items-center gap-3">
     <div className="hidden text-right sm:block"><p className="text-xs text-muted-foreground">最近同步</p><p className="mt-1 text-sm font-medium">{lastUpdated?lastUpdated.toLocaleTimeString("zh-CN",{hour:"2-digit",minute:"2-digit",second:"2-digit"}):"正在连接..."}</p></div>
     <Button variant="outline" onClick={()=>void refresh()} disabled={refreshing} className="bg-background/80"><RefreshCw className={"mr-2 size-4 "+(refreshing?"animate-spin":"")}/>{refreshing?"同步中":"刷新数据"}</Button>
    </div>
   </div>
  </section>
  {error&&<div role="alert" className="rounded-xl border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">{error}。请检查网络后刷新。</div>}
  <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
   <MetricCard title="已接入机器人" value={loading?"—":bots.length} detail={loading?"正在读取数据":activeBots+" 个运行中 · "+(bots.length-activeBots)+" 个已停用"} icon={Bot} index={0}/>
   <MetricCard title="已发现群组" value={loading?"—":groups.length} detail={loading?"正在读取数据":activeGroups+" 个活跃群组"} icon={UsersRound} index={1}/>
   <MetricCard title="近期消息事件" value={loading?"—":events.length} detail="最近采集的群消息记录" icon={MessagesSquare} index={2}/>
   <MetricCard title="账号安全" value={user?.role==="admin"?"管理员":"正常"} detail={user?.role==="admin"?"拥有平台管理权限":"当前账号身份已验证"} icon={ShieldCheck} index={3}/>
  </section>
  <section className="grid gap-4 xl:grid-cols-[1.45fr_0.85fr]">
   <Card className="dashboard-enter overflow-hidden border-border/70 shadow-sm">
    <CardHeader className="flex flex-row items-start justify-between gap-3">
     <div><CardTitle className="flex items-center gap-2 text-base"><Activity className="size-4 text-primary"/>群消息活跃趋势</CardTitle><CardDescription className="mt-1">最近 12 小时已采集消息分布 · 最多读取 6 个活跃群组</CardDescription></div>
     <Badge variant="outline" className="shrink-0 gap-1.5"><Radio className="size-3 text-emerald-500"/>动态</Badge>
    </CardHeader>
    <CardContent>
     <div className="flex h-36 items-end gap-2 border-b border-border/70 pb-2 sm:gap-3">
      {chart.map((bar,index)=><div key={index} className="flex h-full min-w-0 flex-1 flex-col items-center justify-end gap-2"><div className="flex h-full w-full items-end"><div title={bar.count+" 条消息"} className="dashboard-bar w-full rounded-t-md bg-primary/75 transition-[height] duration-700 ease-out hover:bg-primary" style={{height:bar.height+"%",animationDelay:(index*35)+"ms"}}/></div><span className="text-[10px] text-muted-foreground">{bar.label}</span></div>)}
     </div>
     <div className="mt-3 flex items-center justify-between text-xs text-muted-foreground"><span>较早</span><span>最近一小时</span></div>
    </CardContent>
   </Card>
   <Card className="dashboard-enter border-border/70 shadow-sm">
    <CardHeader><CardTitle className="flex items-center gap-2 text-base"><Wifi className="size-4 text-primary"/>系统连接状态</CardTitle><CardDescription>当前账号可访问的数据服务</CardDescription></CardHeader>
    <CardContent className="space-y-4">
     <div className="flex items-center justify-between"><div className="flex items-center gap-2 text-sm"><span className="dashboard-live-dot size-2 rounded-full bg-emerald-500"/>网页 API</div><Badge variant="secondary">已连接</Badge></div>
     <div className="flex items-center justify-between"><div className="flex items-center gap-2 text-sm"><Bot className="size-4 text-muted-foreground"/>Bot 接入</div><span className="text-sm font-medium tabular-nums">{activeBots}/{bots.length} 运行中</span></div>
     <div className="flex items-center justify-between"><div className="flex items-center gap-2 text-sm"><UsersRound className="size-4 text-muted-foreground"/>群组同步</div><span className="text-sm font-medium tabular-nums">{activeGroups}/{groups.length} 活跃</span></div>
     <div className="rounded-lg bg-muted/50 p-3 text-xs leading-5 text-muted-foreground">数据自动刷新间隔为 30 秒。切换到其他标签页时会暂停轮询，返回后继续同步。</div>
    </CardContent>
   </Card>
  </section>
  <section className="grid gap-4 xl:grid-cols-[1.15fr_0.85fr]">
   <Card className="dashboard-enter border-border/70 shadow-sm">
    <CardHeader className="flex flex-row items-center justify-between gap-3"><div><CardTitle className="flex items-center gap-2 text-base"><MessagesSquare className="size-4 text-primary"/>最新群组动态</CardTitle><CardDescription className="mt-1">跨群组聚合的最新消息事件</CardDescription></div><Button asChild variant="ghost" size="sm"><a href="/dashboard/groups">查看群组<ArrowUpRight className="ml-1 size-4"/></a></Button></CardHeader>
    <CardContent>
     {events.length===0?<div className="flex flex-col items-center justify-center rounded-xl border border-dashed py-10 text-center"><MessagesSquare className="mb-3 size-8 text-muted-foreground/50"/><p className="text-sm font-medium">{loading?"正在加载群组动态":"暂时没有采集到消息"}</p><p className="mt-1 text-xs text-muted-foreground">Bot 加入群组并收到新消息后，动态会显示在这里。</p></div>:<div className="space-y-1">{events.slice(0,7).map((event,index)=><div key={event.id} className="dashboard-feed-item flex gap-3 rounded-lg px-2 py-3 transition-colors hover:bg-muted/50" style={{animationDelay:(index*45)+"ms"}}><div className="relative flex w-8 shrink-0 justify-center"><span className="flex size-8 items-center justify-center rounded-full bg-primary/8 text-primary"><MessagesSquare className="size-4"/></span>{index!==Math.min(events.length,7)-1&&<span className="absolute top-9 h-5 w-px bg-border"/>}</div><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1"><p className="truncate text-sm font-medium">{event.sender_display_name||event.sender_username||("用户 "+event.telegram_user_id)}</p><span className="shrink-0 text-[11px] text-muted-foreground">{relativeTime(event.received_at)}</span></div><p className="mt-0.5 truncate text-xs text-muted-foreground">{event.groupTitle} · {eventSummary(event)}</p></div></div>)}</div>}
    </CardContent>
   </Card>
   <Card className="dashboard-enter border-border/70 shadow-sm">
    <CardHeader><CardTitle className="flex items-center gap-2 text-base"><Clock3 className="size-4 text-primary"/>群组运行概况</CardTitle><CardDescription>最近有活动的群组</CardDescription></CardHeader>
    <CardContent className="space-y-2">
     {recentGroups.length===0?<div className="py-8 text-center text-sm text-muted-foreground">{loading?"正在加载群组":"尚未接入群组"}</div>:recentGroups.map((group,index)=><a key={group.id} href={"/dashboard/groups/"+group.id+"/events"} className="dashboard-feed-item flex items-center gap-3 rounded-xl border border-transparent p-3 transition-all hover:border-border hover:bg-muted/50" style={{animationDelay:(index*55)+"ms"}}><span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-muted"><UsersRound className="size-4 text-muted-foreground"/></span><span className="min-w-0 flex-1"><span className="block truncate text-sm font-medium">{group.title||"未命名群组"}</span><span className="mt-0.5 block text-xs text-muted-foreground">{relativeTime(group.last_seen_at)}</span></span><Badge variant={group.status==="active"?"secondary":"outline"}>{group.status==="active"?"活跃":"已离开"}</Badge></a>)}
     <Button asChild variant="outline" className="mt-2 w-full"><a href="/dashboard/groups">管理全部群组<ArrowUpRight className="ml-2 size-4"/></a></Button>
    </CardContent>
   </Card>
  </section>
  <footer className="flex flex-wrap items-center justify-between gap-2 border-t pt-4 text-xs text-muted-foreground"><span>TG-Group · Telegram 社群管理平台</span><span>数据更新时间：{lastUpdated?formatTime(lastUpdated.toISOString()):"等待首次同步"}</span></footer>
 </main>
}
