import React, { useCallback, useEffect, useMemo, useState } from 'react'
import AdminFrame from '@/common/components/layout/AdminFrame'
import SurfacePanel from '@/common/components/layout/SurfacePanel'

const BALANCE_ENDPOINT = '/public/codex/balance'

function clampPercent(value) {
  const n = Number(value)
  if (!Number.isFinite(n)) return 0
  return Math.min(100, Math.max(0, n))
}

function fmtPercent(value) {
  const n = Number(value)
  if (!Number.isFinite(n)) return '-'
  return `${Math.round(n)}%`
}

function fmtDate(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString()
}

function fmtCredits(credits) {
  if (!credits) return '-'
  if (credits.unlimited) return '无限'
  if (credits.balance == null || credits.balance === '') return '0'
  return String(credits.balance)
}

function balanceStatusText(payload, loading) {
  if (loading) return '查询中'
  if (payload?.stale) return '缓存结果'
  if (payload?.status === 'ok') return '正常'
  return '-'
}

function rateLimitTitle(item) {
  if (!item) return 'Codex'
  if (item.limit_name) return item.limit_name
  if (item.limit_id === 'codex') return 'Codex'
  return item.limit_id || 'Codex'
}

function sortRateLimits(payload) {
  const byId = payload?.rate_limits_by_limit_id || {}
  return Object.values(byId).sort((a, b) => {
    if (a?.limit_id === 'codex') return -1
    if (b?.limit_id === 'codex') return 1
    return rateLimitTitle(a).localeCompare(rateLimitTitle(b))
  })
}

function LimitBar({ label, item }) {
  const remaining = clampPercent(item?.remaining_percent)
  const used = clampPercent(item?.used_percent)

  return (
    <div className="grid gap-2">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <span className="text-sm font-semibold text-[#365141]">{label}</span>
        <span className="text-sm text-[#7b8780]">
          {fmtPercent(remaining)} 剩余 / {fmtPercent(used)} 已用
        </span>
      </div>
      <div className="h-3 overflow-hidden rounded-full bg-[#e7efe9]">
        <div
          className="h-full rounded-full bg-[#238a43] transition-[width]"
          style={{ width: `${remaining}%` }}
        />
      </div>
      <div className="text-xs text-[#7b8780]">
        重置时间：{fmtDate(item?.resets_at_time)}
      </div>
    </div>
  )
}

function LimitCard({ item }) {
  return (
    <SurfacePanel variant="admin" className="p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-[#1f2d25]">
            {rateLimitTitle(item)}
          </h2>
          <p className="mt-1 text-sm text-[#7b8780]">
            {item?.limit_id || '-'} · {item?.plan_type || '未记录套餐'}
          </p>
        </div>
        <div className="rounded-md border border-[#dde8df] bg-[#fbfdfb] px-3 py-2 text-right">
          <div className="text-xs text-[#7b8780]">Credits</div>
          <div className="mt-0.5 text-lg font-bold text-[#1f2d25]">
            {fmtCredits(item?.credits)}
          </div>
        </div>
      </div>

      <div className="mt-5 grid gap-5 lg:grid-cols-2">
        <LimitBar label="5 小时额度" item={item?.primary} />
        <LimitBar label="每周额度" item={item?.secondary} />
      </div>
    </SurfacePanel>
  )
}

export default function AdminCodexBalancePage() {
  const [payload, setPayload] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const limits = useMemo(() => sortRateLimits(payload), [payload])

  const loadBalance = useCallback(async ({ signal } = {}) => {
    setLoading(true)
    setError('')

    try {
      const response = await fetch(BALANCE_ENDPOINT, {
        method: 'GET',
        headers: { Accept: 'application/json' },
        signal,
      })
      const data = await response.json().catch(() => null)

      if (!response.ok || data?.status !== 'ok') {
        setPayload(null)
        setError('Codex 余额查询失败，请稍后重试')
        return
      }

      setPayload(data)
    } catch (e) {
      if (e?.name === 'AbortError') return
      setPayload(null)
      setError('Codex 余额查询失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    loadBalance({ signal: controller.signal })
    return () => controller.abort()
  }, [loadBalance])

  return (
    <AdminFrame
      breadcrumb="用量统计 / Codex 余额"
      title="Codex 余额"
      description="查看当前服务器 Codex 登录态对应的额度余额、5 小时窗口和每周窗口；数据来自公开余额接口，不展示账号邮箱或 token。"
      actions={
        <>
          <a
            className="admin-button"
            href={BALANCE_ENDPOINT}
            target="_blank"
            rel="noreferrer noopener"
          >
            打开公开接口
          </a>
          <button
            type="button"
            className="admin-button admin-button-primary"
            disabled={loading}
            onClick={() => loadBalance()}
          >
            {loading ? '刷新中' : '刷新'}
          </button>
        </>
      }
    >
      {error ? (
        <div className="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
          {error}
        </div>
      ) : null}

      {payload?.stale ? (
        <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
          实时查询暂时失败，当前显示上次成功读取的 Codex 余额。
        </div>
      ) : null}

      <SurfacePanel variant="admin" className="p-5">
        <div className="grid gap-4 md:grid-cols-3">
          <div>
            <div className="text-sm text-[#7b8780]">接口状态</div>
            <div className="mt-1 text-2xl font-bold text-[#1f2d25]">
              {balanceStatusText(payload, loading)}
            </div>
          </div>
          <div>
            <div className="text-sm text-[#7b8780]">Credits remaining</div>
            <div className="mt-1 text-2xl font-bold text-[#1f2d25]">
              {fmtCredits(payload?.credits)}
            </div>
          </div>
          <div>
            <div className="text-sm text-[#7b8780]">更新时间</div>
            <div className="mt-2 break-words text-sm font-semibold text-[#1f2d25]">
              {fmtDate(payload?.fetched_at)}
            </div>
          </div>
        </div>
      </SurfacePanel>

      {loading && !payload ? (
        <SurfacePanel variant="admin" className="p-5">
          <div className="text-sm text-[#7b8780]">正在读取 Codex 余额...</div>
        </SurfacePanel>
      ) : null}

      {limits.length > 0 ? (
        <div className="grid gap-5 xl:grid-cols-2">
          {limits.map((item) => (
            <LimitCard
              key={item.limit_id || rateLimitTitle(item)}
              item={item}
            />
          ))}
        </div>
      ) : null}
    </AdminFrame>
  )
}
