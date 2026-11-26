import { useEffect, useRef, useState } from "react"

import { getCaptchaConfig } from "@/lib/api"

type Props = {
  onToken: (token: string) => void
}

const CAP_WIDGET_VERSION = "0.1.58"
const CAP_WIDGET_SRC = `https://cdn.jsdelivr.net/npm/cap-widget@${CAP_WIDGET_VERSION}`

function loadCapWidget(): Promise<void> {
  if (customElements.get("cap-widget")) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const existing = document.querySelector<HTMLScriptElement>(`script[src="${CAP_WIDGET_SRC}"]`)
    const script = existing ?? document.createElement("script")
    script.addEventListener("load", () => resolve(), { once: true })
    script.addEventListener("error", () => reject(new Error("CAP 组件加载失败")), { once: true })
    if (!existing) {
      script.src = CAP_WIDGET_SRC
      script.async = true
      document.head.appendChild(script)
    }
  })
}

export function CaptchaField({ onToken }: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const onTokenRef = useRef(onToken)
  onTokenRef.current = onToken
  const [enabled, setEnabled] = useState(false)
  const [status, setStatus] = useState<"loading" | "disabled" | "ready" | "error">("loading")

  useEffect(() => {
    let cancelled = false
    let widget: HTMLElement | null = null
    const handleSolve = (event: Event) => {
      const token = (event as CustomEvent<{ token?: string }>).detail?.token
      if (typeof token === "string" && token.length > 0) onTokenRef.current(token)
    }
    const handleError = () => onTokenRef.current("")

    async function initialize() {
      try {
        const config = await getCaptchaConfig()
        if (cancelled) return
        if (!config.enabled) {
          setStatus("disabled")
          return
        }
        if (!config.endpoint) throw new Error("CAP 验证地址未配置")
        setEnabled(true)
        await loadCapWidget()
        if (cancelled || !containerRef.current) return
        widget = document.createElement("cap-widget")
        widget.setAttribute("data-cap-api-endpoint", config.endpoint)
        widget.setAttribute("data-cap-lang", "zh")
        widget.addEventListener("solve", handleSolve)
        widget.addEventListener("error", handleError)
        containerRef.current.replaceChildren(widget)
        setStatus("ready")
      } catch {
        if (!cancelled) setStatus("error")
      }
    }

    void initialize()
    return () => {
      cancelled = true
      if (widget) {
        widget.removeEventListener("solve", handleSolve)
        widget.removeEventListener("error", handleError)
        widget.remove()
      }
    }
  }, [])

  if (status === "disabled") return null

  return (
    <div className="space-y-2">
      <div className="text-sm font-medium">人机验证</div>
      {status === "loading" && (
        <p className="text-xs text-muted-foreground">正在加载验证组件...</p>
      )}
      {status === "error" && (
        <p role="alert" className="text-sm text-destructive">
          验证组件暂时不可用，请刷新页面或稍后重试。
        </p>
      )}
      <div ref={containerRef} className={enabled ? "min-h-14" : "hidden"} />
      {status === "ready" && (
        <p className="text-xs text-muted-foreground">请完成验证后再提交。</p>
      )}
    </div>
  )
}
