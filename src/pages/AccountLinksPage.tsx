import {useEffect,useState} from "react"
import {Link2,Loader2,Unlink} from "lucide-react"
import {createTelegramLink,getTelegramIdentity,unlinkTelegramIdentity,type TelegramIdentityStatus} from "@/lib/api"
import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"
import {Card,CardContent,CardDescription,CardHeader,CardTitle} from "@/components/ui/card"

export default function AccountLinksPage(){
 const [identity,setIdentity]=useState<TelegramIdentityStatus|null>(null)
 const [loading,setLoading]=useState(true)
 const [busy,setBusy]=useState(false)
 const [error,setError]=useState("")
 const [notice,setNotice]=useState("")
 async function refresh(){setError("");try{setIdentity(await getTelegramIdentity())}catch(e){setError(e instanceof Error?e.message:"读取账号关联状态失败")}finally{setLoading(false)}}
 useEffect(()=>{void refresh()},[])
 async function link(){setBusy(true);setError("");try{const result=await createTelegramLink();window.location.assign(result.url)}catch(e){setError(e instanceof Error?e.message:"创建 Telegram 绑定链接失败");setBusy(false)}}
 async function unlink(){if(!window.confirm("确定解除当前网站账号与 Telegram 身份的关联吗？"))return;setBusy(true);setError("");setNotice("");try{await unlinkTelegramIdentity();await refresh();setNotice("Telegram 身份已解除关联。")}catch(e){setError(e instanceof Error?e.message:"解除关联失败")}finally{setBusy(false)}}
 const tg=identity?.telegram
 const label=tg?.username?"@"+tg.username:[tg?.first_name,tg?.last_name].filter(Boolean).join(" ")||"Telegram 用户"
 return <main className="mx-auto w-full max-w-5xl space-y-6 p-4 sm:p-6 lg:p-8"><div className="space-y-2"><Badge variant="secondary">账号设置</Badge><h1 className="text-2xl font-semibold tracking-tight">账号关联</h1><p className="text-sm text-muted-foreground">管理网站账号与 Telegram 身份的关联。该功能属于账号设置，不混入密码和登录设备安全页面。</p></div>{error&&<p role="alert" className="text-sm text-destructive">{error}</p>}{notice&&<p role="status" className="text-sm text-emerald-600">{notice}</p>}<Card><CardHeader><div className="flex items-center gap-3"><span className="flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary"><Link2 className="size-5"/></span><div><CardTitle>Telegram 身份</CardTitle><CardDescription>绑定后可通过平台 Telegram Bot 查询账号，并使用账号对应的权限添加群管 Bot。</CardDescription></div></div></CardHeader><CardContent className="flex flex-wrap items-center justify-between gap-4"><div>{loading?<p className="text-sm text-muted-foreground">正在读取关联状态...</p>:identity?.linked?<><div className="flex items-center gap-2"><Badge>已关联</Badge><span className="text-sm font-medium">{label}</span></div><p className="mt-1 text-xs text-muted-foreground">Telegram ID：{tg?.id}</p></>:<><Badge variant="outline">尚未关联</Badge><p className="mt-2 text-sm text-muted-foreground">你可以通过平台 Bot 的一次性链接关联 Telegram。</p></>}</div><div className="flex gap-2">{identity?.linked?<Button variant="outline" onClick={unlink} disabled={busy}><Unlink className="mr-2 size-4"/>解除关联</Button>:<Button onClick={link} disabled={busy||loading}>{busy?<Loader2 className="mr-2 size-4 animate-spin"/>:<Link2 className="mr-2 size-4"/>}关联 Telegram</Button>}<Button variant="ghost" onClick={()=>void refresh()} disabled={busy||loading}>刷新</Button></div></CardContent></Card></main>
}
