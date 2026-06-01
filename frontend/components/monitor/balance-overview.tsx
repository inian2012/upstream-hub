"use client"

import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis, CartesianGrid } from "recharts"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useBalanceTrend, useDashboardSummary } from "@/lib/queries"
import { money } from "@/lib/format"
import { cn } from "@/lib/utils"

function formatY(n: number) {
  if (n === 0) return "$0"
  if (n >= 1000) return `$${(n / 1000).toFixed(0)}K`
  return `$${n.toFixed(0)}`
}

function formatDay(iso: string) {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return `${d.getMonth() + 1}月${d.getDate()}日`
}

interface TooltipPayloadItem { value: number }

function ChartTooltip({ active, payload, label }: { active?: boolean; payload?: TooltipPayloadItem[]; label?: string }) {
  if (!active || !payload?.length) return null
  return (
    <div className="rounded-lg border border-border bg-popover px-3 py-2 shadow-md">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-sm font-semibold text-foreground">
        {"$"}{payload[0].value.toLocaleString("en-US")}
      </p>
    </div>
  )
}

export function BalanceOverview() {
  const trend = useBalanceTrend(7)
  const summary = useDashboardSummary()

  const data = (trend.data ?? []).map((p) => ({
    day: formatDay(p.day),
    balance: p.balance,
  }))

  const channels = summary.data?.channels ?? []
  const yMax = Math.max(15000, ...data.map((d) => d.balance)) * 1.1

  return (
    <Card className="border border-border shadow-none">
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-base font-semibold">{"余额概览"}</CardTitle>
        <span className="text-xs text-muted-foreground">{"最近 7 天"}</span>
      </CardHeader>
      <CardContent>
        <div className="h-[240px] w-full">
          {trend.loading ? (
            <div className="flex h-full items-center justify-center text-xs text-muted-foreground">{"加载中…"}</div>
          ) : data.length === 0 ? (
            <div className="flex h-full items-center justify-center text-xs text-muted-foreground">
              {"暂无余额采样，等待下次扫描或手动刷新"}
            </div>
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={data} margin={{ top: 8, right: 12, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
                <XAxis
                  dataKey="day"
                  tickLine={false}
                  axisLine={false}
                  tick={{ fill: "var(--muted-foreground)", fontSize: 11 }}
                  dy={8}
                />
                <YAxis
                  tickLine={false}
                  axisLine={false}
                  width={48}
                  tick={{ fill: "var(--muted-foreground)", fontSize: 11 }}
                  tickFormatter={formatY}
                  domain={[0, yMax]}
                />
                <Tooltip content={<ChartTooltip />} cursor={{ stroke: "var(--border)", strokeDasharray: "4 4" }} />
                <Line
                  type="monotone"
                  dataKey="balance"
                  stroke="var(--brand)"
                  strokeWidth={2}
                  dot={{ r: 4, fill: "var(--background)", stroke: "var(--brand)", strokeWidth: 2 }}
                  activeDot={{ r: 5, fill: "var(--brand)", strokeWidth: 0 }}
                />
              </LineChart>
            </ResponsiveContainer>
          )}
        </div>

        {/* per-channel chips */}
        {channels.length > 0 ? (
          <div className="mt-3 flex flex-wrap items-center gap-x-5 gap-y-2 border-t border-border pt-3">
            {channels.map((c) => {
              const isFailed = !!c.last_error
              const isUnknown = c.last_balance == null
              return (
                <span key={c.id} className="inline-flex items-center gap-1.5 text-xs">
                  <span
                    className={cn(
                      "size-2 rounded-full",
                      isFailed ? "bg-danger" : isUnknown ? "bg-muted-foreground/40" : "bg-success",
                    )}
                  />
                  <span className="font-medium text-foreground">{c.name}</span>
                  <span className="tabular-nums text-muted-foreground">{money(c.last_balance)}</span>
                </span>
              )
            })}
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}
