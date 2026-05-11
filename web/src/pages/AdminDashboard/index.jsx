import React, { useEffect, useMemo, useState } from 'react'
import AdminFrame from '@/common/components/layout/AdminFrame'
import SurfacePanel from '@/common/components/layout/SurfacePanel'
import { AUTH_SCOPE } from '@/common/auth/auth'
import { ADMIN_BASE_PATH } from '@/common/utils/adminRpc'
import { getActionErrorMessage } from '@/common/utils/errorMessage'
import { JsonRpc } from '@/common/utils/jsonRpc'

const PAGE_SIZE = 12
const DAY_SECONDS = 24 * 60 * 60
const TREND_DAYS = 30
const DASHBOARD_KEY_FETCH_LIMIT = 200
const TREND_METRICS = [
  { key: 'requests', label: '请求', field: 'total_requests', color: '#1478ff' },
  { key: 'errors', label: '错误', field: 'failed_requests', color: '#cf1322' },
  {
    key: 'cost',
    label: '费用',
    field: 'estimated_cost_usd',
    color: '#7c3aed',
  },
  {
    key: 'duration',
    label: '延迟',
    field: 'average_duration_ms',
    color: '#d97706',
  },
  { key: 'tokens', label: 'Token', field: 'total_tokens', color: '#238a43' },
]
const TREND_CHART_TYPES = [
  { key: 'bar', label: '柱状' },
  { key: 'line', label: '折线' },
]
const CODEX_UPSTREAM_MODE_OPTIONS = [
  { label: 'Backend 优先', value: 'codex_backend' },
  { label: '强制 CLI', value: 'codex_cli' },
]

const tableWrapClass = 'overflow-hidden rounded-lg border border-[#dde8df]'
const tableClass = 'min-w-full text-left text-sm text-[#1f2d25]'
const thClass =
  'whitespace-nowrap bg-[#f5fbf7] px-4 py-3 font-semibold text-[#66736b]'
const tdClass = 'px-4 py-4 text-[#1f2d25]'

function asInt(v, fallback = 0) {
  const n = Number(v)
  return Number.isFinite(n) ? Math.trunc(n) : fallback
}

function fmtNumber(v) {
  return new Intl.NumberFormat().format(asInt(v, 0))
}

function billableInputTokens(item) {
  const provided = asInt(item?.billable_input_tokens, -1)
  if (provided >= 0) return provided
  return Math.max(
    0,
    asInt(item?.input_tokens, 0) -
      asInt(item?.cached_input_tokens ?? item?.cached_tokens, 0)
  )
}

function fmtCompact(v) {
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 1,
    notation: 'compact',
  }).format(asInt(v, 0))
}

function fmtTs(ts) {
  if (!ts) return '-'
  const d = new Date(Number(ts) * 1000)
  if (Number.isNaN(d.getTime())) return String(ts)
  return d.toLocaleString()
}

function fmtShortDate(ts) {
  if (!ts) return '-'
  const d = new Date(Number(ts) * 1000)
  if (Number.isNaN(d.getTime())) return String(ts)
  return `${d.getMonth() + 1}/${d.getDate()}`
}

function fmtCost(v) {
  if (v == null || v === '') return '未配置价格'
  const n = Number(v)
  if (!Number.isFinite(n)) return '未配置价格'
  return `$${n.toFixed(4)}`
}

function fmtDuration(v) {
  return `${fmtNumber(v)} ms`
}

function fmtRate(part, total) {
  const safeTotal = asInt(total, 0)
  if (safeTotal <= 0) return '0%'
  const value = (asInt(part, 0) / safeTotal) * 100
  return `${value >= 10 ? value.toFixed(0) : value.toFixed(1)}%`
}

function upstreamModeLabel(value) {
  const item = CODEX_UPSTREAM_MODE_OPTIONS.find(
    (option) => option.value === value
  )
  return item?.label || '未记录'
}

function percentile(values, ratio) {
  const cleanValues = values
    .map((item) => Number(item))
    .filter((item) => Number.isFinite(item) && item >= 0)
    .sort((a, b) => a - b)
  if (cleanValues.length === 0) return 0
  const index = Math.min(
    cleanValues.length - 1,
    Math.max(0, Math.ceil(cleanValues.length * ratio) - 1)
  )
  return Math.round(cleanValues[index])
}

function localDateKey(ts) {
  const d = new Date(Number(ts) * 1000)
  if (Number.isNaN(d.getTime())) return ''
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function startOfLocalDayUnix(date) {
  return Math.floor(
    new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime() /
      1000
  )
}

function pct(part, total) {
  const safeTotal = Math.max(1, Number(total) || 0)
  return Math.min(
    100,
    Math.max(0, Math.round((Number(part) / safeTotal) * 100))
  )
}

function fillDailyBuckets(items, days) {
  const byDay = new Map(
    (Array.isArray(items) ? items : []).map((item) => [
      localDateKey(item.bucket_start),
      item,
    ])
  )
  const today = new Date()
  const todayStart = new Date(
    today.getFullYear(),
    today.getMonth(),
    today.getDate()
  )

  return Array.from({ length: days }, (_, index) => {
    const d = new Date(todayStart)
    d.setDate(todayStart.getDate() - (days - 1 - index))
    const bucketStart = startOfLocalDayUnix(d)
    const source = byDay.get(localDateKey(bucketStart))
    const hasSource = Boolean(source)
    return {
      average_duration_ms: asInt(source?.average_duration_ms, 0),
      billable_input_tokens: billableInputTokens(source),
      bucket_start: asInt(source?.bucket_start, bucketStart),
      cached_tokens: asInt(source?.cached_tokens, 0),
      backend_requests: asInt(source?.backend_requests, 0),
      cli_requests: asInt(source?.cli_requests, 0),
      failed_requests: asInt(source?.failed_requests, 0),
      fallback_requests: asInt(source?.fallback_requests, 0),
      input_tokens: asInt(source?.input_tokens, 0),
      output_tokens: asInt(source?.output_tokens, 0),
      reasoning_tokens: asInt(source?.reasoning_tokens, 0),
      success_requests: asInt(source?.success_requests, 0),
      total_requests: asInt(source?.total_requests, 0),
      total_tokens: asInt(source?.total_tokens, 0),
      estimated_cost_usd: !hasSource
        ? 0
        : source.estimated_cost_usd == null
          ? null
          : Number(source.estimated_cost_usd),
    }
  })
}

function getTrendValue(item, metric) {
  const rawValue = item?.[metric.field]
  if (metric.key === 'cost') {
    return rawValue == null ? 0 : Number(rawValue)
  }
  return asInt(rawValue, 0)
}

function sumBuckets(buckets) {
  const out = buckets.reduce(
    (acc, item) => {
      const requests = asInt(item.total_requests, 0)
      return {
        average_duration_ms: 0,
        billable_input_tokens:
          acc.billable_input_tokens + billableInputTokens(item),
        cached_tokens: acc.cached_tokens + asInt(item.cached_tokens, 0),
        duration_weighted_ms:
          acc.duration_weighted_ms +
          asInt(item.average_duration_ms, 0) * requests,
        failed_requests: acc.failed_requests + asInt(item.failed_requests, 0),
        backend_requests:
          acc.backend_requests + asInt(item.backend_requests, 0),
        cli_requests: acc.cli_requests + asInt(item.cli_requests, 0),
        fallback_requests:
          acc.fallback_requests + asInt(item.fallback_requests, 0),
        input_tokens: acc.input_tokens + asInt(item.input_tokens, 0),
        output_tokens: acc.output_tokens + asInt(item.output_tokens, 0),
        reasoning_tokens:
          acc.reasoning_tokens + asInt(item.reasoning_tokens, 0),
        success_requests:
          acc.success_requests + asInt(item.success_requests, 0),
        total_requests: acc.total_requests + requests,
        total_tokens: acc.total_tokens + asInt(item.total_tokens, 0),
      }
    },
    {
      average_duration_ms: 0,
      billable_input_tokens: 0,
      backend_requests: 0,
      cached_tokens: 0,
      cli_requests: 0,
      duration_weighted_ms: 0,
      fallback_requests: 0,
      failed_requests: 0,
      input_tokens: 0,
      output_tokens: 0,
      reasoning_tokens: 0,
      success_requests: 0,
      total_requests: 0,
      total_tokens: 0,
    }
  )
  return {
    ...out,
    average_duration_ms:
      out.total_requests > 0
        ? Math.round(out.duration_weighted_ms / out.total_requests)
        : 0,
  }
}

function topUsageGroups(items, pickValue) {
  const groups = new Map()
  for (const item of items) {
    const label = String(pickValue(item) || '未标记')
    const current = groups.get(label) || { calls: 0, label, tokens: 0 }
    current.calls += 1
    current.tokens += asInt(item.total_tokens, 0)
    groups.set(label, current)
  }
  return Array.from(groups.values())
    .sort((a, b) => b.tokens - a.tokens || b.calls - a.calls)
    .slice(0, 5)
}

function fmtTrendValue(metricKey, value) {
  if (metricKey === 'cost') return fmtCost(value)
  if (metricKey === 'duration') return fmtDuration(value)
  return fmtNumber(value)
}

function getTrendTooltipRows(item, metricConfig) {
  if (!item) return []
  if (metricConfig.key === 'requests') {
    return [
      ['总请求', fmtNumber(item.total_requests)],
      ['成功', fmtNumber(item.success_requests)],
      ['失败', fmtNumber(item.failed_requests)],
      [
        'Backend / CLI',
        `${fmtNumber(item.backend_requests)} / ${fmtNumber(item.cli_requests)}`,
      ],
      ['Fallback', fmtNumber(item.fallback_requests)],
    ]
  }
  if (metricConfig.key === 'errors') {
    return [
      ['失败请求', fmtNumber(item.failed_requests)],
      ['错误率', fmtRate(item.failed_requests, item.total_requests)],
      ['总请求', fmtNumber(item.total_requests)],
    ]
  }
  if (metricConfig.key === 'cost') {
    return [
      ['费用估算', fmtCost(item.estimated_cost_usd)],
      ['总请求', fmtNumber(item.total_requests)],
      ['总 Token', fmtNumber(item.total_tokens)],
    ]
  }
  if (metricConfig.key === 'duration') {
    return [
      ['平均耗时', fmtDuration(item.average_duration_ms)],
      ['总请求', fmtNumber(item.total_requests)],
      ['失败请求', fmtNumber(item.failed_requests)],
    ]
  }
  return [
    ['总 Token', fmtNumber(item.total_tokens)],
    ['输入', fmtNumber(item.input_tokens)],
    ['非缓存输入', fmtNumber(billableInputTokens(item))],
    ['缓存输入', fmtNumber(item.cached_tokens)],
    ['输出', fmtNumber(item.output_tokens)],
    ['Reasoning', fmtNumber(item.reasoning_tokens)],
  ]
}

function SummaryCard({ label, value, sub, tone = 'green' }) {
  const toneClass =
    tone === 'red'
      ? 'border-rose-200 bg-rose-50'
      : tone === 'blue'
        ? 'border-[#d6e7ff] bg-[#f2f7ff]'
        : tone === 'amber'
          ? 'border-amber-200 bg-amber-50'
          : 'border-[#dce8df] bg-[#f7faf8]'
  return (
    <SurfacePanel variant="admin" className="p-5">
      <div className={`mb-4 h-1.5 w-14 rounded-full border ${toneClass}`} />
      <div className="text-sm font-semibold text-[#7b8780]">{label}</div>
      <div className="mt-3 truncate text-2xl font-semibold text-[#1f2d25]">
        {value}
      </div>
      {sub ? <div className="mt-2 text-xs text-[#9aa39e]">{sub}</div> : null}
    </SurfacePanel>
  )
}

function StatusBadge({ active, text }) {
  return (
    <span
      className={`inline-flex whitespace-nowrap rounded-full px-3 py-1 text-xs font-semibold ${
        active ? 'bg-emerald-50 text-emerald-700' : 'bg-rose-50 text-rose-700'
      }`}
    >
      {text}
    </span>
  )
}

function HeaderWithHelp({ children, help }) {
  if (!help) return children
  return (
    <span className="admin-th-help-wrap">
      <span>{children}</span>
      <button
        type="button"
        className="admin-th-help"
        aria-label={`${children}说明：${help}`}
        data-tooltip={help}
      >
        ?
      </button>
    </span>
  )
}

function apiKeyRemark(item) {
  return item?.api_key_name || item?.name || '无备注'
}

function ApiKeyUsageCell({ item }) {
  return (
    <div className="min-w-[160px]">
      <div className="max-w-[240px] truncate font-medium text-[#1f2d25]">
        {apiKeyRemark(item)}
      </div>
      <div className="mt-1 font-mono text-xs text-[#7b8780]">
        {item?.api_key_prefix || item?.prefix || '-'}
      </div>
    </div>
  )
}

function UsageTrendChart({ buckets, chartType, metric }) {
  const [activeIndex, setActiveIndex] = useState(null)
  const metricConfig =
    TREND_METRICS.find((item) => item.key === metric) || TREND_METRICS[0]
  const values = buckets.map((item) => getTrendValue(item, metricConfig))
  const maxValue = Math.max(1, ...values)
  const hasData = values.some((value) => value > 0)
  const isLineChart = chartType === 'line'
  const linePoints = values.map((value, index) => ({
    x: buckets.length <= 1 ? 50 : (index / (buckets.length - 1)) * 100,
    y: 96 - (value / maxValue) * 88,
  }))
  const linePointText = linePoints
    .map((point) => `${point.x.toFixed(2)},${point.y.toFixed(2)}`)
    .join(' ')
  const activeBucket =
    activeIndex == null
      ? null
      : buckets[Math.min(activeIndex, buckets.length - 1)]
  const activeValue =
    activeIndex == null ? 0 : values[Math.min(activeIndex, values.length - 1)]
  const tooltipRows = getTrendTooltipRows(activeBucket, metricConfig)

  return (
    <div className="min-w-0">
      <div
        className="relative grid h-64 items-end gap-1 rounded-lg bg-[#f7faf8] px-3 pb-8 pt-4"
        data-trend-chart=""
        onMouseLeave={() => setActiveIndex(null)}
        style={{
          gridTemplateColumns: `repeat(${Math.max(1, buckets.length)}, minmax(6px, 1fr))`,
        }}
      >
        {isLineChart && linePoints.length > 0 ? (
          <div
            className="pointer-events-none absolute inset-x-3 bottom-8 top-4 z-0 overflow-hidden"
            data-trend-line-box=""
          >
            <svg
              aria-hidden="true"
              className="h-full w-full"
              data-trend-line=""
              preserveAspectRatio="none"
              viewBox="0 0 100 100"
            >
              <polyline
                fill="none"
                points={linePointText}
                stroke={metricConfig.color}
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth="3"
                vectorEffect="non-scaling-stroke"
              />
            </svg>
          </div>
        ) : null}
        {activeBucket ? (
          <div
            className="pointer-events-none absolute left-3 top-3 z-10 w-[min(18rem,calc(100%-1.5rem))] rounded-lg border border-[#d6ded8] bg-white/95 p-3 text-xs text-[#1f2d25] shadow-[0_12px_32px_rgba(20,52,35,0.16)] backdrop-blur"
            data-trend-tooltip=""
          >
            <div className="mb-2 flex items-center justify-between gap-3">
              <span className="font-semibold">
                {fmtShortDate(activeBucket.bucket_start)}
              </span>
              <span className="font-semibold text-[#1478ff]">
                {metricConfig.label}{' '}
                {fmtTrendValue(metricConfig.key, activeValue)}
              </span>
            </div>
            <div className="space-y-1">
              {tooltipRows.map(([label, value]) => (
                <div
                  key={label}
                  className="flex items-center justify-between gap-4"
                >
                  <span className="text-[#66736b]">{label}</span>
                  <span className="font-medium">{value}</span>
                </div>
              ))}
            </div>
          </div>
        ) : null}
        {buckets.map((item, index) => {
          const metricValue = values[index]
          const metricHeight = Math.max(
            metricValue > 0 ? 4 : 0,
            Math.round((metricValue / maxValue) * 100)
          )
          return (
            <button
              key={item.bucket_start}
              type="button"
              aria-label={`${fmtShortDate(item.bucket_start)} ${metricConfig.label} ${fmtTrendValue(metricConfig.key, metricValue)}`}
              className="group relative z-[1] flex h-full min-w-0 items-end justify-center rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-[#1478ff] focus-visible:ring-offset-2 focus-visible:ring-offset-[#f7faf8]"
              data-trend-bar=""
              onBlur={() => setActiveIndex(null)}
              onFocus={() => setActiveIndex(index)}
              onMouseEnter={() => setActiveIndex(index)}
              title={`${fmtShortDate(item.bucket_start)} ${metricConfig.label} ${fmtTrendValue(metricConfig.key, metricValue)}`}
            >
              <div className="relative flex h-full w-full max-w-6 items-end">
                {isLineChart ? (
                  <span
                    className="absolute left-1/2 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white shadow-[0_0_0_1px_rgba(20,52,35,0.18)] transition-transform group-hover:scale-125 group-focus-visible:scale-125"
                    data-trend-point=""
                    style={{
                      backgroundColor: metricConfig.color,
                      top: `${linePoints[index]?.y ?? 96}%`,
                    }}
                  />
                ) : (
                  <div
                    className="w-full rounded-t transition-[filter,opacity] group-hover:brightness-110 group-focus-visible:brightness-110"
                    style={{
                      backgroundColor: metricConfig.color,
                      height: `${metricHeight}%`,
                      opacity: metricValue > 0 ? 1 : 0.28,
                    }}
                  />
                )}
              </div>
            </button>
          )
        })}
      </div>
      <div className="mt-3 flex flex-wrap items-center justify-between gap-3 text-xs text-[#7b8780]">
        <span>{fmtShortDate(buckets[0]?.bucket_start)}</span>
        <span>
          {hasData
            ? `按${metricConfig.label}展示`
            : `暂无${metricConfig.label}记录`}
        </span>
        <span>{fmtShortDate(buckets[buckets.length - 1]?.bucket_start)}</span>
      </div>
    </div>
  )
}

function TokenComposition({ stats }) {
  const total = Math.max(1, asInt(stats.total_tokens, 0))
  const rows = [
    ['输入 Token', stats.input_tokens, 'bg-[#1478ff]'],
    ['非缓存输入', billableInputTokens(stats), 'bg-[#54a3ff]'],
    ['缓存输入', stats.cached_tokens, 'bg-[#78c596]'],
    ['输出 Token', stats.output_tokens, 'bg-[#d6a23a]'],
    ['Reasoning 输出', stats.reasoning_tokens, 'bg-[#9d7bd9]'],
  ]

  return (
    <div className="space-y-4">
      {rows.map(([label, value, color]) => (
        <div key={label}>
          <div className="mb-1 flex items-center justify-between gap-3 text-sm">
            <span className="text-[#66736b]">{label}</span>
            <span className="font-medium text-[#1f2d25]">
              {fmtNumber(value)}
            </span>
          </div>
          <div className="h-2 rounded-full bg-[#e5ece8]">
            <div
              className={`h-2 rounded-full ${color}`}
              style={{ width: `${pct(value, total)}%` }}
            />
          </div>
        </div>
      ))}
    </div>
  )
}

function UsageGroupList({ items, totalTokens }) {
  if (items.length === 0) {
    return <div className="text-sm text-[#9aa39e]">暂无最近调用样本</div>
  }

  return (
    <div className="space-y-3">
      {items.map((item) => (
        <div key={item.label}>
          <div className="mb-1 flex items-center justify-between gap-3 text-sm">
            <span className="min-w-0 truncate font-medium text-[#1f2d25]">
              {item.label}
            </span>
            <span className="shrink-0 text-xs text-[#7b8780]">
              {fmtNumber(item.calls)} 次 / {fmtCompact(item.tokens)}
            </span>
          </div>
          <div className="h-2 rounded-full bg-[#e5ece8]">
            <div
              className="h-2 rounded-full bg-[#238a43]"
              style={{ width: `${pct(item.tokens, totalTokens)}%` }}
            />
          </div>
        </div>
      ))}
    </div>
  )
}

function RecentCallsTable({ loading, usageItems, usageTotal }) {
  return (
    <SurfacePanel variant="admin" className="p-5 sm:p-6">
      <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-[#1f2d25]">最近调用</h2>
          <div className="mt-1 text-sm text-[#7b8780]">
            最近 24 小时 {usageItems.length} 条样本 / 共 {fmtNumber(usageTotal)}{' '}
            条。
          </div>
        </div>
        <a
          href="/admin-usage"
          className="inline-flex h-9 items-center justify-center rounded-md border border-[#d6ded8] bg-white px-3 text-sm font-semibold text-[#1f2d25] transition hover:border-[#9dc7aa] hover:bg-[#f1faf4]"
        >
          查看明细
        </a>
      </div>

      <div className={tableWrapClass}>
        <div className="overflow-auto">
          <table className={`${tableClass} min-w-[1880px]`}>
            <thead>
              <tr>
                <th className={thClass}>时间</th>
                <th className={thClass}>请求</th>
                <th className={thClass}>凭据</th>
                <th className={thClass}>接口</th>
                <th className={thClass}>模型</th>
                <th className={thClass}>上游</th>
                <th className={thClass}>状态</th>
                <th className={thClass}>
                  <HeaderWithHelp help="总 Token = 输入 Token + 输出 Token；非缓存输入 = 输入 Token - 缓存输入。">
                    Token
                  </HeaderWithHelp>
                </th>
                <th className={thClass}>
                  <HeaderWithHelp help="缓存输入是命中上下文缓存的输入 Token；推理输出是模型内部 reasoning 输出 Token。">
                    缓存输入 / 推理输出
                  </HeaderWithHelp>
                </th>
                <th className={thClass}>
                  <HeaderWithHelp help="按当前模型价格口径估算；未配置价格时显示未配置。">
                    费用估算
                  </HeaderWithHelp>
                </th>
                <th className={thClass}>
                  <HeaderWithHelp help="网关从收到请求到返回响应的耗时，单位毫秒。">
                    耗时
                  </HeaderWithHelp>
                </th>
                <th className={thClass}>
                  <HeaderWithHelp help="请求字节 / 响应字节，用于判断单次调用的数据体大小。">
                    字节
                  </HeaderWithHelp>
                </th>
                <th className={thClass}>错误</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#e7efe9] bg-white">
              {usageItems.length > 0 ? (
                usageItems.map((item) => (
                  <tr key={String(item.id)} className="align-top">
                    <td className={tdClass}>{fmtTs(item.created_at)}</td>
                    <td className={`${tdClass} min-w-[220px]`}>
                      <div className="font-mono text-xs">
                        {item.request_id || '-'}
                      </div>
                      <div className="mt-1 break-all text-xs text-[#9aa39e]">
                        Session：{item.session_id || '未传入'}
                      </div>
                    </td>
                    <td className={tdClass}>
                      <ApiKeyUsageCell item={item} />
                    </td>
                    <td className={tdClass}>
                      {item.endpoint || item.path}
                      <div className="mt-1 text-xs text-[#9aa39e]">
                        {item.method || '-'}
                      </div>
                    </td>
                    <td className={`${tdClass} font-mono text-xs`}>
                      {item.model || '-'}
                    </td>
                    <td className={tdClass}>
                      <div className="whitespace-nowrap text-xs font-semibold">
                        {upstreamModeLabel(item.upstream_mode)}
                      </div>
                      <div className="mt-1 text-xs text-[#9aa39e]">
                        {item.upstream_fallback ? 'fallback' : 'direct'}
                      </div>
                    </td>
                    <td className={tdClass}>
                      <StatusBadge
                        active={!!item.success}
                        text={`HTTP ${item.status_code || '-'}`}
                      />
                    </td>
                    <td className={tdClass}>
                      <div className="font-semibold">
                        总 {fmtNumber(item.total_tokens)}
                      </div>
                      <div className="mt-1 text-xs leading-5 text-[#9aa39e]">
                        输入 {fmtNumber(item.input_tokens)}
                        <span className="mx-1 text-[#c0c9c4]">/</span>
                        输出 {fmtNumber(item.output_tokens)}
                      </div>
                      <div className="mt-1 text-xs leading-5 text-[#9aa39e]">
                        非缓存输入 {fmtNumber(billableInputTokens(item))}
                      </div>
                    </td>
                    <td className={tdClass}>
                      <div className="text-xs leading-5">
                        缓存输入 {fmtNumber(item.cached_tokens)}
                      </div>
                      <div className="mt-1 text-xs leading-5 text-[#9aa39e]">
                        推理输出 {fmtNumber(item.reasoning_tokens)}
                      </div>
                    </td>
                    <td className={`${tdClass} whitespace-nowrap`}>
                      {fmtCost(item.estimated_cost_usd)}
                    </td>
                    <td className={tdClass}>{fmtDuration(item.duration_ms)}</td>
                    <td className={tdClass}>
                      <div className="text-xs leading-5">
                        请求 {fmtNumber(item.request_bytes)}
                      </div>
                      <div className="mt-1 text-xs leading-5 text-[#9aa39e]">
                        响应 {fmtNumber(item.response_bytes)}
                      </div>
                    </td>
                    <td className={tdClass}>{item.error_type || '-'}</td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td
                    colSpan={13}
                    className="px-4 py-10 text-center text-sm text-[#9aa39e]"
                  >
                    {loading ? '加载中...' : '暂无调用记录'}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </SurfacePanel>
  )
}

export default function AdminDashboardPage() {
  const apiRpc = useMemo(
    () =>
      new JsonRpc({
        url: 'api',
        basePath: ADMIN_BASE_PATH,
        authScope: AUTH_SCOPE.ADMIN,
      }),
    []
  )

  const [loading, setLoading] = useState(false)
  const [errMsg, setErrMsg] = useState('')
  const [summary, setSummary] = useState({})
  const [todaySummary, setTodaySummary] = useState({})
  const [minuteSummary, setMinuteSummary] = useState({})
  const [keys, setKeys] = useState([])
  const [usageItems, setUsageItems] = useState([])
  const [usageBuckets, setUsageBuckets] = useState([])
  const [usageTotal, setUsageTotal] = useState(0)
  const [trendChartType, setTrendChartType] = useState('bar')
  const [trendMetric, setTrendMetric] = useState('requests')

  const dailyBuckets = useMemo(
    () => fillDailyBuckets(usageBuckets, TREND_DAYS),
    [usageBuckets]
  )
  const trendStats = useMemo(() => sumBuckets(dailyBuckets), [dailyBuckets])
  const activeKeys = useMemo(
    () => keys.filter((item) => !item.disabled).length,
    [keys]
  )
  const modelGroups = useMemo(
    () => topUsageGroups(usageItems, (item) => item.model),
    [usageItems]
  )
  const endpointGroups = useMemo(
    () => topUsageGroups(usageItems, (item) => item.endpoint || item.path),
    [usageItems]
  )
  const recentTotalTokens = useMemo(
    () =>
      usageItems.reduce(
        (total, item) => total + asInt(item.total_tokens, 0),
        0
      ),
    [usageItems]
  )
  const sampleP95Duration = useMemo(
    () =>
      percentile(
        usageItems.map((item) => item.duration_ms),
        0.95
      ),
    [usageItems]
  )

  const loadAll = async () => {
    setLoading(true)
    setErrMsg('')
    try {
      const now = Math.floor(Date.now() / 1000)
      const startTime = now - DAY_SECONDS
      const todayStartTime = startOfLocalDayUnix(new Date())
      const minuteStartTime = now - 60
      const trendStartTime = now - TREND_DAYS * DAY_SECONDS
      const [
        summaryRes,
        todaySummaryRes,
        minuteSummaryRes,
        keysRes,
        usageRes,
        bucketsRes,
      ] = await Promise.all([
        apiRpc.call('summary', { start_time: startTime }),
        apiRpc.call('summary', { start_time: todayStartTime }),
        apiRpc.call('summary', { start_time: minuteStartTime }),
        apiRpc.call('key_list', {
          limit: DASHBOARD_KEY_FETCH_LIMIT,
          offset: 0,
        }),
        apiRpc.call('usage_list', {
          limit: PAGE_SIZE,
          offset: 0,
          start_time: startTime,
        }),
        apiRpc.call('usage_buckets', {
          end_time: now,
          group_by: 'day',
          start_time: trendStartTime,
        }),
      ])

      setSummary(summaryRes?.data?.summary || {})
      setTodaySummary(todaySummaryRes?.data?.summary || {})
      setMinuteSummary(minuteSummaryRes?.data?.summary || {})
      setKeys(Array.isArray(keysRes?.data?.items) ? keysRes.data.items : [])
      setUsageItems(
        Array.isArray(usageRes?.data?.items) ? usageRes.data.items : []
      )
      setUsageBuckets(
        Array.isArray(bucketsRes?.data?.items) ? bucketsRes.data.items : []
      )
      setUsageTotal(asInt(usageRes?.data?.total, 0))
    } catch (e) {
      setErrMsg(getActionErrorMessage(e, '加载 API 数据'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadAll()
  }, [])

  return (
    <AdminFrame
      breadcrumb="API / 业务看板"
      title="业务看板"
      description="保留关键运行指标、趋势和最近调用样本；配置和深度分析进入独立页面。"
    >
      {errMsg ? (
        <div className="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
          {errMsg}
        </div>
      ) : null}

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4 2xl:grid-cols-7">
        <SummaryCard
          label="今日消费"
          tone="green"
          value={fmtCost(todaySummary.estimated_cost_usd)}
          sub={`24h ${fmtCost(summary.estimated_cost_usd)}`}
        />
        <SummaryCard
          label="今日请求"
          tone="blue"
          value={fmtNumber(todaySummary.total_requests)}
          sub={`${fmtNumber(todaySummary.success_requests)} 成功 / ${fmtNumber(todaySummary.failed_requests)} 失败`}
        />
        <SummaryCard
          label="错误率"
          tone={asInt(todaySummary.failed_requests, 0) > 0 ? 'red' : 'green'}
          value={fmtRate(
            todaySummary.failed_requests,
            todaySummary.total_requests
          )}
          sub={`成功率 ${fmtRate(todaySummary.success_requests, todaySummary.total_requests)}`}
        />
        <SummaryCard
          label="响应耗时"
          tone="amber"
          value={fmtDuration(todaySummary.average_duration_ms)}
          sub={`最近样本 P95 ${fmtDuration(sampleP95Duration)}`}
        />
        <SummaryCard
          label="当前 RPM / TPM"
          tone="blue"
          value={`${fmtNumber(minuteSummary.total_requests)} RPM`}
          sub={`${fmtNumber(minuteSummary.total_tokens)} TPM`}
        />
        <SummaryCard
          label="上游分布"
          tone="amber"
          value={`${fmtNumber(summary.backend_requests)} / ${fmtNumber(summary.cli_requests)}`}
          sub={`${fmtNumber(summary.fallback_requests)} 次 fallback`}
        />
        <SummaryCard
          label="API 凭据"
          value={fmtNumber(keys.length)}
          sub={`${fmtNumber(activeKeys)} 启用 / ${fmtNumber(keys.length - activeKeys)} 禁用`}
        />
      </div>

      <div className="grid gap-5 xl:grid-cols-[minmax(0,1.45fr)_minmax(340px,0.55fr)]">
        <SurfacePanel variant="admin" className="p-5">
          <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 className="text-base font-bold text-[#1f2d25]">30 天趋势</h2>
              <div className="mt-1 text-sm text-[#7b8780]">
                按天聚合请求、错误、费用、延迟和 Token。
              </div>
            </div>
            <div className="flex flex-col gap-2 sm:items-end">
              <div className="flex flex-wrap gap-2" aria-label="图表类型">
                {TREND_CHART_TYPES.map((item) => (
                  <button
                    key={item.key}
                    type="button"
                    aria-pressed={trendChartType === item.key}
                    className={`rounded-md border px-3 py-2 text-sm font-semibold transition ${
                      trendChartType === item.key
                        ? 'border-[#238a43] bg-[#238a43] text-white'
                        : 'border-[#d6ded8] bg-white text-[#1f2d25] hover:border-[#9dc7aa] hover:bg-[#f1faf4]'
                    }`}
                    onClick={() => setTrendChartType(item.key)}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
              <div className="flex flex-wrap gap-2" aria-label="趋势指标">
                {TREND_METRICS.map((item) => (
                  <button
                    key={item.key}
                    type="button"
                    aria-pressed={trendMetric === item.key}
                    className={`rounded-md border px-3 py-2 text-sm font-semibold transition ${
                      trendMetric === item.key
                        ? 'border-[#238a43] bg-[#238a43] text-white'
                        : 'border-[#d6ded8] bg-white text-[#1f2d25] hover:border-[#9dc7aa] hover:bg-[#f1faf4]'
                    }`}
                    onClick={() => setTrendMetric(item.key)}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            </div>
          </div>
          <UsageTrendChart
            buckets={dailyBuckets}
            chartType={trendChartType}
            metric={trendMetric}
          />
        </SurfacePanel>

        <SurfacePanel variant="admin" className="p-5">
          <div className="mb-5">
            <h2 className="text-base font-bold text-[#1f2d25]">Token 构成</h2>
            <div className="mt-1 text-sm text-[#7b8780]">
              30 天窗口内的主要 Token 类型占比。
            </div>
          </div>
          <TokenComposition stats={trendStats} />
        </SurfacePanel>
      </div>

      <div className="grid gap-5 xl:grid-cols-2">
        <SurfacePanel variant="admin" className="p-5">
          <div className="mb-5">
            <h2 className="text-base font-bold text-[#1f2d25]">模型用量分布</h2>
            <div className="mt-1 text-sm text-[#7b8780]">
              基于最近调用样本按 Token 排序。
            </div>
          </div>
          <UsageGroupList items={modelGroups} totalTokens={recentTotalTokens} />
        </SurfacePanel>

        <SurfacePanel variant="admin" className="p-5">
          <div className="mb-5">
            <h2 className="text-base font-bold text-[#1f2d25]">接口分布</h2>
            <div className="mt-1 text-sm text-[#7b8780]">
              基于最近调用样本按 Token 排序。
            </div>
          </div>
          <UsageGroupList
            items={endpointGroups}
            totalTokens={recentTotalTokens}
          />
        </SurfacePanel>
      </div>

      <RecentCallsTable
        loading={loading}
        usageItems={usageItems}
        usageTotal={usageTotal}
      />
    </AdminFrame>
  )
}
