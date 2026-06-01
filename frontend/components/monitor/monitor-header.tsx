import { useEffect, useState } from "react"
import { useTheme } from "next-themes"
import { Activity, ChevronDown, LogOut, RefreshCw, Sun, Moon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import { useAuth } from "@/lib/auth-context"
import { useTriggerRefresh } from "@/lib/refresh-context"

export function MonitorHeader() {
  const { theme, setTheme } = useTheme()
  const { username, logout } = useAuth()
  const refresh = useTriggerRefresh()
  const [mounted, setMounted] = useState(false)
  const [syncing, setSyncing] = useState(false)

  useEffect(() => setMounted(true), [])

  function handleRefresh() {
    setSyncing(true)
    refresh()
    setTimeout(() => setSyncing(false), 800)
  }

  return (
    <header className="sticky top-0 z-20 border-b border-border bg-background/95 backdrop-blur-sm">
      <div className="mx-auto flex h-14 max-w-[1440px] items-center justify-between gap-4 px-5">
        {/* left: logo + title */}
        <div className="flex items-center gap-2.5">
          <div className="flex size-8 items-center justify-center rounded-lg bg-foreground text-background">
            <Activity className="size-4" strokeWidth={2.5} />
          </div>
          <h1 className="text-base font-semibold tracking-tight text-foreground">Upstream-hub</h1>
        </div>

        {/* right: actions */}
        <div className="flex items-center gap-3">
          {/* refresh */}
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={syncing}
            className="gap-1.5 border-border bg-background text-foreground hover:bg-muted"
          >
            <RefreshCw className={cn("size-3.5", syncing && "animate-spin")} />
            {"刷新"}
          </Button>

          {/* theme toggle */}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            className="gap-1.5 border-border bg-background text-foreground hover:bg-muted"
            aria-label="切换主题"
          >
            {mounted && theme === "dark" ? (
              <>
                <Moon className="size-3.5" />
                {"深色"}
              </>
            ) : (
              <>
                <Sun className="size-3.5" />
                {"浅色"}
              </>
            )}
            <ChevronDown className="size-3 text-muted-foreground" />
          </Button>

          {/* logout */}
          <Button
            variant="outline"
            size="sm"
            onClick={logout}
            className="gap-1.5 border-border bg-background text-foreground hover:bg-muted"
            aria-label="退出登录"
            title={username ? `${username} · 退出登录` : "退出登录"}
          >
            <LogOut className="size-3.5" />
            <span className="hidden sm:inline">{username ?? "退出"}</span>
          </Button>
        </div>
      </div>
    </header>
  )
}
