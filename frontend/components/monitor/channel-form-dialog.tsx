"use client"

import { useEffect, useState, type FormEvent } from "react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import type { Channel, ChannelType } from "@/lib/api-types"
import { apiFetch } from "@/lib/api"
import { useTriggerRefresh } from "@/lib/refresh-context"
import { useCaptchaConfigs } from "@/lib/queries"

interface ChannelFormDialogProps {
  open: boolean
  onOpenChange: (v: boolean) => void
  /** 编辑模式时传入；为空表示新增 */
  channel?: Channel | null
}

interface FormState {
  name: string
  type: ChannelType
  site_url: string
  username: string
  password: string
  balance_threshold: string
  monitor_enabled: boolean
  turnstile_enabled: boolean
  captcha_config_id: string // "" 表示不绑定
}

function initialState(c?: Channel | null): FormState {
  return {
    name: c?.name ?? "",
    type: c?.type ?? "newapi",
    site_url: c?.site_url ?? "",
    username: c?.username ?? "",
    password: "",
    balance_threshold: c?.balance_threshold != null ? String(c.balance_threshold) : "0",
    monitor_enabled: c?.monitor_enabled ?? true,
    turnstile_enabled: c?.turnstile_enabled ?? false,
    captcha_config_id: c?.captcha_config_id != null ? String(c.captcha_config_id) : "",
  }
}

export function ChannelFormDialog({ open, onOpenChange, channel }: ChannelFormDialogProps) {
  const [form, setForm] = useState<FormState>(() => initialState(channel))
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const refresh = useTriggerRefresh()
  const captchas = useCaptchaConfigs()

  // 打开 / 切换目标渠道时重置表单。
  useEffect(() => {
    if (open) {
      setForm(initialState(channel))
      setError(null)
    }
  }, [open, channel])

  const isEdit = !!channel

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setError(null)
    setSubmitting(true)
    try {
      const threshold = Number(form.balance_threshold)
      if (!Number.isFinite(threshold) || threshold < 0) {
        throw new Error("余额阈值必须是非负数")
      }
      const captchaConfigID =
        form.turnstile_enabled && form.captcha_config_id
          ? Number(form.captcha_config_id)
          : null
      if (form.turnstile_enabled && captchaConfigID == null) {
        throw new Error("启用 Turnstile 时必须选择一个打码 provider")
      }

      if (isEdit) {
        const body: Record<string, unknown> = {
          name: form.name,
          site_url: form.site_url,
          username: form.username,
          balance_threshold: threshold,
          monitor_enabled: form.monitor_enabled,
          turnstile_enabled: form.turnstile_enabled,
          captcha_config_id: captchaConfigID,
        }
        if (form.password) body.password = form.password
        await apiFetch(`/channels/${channel!.id}`, {
          method: "PUT",
          body: JSON.stringify(body),
        })
      } else {
        if (!form.password) throw new Error("新建时必须填写密码")
        await apiFetch(`/channels`, {
          method: "POST",
          body: JSON.stringify({
            name: form.name,
            type: form.type,
            site_url: form.site_url,
            username: form.username,
            password: form.password,
            balance_threshold: threshold,
            monitor_enabled: form.monitor_enabled,
            turnstile_enabled: form.turnstile_enabled,
            captcha_config_id: captchaConfigID,
          }),
        })
      }
      onOpenChange(false)
      refresh()
    } catch (e) {
      const err = e as Error
      setError(err.message || "保存失败")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? "编辑渠道" : "新增渠道"}</DialogTitle>
          <DialogDescription>
            {isEdit ? "修改后会清空已缓存的登录会话。" : "添加上游账号，开启监控后将按计划自动登录。"}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-3">
          <div className="space-y-1.5">
            <Label htmlFor="name">渠道名</Label>
            <Input
              id="name"
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
              required
              disabled={submitting}
            />
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="type">类型</Label>
            <Select
              value={form.type}
              onValueChange={(v) => setForm({ ...form, type: v as ChannelType })}
              disabled={isEdit || submitting}
            >
              <SelectTrigger id="type" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="newapi">NewAPI</SelectItem>
                <SelectItem value="sub2api">Sub2API</SelectItem>
              </SelectContent>
            </Select>
            {isEdit ? (
              <p className="text-[11px] text-muted-foreground">类型创建后不可修改</p>
            ) : null}
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="site_url">站点地址</Label>
            <Input
              id="site_url"
              placeholder="https://example.com"
              value={form.site_url}
              onChange={(e) => setForm({ ...form, site_url: e.target.value })}
              required
              disabled={submitting}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="username">账号 / 邮箱</Label>
              <Input
                id="username"
                value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })}
                required
                disabled={submitting}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="password">{isEdit ? "新密码 (留空不变)" : "密码"}</Label>
              <Input
                id="password"
                type="password"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
                required={!isEdit}
                disabled={submitting}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="threshold">余额阈值（低于此值发告警，0 = 不告警）</Label>
            <Input
              id="threshold"
              type="number"
              step="0.01"
              min="0"
              value={form.balance_threshold}
              onChange={(e) => setForm({ ...form, balance_threshold: e.target.value })}
              disabled={submitting}
            />
          </div>

          <div className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
            <div>
              <p className="text-sm font-medium">启用监控</p>
              <p className="text-xs text-muted-foreground">关闭后调度器不会扫描此渠道</p>
            </div>
            <Switch
              checked={form.monitor_enabled}
              onCheckedChange={(v) => setForm({ ...form, monitor_enabled: v })}
              disabled={submitting}
            />
          </div>

          <div className="flex items-center justify-between rounded-lg border border-border px-3 py-2">
            <div>
              <p className="text-sm font-medium">Turnstile 人机校验</p>
              <p className="text-xs text-muted-foreground">站点开启 Cloudflare Turnstile 时打开</p>
            </div>
            <Switch
              checked={form.turnstile_enabled}
              onCheckedChange={(v) => setForm({ ...form, turnstile_enabled: v })}
              disabled={submitting}
            />
          </div>

          {form.turnstile_enabled ? (
            <div className="space-y-1.5">
              <Label htmlFor="captcha-config">打码 provider</Label>
              <Select
                value={form.captcha_config_id}
                onValueChange={(v) => setForm({ ...form, captcha_config_id: v })}
                disabled={submitting}
              >
                <SelectTrigger id="captcha-config" className="w-full">
                  <SelectValue
                    placeholder={
                      captchas.data && captchas.data.length > 0
                        ? "选择 provider"
                        : "先到底部 [验证码服务] 卡片新增"
                    }
                  />
                </SelectTrigger>
                <SelectContent>
                  {(captchas.data ?? [])
                    .filter((c) => c.enabled)
                    .map((c) => (
                      <SelectItem key={c.id} value={String(c.id)}>
                        {c.name}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
              <p className="text-[11px] text-muted-foreground">
                {"siteKey 会自动从上游公开接口拉取，无需在此填写。"}
              </p>
            </div>
          ) : null}

          {error ? (
            <p className="text-sm text-destructive" role="alert">
              {error}
            </p>
          ) : null}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={submitting}>
              取消
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? "保存中…" : "保存"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
