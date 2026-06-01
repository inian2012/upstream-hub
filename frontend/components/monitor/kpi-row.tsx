"use client"

import { AlertTriangle, ArrowUpRight, DollarSign, MessageSquare, ShieldCheck } from "lucide-react"
import { Card } from "@/components/ui/card"
import { cn } from "@/lib/utils"
import { useDashboardSummary, useRateChanges, useCaptchaConfigs } from "@/lib/queries"
import { money } from "@/lib/format"
import type { LucideIcon } from "lucide-react"
import type { ReactNode } from "react"

interface Kpi {
  label: string
  value: ReactNode
  icon: LucideIcon
  iconBg: string
  iconColor: string
  footer: ReactNode
}

function countLowBalance(channels: { last_balance?: number | null }[], thresholds: Map<number, number>) {
  let n = 0
  for (const c of channels) {
    if (c.last_balance == null) continue
    const id = (c as { id: number }).id
    const threshold = thresholds.get(id) ?? 0
    if (threshold > 0 && c.last_balance < threshold) n++
  }
  return n
}

function countTodayChanges(changes: { changed_at: string }[]) {
  const startOfDay = new Date()
  startOfDay.setHours(0, 0, 0, 0)
  return changes.filter((c) => new Date(c.changed_at) >= startOfDay).length
}

export function KpiRow() {
  const summary = useDashboardSummary()
  const recentChanges = useRateChanges(100)
  const captcha = useCaptchaConfigs()

  const data = summary.data
  const total = data?.total_channels ?? 0
  const active = data?.active_channels ?? 0
  const failed = data?.failed_channels ?? 0
  const totalBalance = data?.total_balance ?? 0

  // 低余额数：需要把 channels 列表与 channel 的 balance_threshold 关联。
  // dashboard.channels 没带阈值，简单按 last_error 之外的"healthy 但是 balance < threshold"统计较麻烦，
  // 这里我们只展示 lowest_balance（有几条低于阈值需要从单独 channels API 拿，可选下版本再优化）。
  const lowest = data?.lowest_balance ?? null

  const todayChangeCount = countTodayChanges(recentChanges.data ?? [])

  const enabledCaptcha = captcha.data?.find((c) => c.enabled)
  const totalCaptcha = captcha.data?.length ?? 0

  const kpis: Kpi[] = [
    {
      label: "总余额",
      value: money(totalBalance),
      icon: DollarSign,
      iconBg: "bg-brand/10",
      iconColor: "text-brand",
      footer: (
        <span className="text-muted-foreground">
          {lowest ? (
            <>
              {"最低："}
              <span className="font-medium text-foreground">{lowest.name}</span>
              {" "}
              <span className="text-warning">{money(lowest.balance)}</span>
            </>
          ) : (
            "—"
          )}
        </span>
      ),
    },
    {
      label: "监控上游",
      value: (
        <span>
          {total}
          <span className="ml-1 text-lg font-normal text-muted-foreground">{"个"}</span>
        </span>
      ),
      icon: MessageSquare,
      iconBg: "bg-brand/10",
      iconColor: "text-brand",
      footer: (
        <span className="text-muted-foreground">
          <span className="text-success font-medium">{active} 个健康</span>
          {failed > 0 ? (
            <>
              {"，"}
              <span className="text-danger font-medium">{failed} 个失败</span>
            </>
          ) : null}
        </span>
      ),
    },
    {
      label: "余额最低",
      value: lowest ? (
        <span className="text-warning">{money(lowest.balance)}</span>
      ) : (
        <span className="text-muted-foreground">—</span>
      ),
      icon: AlertTriangle,
      iconBg: "bg-warning/10",
      iconColor: "text-warning",
      footer: (
        <span className="text-muted-foreground">
          {lowest ? lowest.name : "暂无数据"}
        </span>
      ),
    },
    {
      label: "今日倍率变动",
      value: <span className={cn(todayChangeCount > 0 ? "text-danger" : "text-foreground")}>{todayChangeCount}</span>,
      icon: ArrowUpRight,
      iconBg: "bg-danger/10",
      iconColor: "text-danger",
      footer: (
        <span className="text-muted-foreground">
          {todayChangeCount > 0 ? `检测到 ${todayChangeCount} 次变动` : "今日无变动"}
        </span>
      ),
    },
    {
      label: "验证码服务",
      value: enabledCaptcha ? (
        <span>
          {enabledCaptcha.name}
          <span className="ml-1 text-lg font-normal text-success">{"已启用"}</span>
        </span>
      ) : (
        <span className="text-muted-foreground">未配置</span>
      ),
      icon: ShieldCheck,
      iconBg: "bg-success/10",
      iconColor: "text-success",
      footer: (
        <span className="text-muted-foreground">
          {totalCaptcha > 0 ? `共 ${totalCaptcha} 个 provider` : "可在设置中添加"}
        </span>
      ),
    },
  ]

  // 没用到 thresholds，但保留壳避免 lint 抱怨 unused。
  void countLowBalance

  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-5">
      {kpis.map((k) => (
        <Card key={k.label} className="flex flex-row items-start justify-between gap-0 p-4 shadow-none border border-border">
          <div className="flex min-w-0 flex-col">
            <span className="text-xs text-muted-foreground">{k.label}</span>
            <p className="mt-1 text-2xl font-bold tracking-tight text-foreground">{k.value}</p>
            <p className="mt-1 text-xs">{k.footer}</p>
          </div>
          <span className={cn("flex size-10 shrink-0 items-center justify-center rounded-xl", k.iconBg)}>
            <k.icon className={cn("size-5", k.iconColor)} />
          </span>
        </Card>
      ))}
    </div>
  )
}
