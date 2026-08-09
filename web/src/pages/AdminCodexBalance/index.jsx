import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { AUTH_SCOPE } from '@/common/auth/auth'
import AdminFrame from '@/common/components/layout/AdminFrame'
import SurfacePanel from '@/common/components/layout/SurfacePanel'
import { ADMIN_BASE_PATH } from '@/common/utils/adminRpc'
import {
  buildCodexUsageOverview,
  calculateRateLimitPace,
  formatCodexUsageCost,
} from '@/common/utils/codexUsageStats'
import { getActionErrorMessage } from '@/common/utils/errorMessage'
import { JsonRpc } from '@/common/utils/jsonRpc'
import {
  DEFAULT_DAILY_USAGE_TIME_RANGE,
  getUsageTimeWindow,
  startOfLocalDayUnix,
} from '@/common/utils/usageTimeRange'

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

function fmtBeijingDate(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return date.toLocaleString('zh-CN', {
    timeZone: 'Asia/Shanghai',
    hour12: false,
  })
}

function fmtCredits(credits) {
  if (!credits) return '-'
  if (credits.unlimited) return '无限'
  if (credits.balance == null || credits.balance === '') return '0'
  return String(credits.balance)
}

function fmtCompact(value) {
  const number = Number(value)
  if (!Number.isFinite(number)) return '-'
  return new Intl.NumberFormat('zh-CN', {
    maximumFractionDigits: 1,
    notation: 'compact',
  }).format(number)
}

function fmtDuration(seconds) {
  const number = Number(seconds)
  if (!Number.isFinite(number) || number < 0) return '-'
  if (number < 60) return '不到 1 分钟'
  const totalMinutes = Math.max(1, Math.round(number / 60))
  const days = Math.floor(totalMinutes / (24 * 60))
  const hours = Math.floor((totalMinutes % (24 * 60)) / 60)
  const minutes = totalMinutes % 60
  const parts = []
  if (days > 0) parts.push(`${days} 天`)
  if (hours > 0) parts.push(`${hours} 小时`)
  if (days === 0 && minutes > 0) parts.push(`${minutes} 分钟`)
  return parts.join(' ') || '不到 1 分钟'
}

function fmtResetCountdown(item, nowValue) {
  const resetAt = item?.resets_at_time
    ? new Date(item.resets_at_time).getTime()
    : Number(item?.resets_at) * 1000
  const nowAt = new Date(nowValue || Date.now()).getTime()
  if (!Number.isFinite(resetAt) || !Number.isFinite(nowAt)) return '-'
  if (resetAt <= nowAt) return '等待额度刷新'
  return `${fmtDuration((resetAt - nowAt) / 1000)}后重置`
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

function sortResetCredits(payload) {
  const credits = payload?.rate_limit_reset_credits?.credits || []
  return [...credits].sort((a, b) =>
    String(a?.granted_at || '').localeCompare(String(b?.granted_at || ''))
  )
}

function resetCreditsSummary(payload) {
  const resetCredits = payload?.rate_limit_reset_credits
  if (!resetCredits) return '-'
  if (resetCredits.status !== 'ok') return '暂不可用'
  return `${resetCredits.available_count || 0} / ${
    resetCredits.total_earned_count || 0
  }`
}

function resetCreditStatusText(value) {
  if (value === 'available') return '可用'
  if (value === 'redeemed') return '已使用'
  if (value === 'expired') return '已过期'
  return value || '-'
}

function resetCreditTitle(item) {
  return item?.title || item?.reset_type || 'Rate limit reset credit'
}

function paceCopy(pace) {
  if (!pace) return '窗口时间进度达到 3% 后显示线性节奏估算'
  const delta = Math.round(Math.abs(pace.deltaPercent))
  let left = '用量正常'
  if (pace.deltaPercent > 2) {
    left = `比可持续节奏快 ${delta}%`
  } else if (pace.deltaPercent < -2) {
    left = `尚有 ${delta}% 节奏余量`
  }

  if (pace.willLastToReset) return `${left} · 可持续到重置`
  if (pace.etaSeconds === 0) return `${left} · 当前窗口已用尽`
  if (pace.etaSeconds != null) {
    return `${left} · 预计 ${fmtDuration(pace.etaSeconds)}后用尽`
  }
  return left
}

function LimitBar({ item, label, sampledAt }) {
  if (!item) return null
  const remaining = clampPercent(item?.remaining_percent)
  const used = clampPercent(item?.used_percent)
  const pace = calculateRateLimitPace(item, sampledAt || Date.now())

  return (
    <div className="grid gap-2" data-codex-limit-window={label}>
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <span className="text-sm font-semibold text-[var(--admin-text)]">
          {label} {fmtPercent(used)} 已用
        </span>
        <span className="text-sm text-[var(--admin-muted)]">
          {fmtResetCountdown(item, sampledAt)}
        </span>
      </div>
      <div className="relative h-3 overflow-hidden rounded-full bg-[var(--admin-surface-soft)]">
        <div
          className="h-full rounded-full bg-[#2f9e5b] transition-[width]"
          style={{ width: `${used}%` }}
        />
        {pace ? (
          <span
            aria-label={`可持续节奏标记 ${fmtPercent(
              pace.expectedUsedPercent
            )}`}
            className="absolute inset-y-0 w-0.5 bg-[var(--admin-text)] opacity-60"
            style={{ left: `${pace.expectedUsedPercent}%` }}
            title={`按窗口时间进度，当前可持续用量约 ${fmtPercent(
              pace.expectedUsedPercent
            )}`}
          />
        ) : null}
      </div>
      <div className="flex flex-wrap justify-between gap-2 text-xs text-[var(--admin-muted)]">
        <span>{paceCopy(pace)}</span>
        <span>{fmtPercent(remaining)} 剩余</span>
      </div>
    </div>
  )
}

function LimitCard({ item, sampledAt }) {
  return (
    <SurfacePanel variant="admin" className="p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-[var(--admin-text)]">
            {rateLimitTitle(item)}
          </h2>
          <p className="mt-1 text-sm text-[var(--admin-muted)]">
            {item?.limit_id || '-'} · {item?.plan_type || '未记录套餐'}
          </p>
        </div>
        <div className="rounded-md border border-[var(--admin-border)] bg-[var(--admin-surface-muted)] px-3 py-2 text-right">
          <div className="text-xs text-[var(--admin-muted)]">Credits</div>
          <div className="mt-0.5 text-lg font-bold text-[var(--admin-text)]">
            {fmtCredits(item?.credits)}
          </div>
        </div>
      </div>

      <div className="mt-5 grid gap-5 lg:grid-cols-2">
        <LimitBar
          label="5 小时额度"
          item={item?.primary}
          sampledAt={sampledAt}
        />
        <LimitBar
          label="每周额度"
          item={item?.secondary}
          sampledAt={sampledAt}
        />
      </div>
    </SurfacePanel>
  )
}

function ResetCreditsPanel({ payload, credits }) {
  const resetCredits = payload?.rate_limit_reset_credits
  const unavailable = resetCredits?.status && resetCredits.status !== 'ok'

  return (
    <SurfacePanel variant="admin" className="p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-[var(--admin-text)]">
            Rate limit reset credits
          </h2>
          <p className="mt-1 text-sm text-[var(--admin-muted)]">
            可用 {resetCredits?.available_count || 0} 个 / 累计{' '}
            {resetCredits?.total_earned_count || 0} 个
          </p>
        </div>
        <div className="rounded-md border border-[var(--admin-border)] bg-[var(--admin-surface-muted)] px-3 py-2 text-right">
          <div className="text-xs text-[var(--admin-muted)]">状态</div>
          <div className="mt-0.5 text-base font-bold text-[var(--admin-text)]">
            {unavailable ? '暂不可用' : '正常'}
          </div>
        </div>
      </div>

      {unavailable ? (
        <div className="mt-4 rounded-lg border border-[var(--admin-warning-border)] bg-[var(--admin-warning-bg)] px-4 py-3 text-sm text-[var(--admin-warning-text)]">
          重置券读取暂时不可用，当前余额和限额窗口仍可正常查看。
        </div>
      ) : null}

      {credits.length > 0 ? (
        <div className="mt-4 overflow-x-auto">
          <table className="admin-data-table min-w-[760px] text-left text-sm">
            <thead>
              <tr className="bg-[var(--admin-surface-soft)] text-[var(--admin-muted-strong)]">
                <th className="w-16 px-4 py-3 font-semibold">#</th>
                <th className="px-4 py-3 font-semibold">类型</th>
                <th className="w-24 px-4 py-3 font-semibold">状态</th>
                <th className="w-56 px-4 py-3 font-semibold">
                  获得时间（北京时间）
                </th>
                <th className="w-56 px-4 py-3 font-semibold">
                  过期时间（北京时间）
                </th>
              </tr>
            </thead>
            <tbody>
              {credits.map((item, index) => (
                <tr
                  key={`${item?.granted_at || index}-${item?.expires_at || ''}`}
                  className="border-t border-[var(--admin-border-soft)] text-[var(--admin-text)]"
                >
                  <td className="px-4 py-3 font-semibold">{index + 1}</td>
                  <td className="px-4 py-3 font-medium">
                    {resetCreditTitle(item)}
                  </td>
                  <td className="px-4 py-3">
                    {resetCreditStatusText(item?.status)}
                  </td>
                  <td className="px-4 py-3">
                    {fmtBeijingDate(item?.granted_at)}
                  </td>
                  <td className="px-4 py-3">
                    {fmtBeijingDate(item?.expires_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : !unavailable ? (
        <div className="mt-4 text-sm text-[var(--admin-muted)]">
          当前没有可展示的重置券。
        </div>
      ) : null}
    </SurfacePanel>
  )
}

function UsageBars({ daily }) {
  if (!daily?.length) {
    return (
      <div className="mt-5 rounded-lg border border-dashed border-[var(--admin-border)] px-4 py-8 text-center text-sm text-[var(--admin-muted)]">
        近 30 天暂无本服务调用记录。
      </div>
    )
  }

  const useCost = daily.some((item) => item.costUSD != null)
  const values = daily.map((item) =>
    useCost ? Number(item.costUSD || 0) : Number(item.totalTokens || 0)
  )
  const maximum = Math.max(...values, 0)
  const firstDay = daily[0]?.date || '-'
  const lastDay = daily.at(-1)?.date || '-'

  return (
    <div className="mt-5" data-codex-usage-chart>
      <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-[var(--admin-muted)]">
        <span>{useCost ? '每日费用估算' : '每日 Token'}</span>
        <span>
          峰值 {useCost ? formatCodexUsageCost(maximum) : fmtCompact(maximum)}
        </span>
      </div>
      <div
        aria-label={`近 30 天${useCost ? '费用估算' : 'Token'}柱状图`}
        className="mt-3 grid h-32 items-end gap-1 rounded-lg border border-[var(--admin-border-soft)] bg-[var(--admin-surface-muted)] px-3 pb-2 pt-3"
        role="img"
        style={{
          gridTemplateColumns: `repeat(${daily.length}, minmax(5px, 1fr))`,
        }}
      >
        {daily.map((item, index) => {
          const value = values[index]
          const unknownCost = useCost && item.costUSD == null
          const height =
            maximum > 0 && !unknownCost
              ? Math.max(3, Math.round((value / maximum) * 100))
              : 3
          const label = `${item.date}：${
            useCost
              ? formatCodexUsageCost(item.costUSD)
              : `${fmtCompact(item.totalTokens)} Token`
          } · ${fmtCompact(item.totalRequests)} 次请求`
          return (
            <span
              key={item.date}
              aria-label={label}
              className={`min-w-0 rounded-sm ${
                unknownCost
                  ? 'border border-dashed border-[var(--admin-muted)] bg-transparent'
                  : 'bg-[#3aa0ad]'
              }`}
              style={{ height: `${height}%` }}
              title={label}
            />
          )
        })}
      </div>
      <div className="mt-2 flex justify-between text-xs text-[var(--admin-muted)]">
        <span>{firstDay}</span>
        <span>{lastDay}</span>
      </div>
    </div>
  )
}

function UsageEstimatePanel({ error, loading, overview }) {
  return (
    <SurfacePanel variant="admin" className="p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h2 className="text-base font-semibold text-[var(--admin-text)]">
            本服务调用估算
          </h2>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-[var(--admin-muted)]">
            参考 CodexBar 的 Today /
            30d、Token、主模型和日柱图统计方式；这里只统计经本服务转发并落库的
            usage，不等于 Codex 订阅账单或账户全部消耗。
          </p>
        </div>
        <a className="admin-button" href="/admin-usage">
          查看用量明细
        </a>
      </div>

      {error ? (
        <div className="mt-4 rounded-lg border border-[var(--admin-warning-border)] bg-[var(--admin-warning-bg)] px-4 py-3 text-sm text-[var(--admin-warning-text)]">
          {error}；额度真值仍可单独查看。
        </div>
      ) : null}

      {loading && !overview ? (
        <div className="mt-5 text-sm text-[var(--admin-muted)]">
          正在汇总本服务近 30 天调用...
        </div>
      ) : null}

      {overview ? (
        <>
          <div className="mt-5 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <div className="rounded-lg border border-[var(--admin-border-soft)] bg-[var(--admin-surface-muted)] p-4">
              <div className="text-sm text-[var(--admin-muted)]">
                今日费用估算
              </div>
              <div className="mt-1 text-2xl font-bold text-[var(--admin-text)]">
                {formatCodexUsageCost(overview.todayCostUSD)}
              </div>
            </div>
            <div className="rounded-lg border border-[var(--admin-border-soft)] bg-[var(--admin-surface-muted)] p-4">
              <div className="text-sm text-[var(--admin-muted)]">
                近 30 天费用估算
              </div>
              <div className="mt-1 text-2xl font-bold text-[var(--admin-text)]">
                {formatCodexUsageCost(overview.periodCostUSD)}
              </div>
            </div>
            <div className="rounded-lg border border-[var(--admin-border-soft)] bg-[var(--admin-surface-muted)] p-4">
              <div className="text-sm text-[var(--admin-muted)]">
                最近有记录日 Token
              </div>
              <div className="mt-1 text-2xl font-bold text-[var(--admin-text)]">
                {overview.latestTokens == null
                  ? '-'
                  : fmtCompact(overview.latestTokens)}
              </div>
              <div className="mt-1 text-xs text-[var(--admin-muted)]">
                {overview.latestDay || '暂无日期'}
              </div>
            </div>
            <div className="rounded-lg border border-[var(--admin-border-soft)] bg-[var(--admin-surface-muted)] p-4">
              <div className="text-sm text-[var(--admin-muted)]">
                近 30 天 Token
              </div>
              <div className="mt-1 text-2xl font-bold text-[var(--admin-text)]">
                {fmtCompact(overview.periodTokens)}
              </div>
            </div>
          </div>

          <UsageBars daily={overview.daily} />

          <div className="mt-4 flex flex-wrap gap-x-6 gap-y-2 text-sm text-[var(--admin-muted)]">
            <span>
              主模型：
              <strong className="text-[var(--admin-text)]">
                {overview.topModelName || '-'}
              </strong>
              {overview.topModelName
                ? `（按${
                    overview.topModelBasis === 'cost' ? '费用估算' : 'Token'
                  }）`
                : ''}
            </span>
            <span>近 30 天请求：{fmtCompact(overview.periodRequests)}</span>
            <span>今日 Token：{fmtCompact(overview.todayTokens)}</span>
          </div>
          <p className="mt-3 text-xs leading-5 text-[var(--admin-muted)]">
            费用按 usage Token
            与本服务模型价格估算，不是订阅账单；长上下文附加计费等未建模项目不在此口径。账号邮箱与续费日不展示，因为当前额度接口不提供这些字段。
          </p>
        </>
      ) : null}
    </SurfacePanel>
  )
}

export default function AdminCodexBalancePage() {
  const apiRpc = useMemo(
    () =>
      new JsonRpc({
        url: 'api',
        basePath: ADMIN_BASE_PATH,
        authScope: AUTH_SCOPE.ADMIN,
      }),
    []
  )
  const [payload, setPayload] = useState(null)
  const [usageData, setUsageData] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [usageError, setUsageError] = useState('')

  const limits = useMemo(() => sortRateLimits(payload), [payload])
  const resetCredits = useMemo(() => sortResetCredits(payload), [payload])
  const usageOverview = useMemo(
    () =>
      usageData
        ? buildCodexUsageOverview({
            buckets: usageData.buckets,
            endTime: usageData.endTime,
            periodSummary: usageData.periodSummary,
            startTime: usageData.startTime,
            todaySummary: usageData.todaySummary,
          })
        : null,
    [usageData]
  )

  const loadData = useCallback(
    async ({ signal } = {}) => {
      setLoading(true)
      setError('')
      setUsageError('')

      const balanceRequest = async () => {
        const response = await fetch(BALANCE_ENDPOINT, {
          method: 'GET',
          headers: { Accept: 'application/json' },
          signal,
        })
        const data = await response.json().catch(() => null)
        if (!response.ok || data?.status !== 'ok') {
          throw new Error('codex_balance_query_failed')
        }
        return data
      }

      const usageRequest = async () => {
        const now = Math.floor(Date.now() / 1000)
        const todayStart = startOfLocalDayUnix(new Date(now * 1000))
        const period = getUsageTimeWindow(
          DEFAULT_DAILY_USAGE_TIME_RANGE,
          now,
          DEFAULT_DAILY_USAGE_TIME_RANGE
        )
        const [todayRes, periodRes, bucketsRes] = await Promise.all([
          apiRpc.call('summary', {
            end_time: now,
            start_time: todayStart,
          }),
          apiRpc.call('summary', {
            end_time: period.endTime,
            start_time: period.startTime,
          }),
          apiRpc.call('usage_buckets', {
            end_time: period.endTime,
            group_by: 'day_model',
            start_time: period.startTime,
          }),
        ])
        return {
          buckets: Array.isArray(bucketsRes?.data?.items)
            ? bucketsRes.data.items
            : [],
          endTime: period.endTime,
          periodSummary: periodRes?.data?.summary || {},
          startTime: period.startTime,
          todaySummary: todayRes?.data?.summary || {},
        }
      }

      try {
        const [balanceResult, usageResult] = await Promise.allSettled([
          balanceRequest(),
          usageRequest(),
        ])
        if (signal?.aborted) return

        if (balanceResult.status === 'fulfilled') {
          setPayload(balanceResult.value)
        } else {
          setPayload(null)
          setError('Codex 余额查询失败，请稍后重试')
        }

        if (usageResult.status === 'fulfilled') {
          setUsageData(usageResult.value)
        } else {
          setUsageError(
            getActionErrorMessage(usageResult.reason, '加载调用估算')
          )
        }
      } finally {
        if (!signal?.aborted) setLoading(false)
      }
    },
    [apiRpc]
  )

  useEffect(() => {
    const controller = new AbortController()
    loadData({ signal: controller.signal })
    return () => controller.abort()
  }, [loadData])

  return (
    <AdminFrame
      breadcrumb="用量统计 / Codex 余额"
      title="Codex 余额"
      description="上游账户额度与重置时间保持实时真值；节奏为当前窗口线性估算，Today / 30d 费用、Token、主模型和日柱图来自本服务 usage 日志。"
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
            onClick={() => loadData()}
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
        <div className="grid gap-4 md:grid-cols-4">
          <div>
            <div className="text-sm text-[var(--admin-muted)]">接口状态</div>
            <div className="mt-1 text-2xl font-bold text-[var(--admin-text)]">
              {balanceStatusText(payload, loading)}
            </div>
          </div>
          <div>
            <div className="text-sm text-[var(--admin-muted)]">
              Credits remaining
            </div>
            <div className="mt-1 text-2xl font-bold text-[var(--admin-text)]">
              {fmtCredits(payload?.credits)}
            </div>
          </div>
          <div>
            <div className="text-sm text-[var(--admin-muted)]">可用重置券</div>
            <div className="mt-1 text-2xl font-bold text-[var(--admin-text)]">
              {resetCreditsSummary(payload)}
            </div>
          </div>
          <div>
            <div className="text-sm text-[var(--admin-muted)]">更新时间</div>
            <div className="mt-2 break-words text-sm font-semibold text-[var(--admin-text)]">
              {fmtDate(payload?.fetched_at)}
            </div>
          </div>
        </div>
      </SurfacePanel>

      {loading && !payload ? (
        <SurfacePanel variant="admin" className="p-5">
          <div className="text-sm text-[var(--admin-muted)]">
            正在读取 Codex 余额...
          </div>
        </SurfacePanel>
      ) : null}

      {limits.length > 0 ? (
        <div className="grid gap-5 xl:grid-cols-2">
          {limits.map((item) => (
            <LimitCard
              key={item.limit_id || rateLimitTitle(item)}
              item={item}
              sampledAt={payload?.fetched_at}
            />
          ))}
        </div>
      ) : null}

      {payload ? (
        <ResetCreditsPanel payload={payload} credits={resetCredits} />
      ) : null}

      <UsageEstimatePanel
        error={usageError}
        loading={loading}
        overview={usageOverview}
      />
    </AdminFrame>
  )
}
