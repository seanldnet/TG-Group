import {useEffect,useState} from "react"
import {Bot,Loader2,Save,Trash2} from "lucide-react"
import {clearPlatformTelegramBot,getPlatformTelegramBotStatus,savePlatformTelegramBot,type PlatformTelegramBotStatus} from "@/lib/api"
import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"
import {Card,CardContent,CardDescription,CardHeader,CardTitle} from "@/components/ui/card"
import {Input} from "@/components/ui/input"
import {Label} from "@/components/ui/label"

export default function PlatformSettingsPage(){
 const [status,setStatus]=useState<PlatformTelegramBotStatus|null>(null)
 const [token,setToken]=useState("")
 const [username,setUsername]=useState("")
 const [loading,setLoading]=useState(true)
 const [saving,setSaving]=useState(false)
 const [error,setError]=useState("")
 const [message,setMessage]=useState("")
 useEffect(()=>{let active=true;getPlatformTelegramBotStatus().then(s=>{if(active){setStatus(s);setUsername(s.username)}}).catch(e=>{if(active)setError(e instanceof Error?e.message:"读取平台设置失败")}).finally(()=>{if(active)setLoading(false)});return()=>{active=false}},[])
 async function save(){setError("");setMessage("");if(!token.trim()){setError("请输入平台 Telegram Bot Token");return}if(username.trim()&&!/^[A-Za-z0-9_]{5,32}$/.test(username.trim())){setError("Bot Username 需为 5-32 位字母、数字或下划线，且不包含 @");return}setSaving(true);try{const s=await savePlatformTelegramBot({token:token.trim(),username:username.trim()});setStatus(s);setToken("");setMessage("配置已加密保存，并已向运行管理器提交重启。请稍后确认运行状态。")}catch(e){setError(e instanceof Error?e.message:"保存配置失败")}finally{setSaving(false)}}
 async function refresh(){setError("");setLoading(true);try{const s=await getPlatformTelegramBotStatus();setStatus(s);setUsername(s.username)}catch(e){setError(e instanceof Error?e.message:"刷新状态失败")}finally{setLoading(false)}}
 async function clear(){if(!window.confirm("确定停用并清除平台 Telegram 注册 Bot 配置吗？"))return;setSaving(true);setError("");setMessage("");try{const s=await clearPlatformTelegramBot();setStatus(s);setUsername("");setToken("");setMessage("平台 Telegram Bot 已停用，已保存的加密 Token 已清除。")}catch(e){setError(e instanceof Error?e.message:"停用 Bot 失败")}finally{setSaving(false)}}
 return <main className="mx-auto w-full max-w-5xl space-y-6 p-4 sm:p-6 lg:p-8"><div className="space-y-2"><Badge variant="secondary">平台管理</Badge><h1 className="text-2xl font-semibold tracking-tight">平台设置</h1><p className="text-sm text-muted-foreground">管理平台注册与身份绑定 Bot。此设置仅平台管理员可访问，与用户自行接入的群管 Bot 分开管理。</p></div>{error&&<p role="alert" className="text-sm text-destructive">{error}</p>}{message&&<p role="status" className="text-sm text-emerald-600">{message}</p>}
 <Card><CardHeader><div className="flex items-center gap-3"><span className="flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary"><Bot className="size-5"/></span><div><CardTitle>Telegram 注册 / 身份 Bot</CardTitle><CardDescription>负责账号注册与身份绑定；日常操作通过 Telegram 按钮菜单完成。</CardDescription></div></div></CardHeader><CardContent className="space-y-5">
 <div className="flex flex-wrap items-center gap-2"><Badge variant={status?.configured?"default":"outline"}>{loading?"读取状态中":status?.configured?"已配置":"未配置"}</Badge><Badge variant={status?.running?"default":"secondary"}>{status?.running?"运行管理器已启动":"当前未运行"}</Badge>{status?.username&&<span className="text-sm text-muted-foreground">@{status.username}</span>}</div>
 <div className="grid gap-4 sm:grid-cols-2"><div className="space-y-2"><Label htmlFor="platform-bot-username">Bot Username</Label><Input id="platform-bot-username" placeholder="YourPlatformBot" value={username} onChange={e=>setUsername(e.target.value)}/><p className="text-xs text-muted-foreground">填写 Telegram Bot 用户名，不需要输入 @。</p></div><div className="space-y-2"><Label htmlFor="platform-bot-token">Bot Token</Label><Input id="platform-bot-token" type="password" autoComplete="new-password" placeholder={status?.configured?"已配置（不会回显），输入新 Token 可轮换":"输入 BotFather 提供的 Token"} value={token} onChange={e=>setToken(e.target.value)}/><p className="text-xs text-muted-foreground">Token 使用 AES-256-GCM 加密存储，接口不会返回明文。</p></div></div>
 <div className="flex flex-wrap gap-2"><Button onClick={save} disabled={saving||loading||!token.trim()}>{saving?<Loader2 className="mr-2 size-4 animate-spin"/>:<Save className="mr-2 size-4"/>}保存并重启 Bot</Button><Button variant="outline" onClick={()=>void refresh()} disabled={saving||loading}>刷新状态</Button>{status?.configured&&<Button variant="destructive" onClick={clear} disabled={saving}><Trash2 className="mr-2 size-4"/>停用并清除配置</Button>}</div>
 <p className="text-xs leading-5 text-muted-foreground">安全提示：保存后 Bot 会使用新配置重新启动；停用会取消当前轮询并删除数据库中的加密凭据。若服务重启，程序会从数据库读取加密配置恢复运行。</p>
 </CardContent></Card></main>
}
