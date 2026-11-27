import {useEffect,useState,type ReactNode} from "react"
import {Activity,Bot,ChevronRight,ClipboardList,KeyRound,LayoutDashboard,Link2,LogOut,Menu,Settings,ShieldCheck,UserCog,UsersRound,X} from "lucide-react"
import {clearToken,currentUser,logout,type User} from "@/lib/api"
import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"
import {Separator} from "@/components/ui/separator"

const roleLabel=(role?:string)=>role==="admin"?"管理员":"普通用户"

export default function DashboardLayout({children}:{children:ReactNode}){
 const [user,setUser]=useState<User|null>(null)
 const [menuOpen,setMenuOpen]=useState(false)
 const [signingOut,setSigningOut]=useState(false)
 const pathname=window.location.pathname
 useEffect(()=>{let active=true;currentUser().then(u=>{if(active)setUser(u)}).catch(()=>{clearToken();window.location.replace("/login")});return()=>{active=false}},[])
 async function handleLogout(){setSigningOut(true);try{await logout()}finally{clearToken();window.location.replace("/login")}}
 function item(href:string,label:string,Icon:any){
  const active=pathname===href||((href==="/dashboard/groups")&&pathname.startsWith("/dashboard/groups"))
  return <a key={href} href={href} onClick={()=>setMenuOpen(false)} className={"group relative flex items-center gap-3 rounded-lg px-3 py-2.5 text-[13px] font-medium transition-colors "+(active?"bg-primary/[0.08] text-primary":"text-muted-foreground hover:bg-muted/80 hover:text-foreground")}>
   {active&&<span className="absolute inset-y-2 left-0 w-[3px] rounded-r-full bg-primary"/>}
   <Icon className="size-[17px] shrink-0 opacity-85"/><span className="flex-1">{label}</span>{active&&<ChevronRight className="size-3.5 opacity-60"/>}
  </a>
 }
 const sidebar=<div className="flex h-full flex-col bg-background">
  <a href="/dashboard" className="flex h-[68px] items-center gap-3 px-5">
   <span className="flex size-9 items-center justify-center rounded-xl bg-primary text-primary-foreground"><ShieldCheck className="size-[19px]"/></span>
   <span><span className="block text-[15px] font-semibold tracking-tight">TG-Group</span><span className="mt-0.5 block text-[10px] tracking-[.12em] text-muted-foreground">GROUP MANAGEMENT</span></span>
  </a>
  <Separator/>
  <div className="px-3 pt-5">
   <p className="px-3 pb-2 text-[10px] font-semibold tracking-[.12em] text-muted-foreground/75">工作台</p>
   <nav className="space-y-1">{item("/dashboard","总览",LayoutDashboard)}{item("/dashboard/bots","机器人",Bot)}{item("/dashboard/groups","群组",UsersRound)}</nav>
  </div>
  <div className="mt-5 px-3">
   <p className="px-3 pb-2 text-[10px] font-semibold tracking-[.12em] text-muted-foreground/75">账号</p>
   <nav className="space-y-1">{item("/dashboard/account/links","账号关联",Link2)}{item("/dashboard/security/password","修改密码",KeyRound)}{item("/dashboard/security/sessions","登录设备",Activity)}{item("/dashboard/security/logs","安全日志",ClipboardList)}</nav>
  </div>
  {user?.role==="admin"&&<div className="mt-5 px-3"><p className="px-3 pb-2 text-[10px] font-semibold tracking-[.12em] text-muted-foreground/75">管理</p><nav className="space-y-1">{item("/dashboard/admin/users","用户管理",UserCog)}{item("/dashboard/platform","平台设置",Settings)}</nav></div>}
  <div className="mt-auto p-4"><Separator className="mb-4"/>
   <div className="mb-3 flex items-center gap-2.5">
    <div className="flex size-9 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold">{(user?.display_name||user?.email||"U").slice(0,1).toUpperCase()}</div>
    <div className="min-w-0 flex-1"><p className="truncate text-xs font-medium">{user?.display_name||user?.email||"正在加载"}</p><p className="mt-0.5 text-[11px] text-muted-foreground">{roleLabel(user?.role)}</p></div>
    <Badge variant="outline" className="px-1.5 py-0 text-[10px] font-normal">在线</Badge>
   </div>
   <Button variant="outline" size="sm" className="h-9 w-full justify-start text-xs" onClick={handleLogout} disabled={signingOut}><LogOut className="mr-2 size-3.5"/>{signingOut?"正在退出":"退出登录"}</Button>
  </div>
 </div>
 return <div className="min-h-svh bg-[#f6f7f9] dark:bg-background">
  <aside className="fixed inset-y-0 left-0 z-30 hidden w-[248px] border-r border-border/80 bg-background lg:block">{sidebar}</aside>
  <div className="lg:pl-[248px]">
   <header className="sticky top-0 z-20 flex h-[60px] items-center justify-between border-b border-border/80 bg-background/95 px-4 backdrop-blur-xl sm:px-6">
    <div className="flex items-center gap-3"><Button variant="ghost" size="icon" className="lg:hidden" onClick={()=>setMenuOpen(v=>!v)} aria-label="打开导航菜单">{menuOpen?<X className="size-5"/>:<Menu className="size-5"/>}</Button><span className="text-[13px] font-medium text-foreground/80">管理中心</span></div>
    <div className="flex items-center gap-3"><span className="hidden text-xs text-muted-foreground sm:inline">{user?.email||"账户"}</span><span className="flex size-8 items-center justify-center rounded-full border bg-muted/60 text-xs font-semibold">{(user?.display_name||user?.email||"U").slice(0,1).toUpperCase()}</span></div>
   </header>
   {menuOpen&&<div className="fixed inset-0 z-40 bg-black/30 backdrop-blur-[2px] lg:hidden" onClick={()=>setMenuOpen(false)}><aside className="h-full w-[280px] border-r bg-background shadow-2xl" onClick={e=>e.stopPropagation()}>{sidebar}</aside></div>}
   <div className="min-h-[calc(100svh-60px)] min-w-0">{children}</div>
  </div>
 </div>
}
