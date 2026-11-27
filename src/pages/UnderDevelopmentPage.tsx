import { ArrowLeft, Clock3, Construction } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

const modules: Record<string, { title: string; description: string }> = {
  "/dashboard/agents": { title: "代理管理", description: "代理账号、上下级关系、代理权限与团队管理" },
  "/dashboard/agents/levels": { title: "代理等级", description: "代理等级、升级条件与专属权限" },
  "/dashboard/agents/commissions": { title: "佣金管理", description: "佣金明细、结算记录与佣金规则" },
  "/dashboard/agents/api-keys": { title: "代理 API", description: "代理密钥、接口权限与调用记录" },
  "/dashboard/plans": { title: "套餐管理", description: "套餐价格、有效期、功能额度与上下架" },
  "/dashboard/orders": { title: "订单管理", description: "订单列表、订单状态与支付详情" },
  "/dashboard/payments": { title: "支付管理", description: "支付渠道、支付记录与交易核验" },
  "/dashboard/licenses": { title: "授权管理", description: "授权状态、开通记录、续期与变更" },
  "/dashboard/finance": { title: "财务概览", description: "收入汇总、交易流水与财务统计" },
  "/dashboard/bot-sponsors": { title: "机器人赞助商", description: "按机器人或群组管理多个赞助商，配置名称、展示颜色、介绍文案、跳转链接和按钮；支持定时推送、新成员入群、群聊沉寂后重新活跃等触发条件同时启用，并分别设置触发间隔与展示频率。" },
  "/dashboard/ai": { title: "AI 智能审核", description: "AI 服务商、模型配置、审核策略与用量" },
  "/dashboard/tasks": { title: "定时任务", description: "定时公告、提醒任务、执行状态与失败记录" },
  "/dashboard/analytics": { title: "数据统计", description: "机器人、群组、成员、消息与处罚数据" },
  "/dashboard/audit": { title: "操作审计", description: "管理员操作、配置变更与业务记录" },
  "/dashboard/admin/agents": { title: "代理审核", description: "代理申请、代理状态与权限分配" },
  "/dashboard/admin/plans": { title: "平台套餐", description: "平台套餐配置、定价与可售状态" },
  "/dashboard/admin/orders": { title: "平台订单", description: "全站订单查询、支付状态与异常处理" },
  "/dashboard/admin/payments": { title: "支付渠道", description: "USDT、EPUSDT、GMPay 等支付渠道配置" },
  "/dashboard/admin/licenses": { title: "全站授权", description: "用户授权、套餐期限与授权变更" },
  "/dashboard/admin/ai": { title: "AI 服务配置", description: "平台 AI 服务接入与全局策略" },
  "/dashboard/admin/audit": { title: "平台审计", description: "全站管理操作与系统事件" },
}

export default function UnderDevelopmentPage({ pathname }: { pathname: string }) {
  const module = modules[pathname] || { title: "功能页面", description: "该功能入口已预留，页面正在开发中" }
  return <main className="mx-auto flex w-full max-w-5xl flex-col gap-6 p-4 sm:p-6 lg:p-8">
    <div className="flex items-center gap-2 text-sm text-muted-foreground"><span>管理中心</span><span>/</span><span className="text-foreground">{module.title}</span></div>
    <Card className="overflow-hidden">
      <CardHeader className="border-b bg-muted/20 p-6 sm:p-8">
        <div className="mb-3 flex items-center gap-2"><Badge variant="secondary">开发中</Badge><span className="text-xs text-muted-foreground">功能入口已开放</span></div>
        <CardTitle className="text-2xl">{module.title}</CardTitle>
        <CardDescription className="max-w-2xl text-sm leading-6">{module.description}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col items-center px-6 py-16 text-center sm:py-20">
        <div className="mb-5 flex size-16 items-center justify-center rounded-2xl bg-muted text-muted-foreground"><Construction className="size-8"/></div>
        <h2 className="text-lg font-semibold">页面正在开发中</h2>
        <p className="mt-2 max-w-md text-sm leading-6 text-muted-foreground">该模块已加入系统导航，相关功能完成后会在此页面开放。现有账户和业务数据不会受到影响。</p>
        <div className="mt-6 inline-flex items-center gap-2 text-xs text-muted-foreground"><Clock3 className="size-3.5"/>尚未开放操作</div>
        <Button asChild variant="outline" className="mt-7"><a href="/dashboard"><ArrowLeft className="mr-2 size-4"/>返回总览</a></Button>
      </CardContent>
    </Card>
  </main>
}
