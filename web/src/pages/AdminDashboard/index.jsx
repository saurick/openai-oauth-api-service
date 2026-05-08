import React, { useEffect, useMemo, useState } from 'react'
import AdminFrame from '@/common/components/layout/AdminFrame'
import SurfacePanel from '@/common/components/layout/SurfacePanel'
import { AUTH_SCOPE } from '@/common/auth/auth'
import { ADMIN_BASE_PATH } from '@/common/utils/adminRpc'
import { getActionErrorMessage } from '@/common/utils/errorMessage'
import { JsonRpc } from '@/common/utils/jsonRpc'

const PAGE_SIZE = 30
const DAY_SECONDS = 24 * 60 * 60
const TREND_DAYS = 30

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
  const safeTotal = Math.max(1, asInt(total, 0))
  return Math.min(
    100,
    Math.max(0, Math.round((asInt(part, 0) / safeTotal) * 100))
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
    const source = byDay.get(localDateKey(bucketStart)) || {}
    return {
      bucket_start: asInt(source.bucket_start, bucketStart),
      cached_tokens: asInt(source.cached_tokens, 0),
      failed_requests: asInt(source.failed_requests, 0),
      input_tokens: asInt(source.input_tokens, 0),
      output_tokens: asInt(source.output_tokens, 0),
      reasoning_tokens: asInt(source.reasoning_tokens, 0),
      success_requests: asInt(source.success_requests, 0),
      total_requests: asInt(source.total_requests, 0),
      total_tokens: asInt(source.total_tokens, 0),
      estimated_cost_usd:
        source.estimated_cost_usd == null
          ? null
          : Number(source.estimated_cost_usd),
    }
  })
}

function sumBuckets(buckets) {
  return buckets.reduce(
    (acc, item) => ({
      cached_tokens: acc.cached_tokens + asInt(item.cached_tokens, 0),
      failed_requests: acc.failed_requests + asInt(item.failed_requests, 0),
      input_tokens: acc.input_tokens + asInt(item.input_tokens, 0),
      output_tokens: acc.output_tokens + asInt(item.output_tokens, 0),
      reasoning_tokens: acc.reasoning_tokens + asInt(item.reasoning_tokens, 0),
      success_requests: acc.success_requests + asInt(item.success_requests, 0),
      total_requests: acc.total_requests + asInt(item.total_requests, 0),
      total_tokens: acc.total_tokens + asInt(item.total_tokens, 0),
      estimated_cost_usd:
        acc.estimated_cost_usd == null || item.estimated_cost_usd == null
          ? null
          : acc.estimated_cost_usd + Number(item.estimated_cost_usd || 0),
    }),
    {
      cached_tokens: 0,
      failed_requests: 0,
      input_tokens: 0,
      output_tokens: 0,
      reasoning_tokens: 0,
      success_requests: 0,
      total_requests: 0,
      total_tokens: 0,
      estimated_cost_usd: 0,
    }
  )
}

function mergeKeyUsage(keys, stats) {
  const byID = new Map(
    (Array.isArray(stats) ? stats : []).map((item) => [
      asInt(item.api_key_id, 0),
      item,
    ])
  )
  const rows = (Array.isArray(keys) ? keys : []).map((key) => {
    const keyID = asInt(key.id, 0)
    const hasStat = byID.has(keyID)
    const stat = byID.get(keyID) || {}
    byID.delete(keyID)
    return {
      api_key_id: keyID,
      api_key_name: key.name || stat.api_key_name || '-',
      api_key_prefix: key.key_prefix || stat.api_key_prefix || '-',
      average_duration_ms: asInt(stat.average_duration_ms, 0),
      cached_tokens: asInt(stat.cached_tokens, 0),
      disabled: Boolean(key.disabled ?? stat.disabled),
      estimated_cost_usd:
        hasStat && stat.estimated_cost_usd == null
          ? null
          : Number(stat.estimated_cost_usd || 0),
      failed_requests: asInt(stat.failed_requests, 0),
      input_tokens: asInt(stat.input_tokens, 0),
      output_tokens: asInt(stat.output_tokens, 0),
      reasoning_tokens: asInt(stat.reasoning_tokens, 0),
      success_requests: asInt(stat.success_requests, 0),
      total_requests: asInt(stat.total_requests, 0),
      total_tokens: asInt(stat.total_tokens, 0),
    }
  })

  for (const stat of byID.values()) {
    rows.push({
      api_key_id: asInt(stat.api_key_id, 0),
      api_key_name: stat.api_key_name || '-',
      api_key_prefix: stat.api_key_prefix || '-',
      average_duration_ms: asInt(stat.average_duration_ms, 0),
      cached_tokens: asInt(stat.cached_tokens, 0),
      disabled: Boolean(stat.disabled),
      estimated_cost_usd:
        stat.estimated_cost_usd == null
          ? null
          : Number(stat.estimated_cost_usd),
      failed_requests: asInt(stat.failed_requests, 0),
      input_tokens: asInt(stat.input_tokens, 0),
      output_tokens: asInt(stat.output_tokens, 0),
      reasoning_tokens: asInt(stat.reasoning_tokens, 0),
      success_requests: asInt(stat.success_requests, 0),
      total_requests: asInt(stat.total_requests, 0),
      total_tokens: asInt(stat.total_tokens, 0),
    })
  }

  return rows.sort(
    (a, b) =>
      b.total_tokens - a.total_tokens ||
      b.total_requests - a.total_requests ||
      String(a.api_key_name).localeCompare(String(b.api_key_name))
  )
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

function SummaryCard({ label, value, sub }) {
  return (
    <SurfacePanel variant="admin" className="p-5">
      <div className="text-sm text-[#7b8780]">{label}</div>
      <div className="mt-3 truncate text-2xl font-medium text-[#1f2d25]">
        {value}
      </div>
      {sub ? <div className="mt-2 text-xs text-[#9aa39e]">{sub}</div> : null}
    </SurfacePanel>
  )
}

function StatusBadge({
  active,
  trueText = '启用',
  falseText = '禁用',
  falseTone = 'neutral',
}) {
  const inactiveClass =
    falseTone === 'danger'
      ? 'bg-rose-50 text-rose-700'
      : 'bg-zinc-100 text-zinc-600'
  return (
    <span
      className={`inline-flex whitespace-nowrap rounded-full px-3 py-1 text-xs font-semibold ${
        active ? 'bg-emerald-50 text-emerald-700' : inactiveClass
      }`}
    >
      {active ? trueText : falseText}
    </span>
  )
}

function ProgressLine({ label, value, tone = 'blue' }) {
  const color = tone === 'green' ? 'bg-[#238a43]' : 'bg-[#1478ff]'
  return (
    <div className="rounded-lg bg-[#f7faf8] px-4 py-3 shadow-[0_2px_8px_rgba(24,61,42,0.04)]">
      <div className="mb-3 inline-flex rounded-md border border-[#d6ded8] bg-white px-3 py-1 text-xs text-[#1f2d25]">
        {label}
      </div>
      <div className="flex items-center gap-3">
        <div className="h-2 flex-1 rounded-full bg-[#e5ece8]">
          <div
            className={`h-2 rounded-full ${color}`}
            style={{ width: `${value}%` }}
          />
        </div>
        <div className="w-10 text-right text-sm text-[#1f2d25]">{value}%</div>
      </div>
    </div>
  )
}

function UsageTrendChart({ buckets }) {
  const maxTokens = Math.max(
    1,
    ...buckets.map((item) => asInt(item.total_tokens, 0))
  )
  const hasData = buckets.some((item) => asInt(item.total_tokens, 0) > 0)

  return (
    <div className="min-w-0">
      <div
        className="grid h-56 items-end gap-1 rounded-lg bg-[#f7faf8] px-3 pb-8 pt-4"
        style={{
          gridTemplateColumns: `repeat(${Math.max(1, buckets.length)}, minmax(6px, 1fr))`,
        }}
      >
        {buckets.map((item) => {
          const total = asInt(item.total_tokens, 0)
          const cachedTokens = Math.min(
            asInt(item.cached_tokens, 0),
            asInt(item.input_tokens, 0)
          )
          const uncachedInputTokens = Math.max(
            0,
            asInt(item.input_tokens, 0) - cachedTokens
          )
          const inputHeight = pct(uncachedInputTokens, total)
          const outputHeight = pct(item.output_tokens, total)
          const cachedHeight = pct(cachedTokens, total)
          const totalHeight = Math.max(total > 0 ? 4 : 0, pct(total, maxTokens))
          return (
            <div
              key={item.bucket_start}
              className="group flex h-full min-w-0 items-end justify-center"
              title={`${fmtShortDate(item.bucket_start)} 请求 ${fmtNumber(item.total_requests)} / Token ${fmtNumber(total)}`}
            >
              <div className="relative flex h-full w-full max-w-5 items-end">
                <div
                  className="flex w-full flex-col justify-end overflow-hidden rounded-t bg-[#cbd8d0]"
                  style={{ height: `${totalHeight}%` }}
                >
                  <div
                    className="bg-[#d6a23a]"
                    style={{ height: `${outputHeight}%` }}
                  />
                  <div
                    className="bg-[#1478ff]"
                    style={{ height: `${inputHeight}%` }}
                  />
                  <div
                    className="bg-[#78c596]"
                    style={{ height: `${cachedHeight}%` }}
                  />
                </div>
              </div>
            </div>
          )
        })}
      </div>
      <div className="mt-3 flex flex-wrap items-center justify-between gap-3 text-xs text-[#7b8780]">
        <span>{fmtShortDate(buckets[0]?.bucket_start)}</span>
        <span>{hasData ? '按 Token 堆叠展示' : '暂无 Token 记录'}</span>
        <span>{fmtShortDate(buckets[buckets.length - 1]?.bucket_start)}</span>
      </div>
      <div className="mt-4 flex flex-wrap gap-4 text-xs text-[#66736b]">
        <span className="inline-flex items-center gap-2">
          <i className="h-2.5 w-2.5 rounded-full bg-[#78c596]" />
          缓存输入
        </span>
        <span className="inline-flex items-center gap-2">
          <i className="h-2.5 w-2.5 rounded-full bg-[#1478ff]" />
          非缓存输入
        </span>
        <span className="inline-flex items-center gap-2">
          <i className="h-2.5 w-2.5 rounded-full bg-[#d6a23a]" />
          输出
        </span>
      </div>
    </div>
  )
}

function TokenComposition({ stats }) {
  const total = Math.max(1, asInt(stats.total_tokens, 0))
  const rows = [
    ['输入 Token', stats.input_tokens, 'bg-[#1478ff]'],
    ['缓存输入 Token', stats.cached_tokens, 'bg-[#78c596]'],
    ['输出 Token', stats.output_tokens, 'bg-[#d6a23a]'],
    ['Reasoning 输出 Token', stats.reasoning_tokens, 'bg-[#9d7bd9]'],
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

function KeyUsageTable({ items, loading }) {
  return (
    <div className={tableWrapClass}>
      <div className="overflow-auto">
        <table className={`${tableClass} min-w-[1080px]`}>
          <thead>
            <tr>
              <th className={thClass}>凭据</th>
              <th className={thClass}>状态</th>
              <th className={thClass}>请求数</th>
              <th className={thClass}>成功 / 失败</th>
              <th className={thClass}>输入 Token</th>
              <th className={thClass}>缓存输入</th>
              <th className={thClass}>输出 Token</th>
              <th className={thClass}>总 Token</th>
              <th className={thClass}>费用估算</th>
              <th className={thClass}>平均耗时</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[#e7efe9] bg-white">
            {items.length > 0 ? (
              items.map((item) => (
                <tr key={String(item.api_key_id || item.api_key_prefix)}>
                  <td className={tdClass}>
                    <div className="max-w-[260px] truncate font-medium text-[#1f2d25]">
                      {item.api_key_name}
                    </div>
                    <div className="mt-1 font-mono text-xs text-[#7b8780]">
                      {item.api_key_prefix}
                    </div>
                  </td>
                  <td className={tdClass}>
                    <StatusBadge active={!item.disabled} />
                  </td>
                  <td className={tdClass}>{fmtNumber(item.total_requests)}</td>
                  <td className={tdClass}>
                    {fmtNumber(item.success_requests)} /{' '}
                    {fmtNumber(item.failed_requests)}
                  </td>
                  <td className={tdClass}>{fmtNumber(item.input_tokens)}</td>
                  <td className={tdClass}>{fmtNumber(item.cached_tokens)}</td>
                  <td className={tdClass}>{fmtNumber(item.output_tokens)}</td>
                  <td className={`${tdClass} font-semibold`}>
                    {fmtNumber(item.total_tokens)}
                  </td>
                  <td className={tdClass}>
                    {fmtCost(item.estimated_cost_usd)}
                  </td>
                  <td className={tdClass}>
                    {fmtNumber(item.average_duration_ms)} ms
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td
                  colSpan={10}
                  className="px-4 py-10 text-center text-sm text-[#9aa39e]"
                >
                  {loading ? '加载中...' : '暂无凭据消耗数据'}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
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
  const [keys, setKeys] = useState([])
  const [keyUsageItems, setKeyUsageItems] = useState([])
  const [usageItems, setUsageItems] = useState([])
  const [usageBuckets, setUsageBuckets] = useState([])
  const [usageTotal, setUsageTotal] = useState(0)

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
  const keyUsageRows = useMemo(
    () => mergeKeyUsage(keys, keyUsageItems),
    [keys, keyUsageItems]
  )
  const recentTotalTokens = useMemo(
    () =>
      usageItems.reduce(
        (total, item) => total + asInt(item.total_tokens, 0),
        0
      ),
    [usageItems]
  )

  const loadAll = async () => {
    setLoading(true)
    setErrMsg('')
    try {
      const now = Math.floor(Date.now() / 1000)
      const startTime = now - DAY_SECONDS
      const trendStartTime = now - TREND_DAYS * DAY_SECONDS
      const [summaryRes, keysRes, usageRes, bucketsRes, keyUsageRes] =
        await Promise.all([
          apiRpc.call('summary', { start_time: startTime }),
          apiRpc.call('key_list', { limit: 100, offset: 0 }),
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
          apiRpc.call('usage_key_summaries', {
            limit: 100,
            offset: 0,
            start_time: startTime,
          }),
        ])

      setSummary(summaryRes?.data?.summary || {})
      setKeys(Array.isArray(keysRes?.data?.items) ? keysRes.data.items : [])
      setUsageItems(
        Array.isArray(usageRes?.data?.items) ? usageRes.data.items : []
      )
      setUsageBuckets(
        Array.isArray(bucketsRes?.data?.items) ? bucketsRes.data.items : []
      )
      setKeyUsageItems(
        Array.isArray(keyUsageRes?.data?.items) ? keyUsageRes.data.items : []
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
      description="只展示调用、Token、费用估算和最近异常线索，不承载配置操作。"
    >
      {errMsg ? (
        <div className="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
          {errMsg}
        </div>
      ) : null}

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <SummaryCard
          label="24h 请求"
          value={fmtNumber(summary.total_requests)}
          sub={`${fmtNumber(summary.success_requests)} 成功 / ${fmtNumber(summary.failed_requests)} 失败`}
        />
        <SummaryCard
          label="24h Token"
          value={fmtNumber(summary.total_tokens)}
          sub={`${fmtNumber(summary.input_tokens)} 输入 / ${fmtNumber(summary.output_tokens)} 输出 · ${fmtCost(summary.estimated_cost_usd)}`}
        />
        <SummaryCard
          label="30 天 Token"
          value={fmtNumber(trendStats.total_tokens)}
          sub={`${fmtNumber(trendStats.cached_tokens)} 缓存输入 / ${fmtCost(trendStats.estimated_cost_usd)}`}
        />
        <SummaryCard
          label="API 凭据"
          value={fmtNumber(keys.length)}
          sub={`${fmtNumber(activeKeys)} 个启用`}
        />
      </div>

      <div className="grid gap-5 xl:grid-cols-[minmax(0,1.45fr)_minmax(340px,0.55fr)]">
        <SurfacePanel variant="admin" className="p-5">
          <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 className="text-base font-bold text-[#1f2d25]">
                30 天调用趋势
              </h2>
              <div className="mt-1 text-sm text-[#7b8780]">
                按天聚合请求数、输入、缓存输入、输出和 Reasoning Token。
              </div>
            </div>
            <div className="text-sm text-[#7b8780]">
              请求 {fmtNumber(trendStats.total_requests)}
            </div>
          </div>
          <UsageTrendChart buckets={dailyBuckets} />
        </SurfacePanel>

        <SurfacePanel variant="admin" className="p-5">
          <div className="mb-5">
            <h2 className="text-base font-bold text-[#1f2d25]">Token 构成</h2>
            <div className="mt-1 text-sm text-[#7b8780]">
              30 天窗口内的主要 token 类型占比。
            </div>
          </div>
          <TokenComposition stats={trendStats} />
        </SurfacePanel>
      </div>

      <SurfacePanel variant="admin" className="p-5">
        <h2 className="text-base font-bold text-[#1f2d25]">调用状态概览</h2>
        <div className="mt-1 text-sm text-[#7b8780]">
          最近 24 小时请求成功 / 失败占比，以及当前 API 凭据启用比例。
        </div>
        <div className="mt-4 grid gap-4 lg:grid-cols-3">
          <ProgressLine
            label="成功请求"
            tone="green"
            value={pct(summary.success_requests, summary.total_requests)}
          />
          <ProgressLine
            label="失败请求"
            value={pct(summary.failed_requests, summary.total_requests)}
          />
          <ProgressLine
            label="启用 API 凭据"
            tone="green"
            value={pct(activeKeys, keys.length)}
          />
        </div>
      </SurfacePanel>

      <SurfacePanel variant="admin" className="p-5 sm:p-6">
        <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-[#1f2d25]">
              24h 凭据消耗
            </h2>
            <div className="mt-1 text-sm text-[#7b8780]">
              按 API 凭据汇总请求数、Token、费用估算和平均耗时。
            </div>
          </div>
          <div className="text-sm text-[#7b8780]">
            {fmtNumber(keyUsageRows.length)} 个凭据
          </div>
        </div>
        <KeyUsageTable items={keyUsageRows} loading={loading} />
      </SurfacePanel>

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

      <SurfacePanel variant="admin" className="p-5 sm:p-6">
        <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-[#1f2d25]">
              30 天按天统计
            </h2>
            <div className="mt-1 text-sm text-[#7b8780]">
              对齐请求数、输入、缓存输入、输出、Reasoning 和总 Token。
            </div>
          </div>
        </div>

        <div className={tableWrapClass}>
          <div className="overflow-auto">
            <table className={`${tableClass} min-w-[980px]`}>
              <thead>
                <tr>
                  <th className={thClass}>日期</th>
                  <th className={thClass}>请求数</th>
                  <th className={thClass}>输入 Token</th>
                  <th className={thClass}>缓存输入</th>
                  <th className={thClass}>输出 Token</th>
                  <th className={thClass}>Reasoning 输出</th>
                  <th className={thClass}>总 Token</th>
                  <th className={thClass}>费用估算</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#e7efe9] bg-white">
                {dailyBuckets
                  .filter((item) => asInt(item.total_requests, 0) > 0)
                  .slice(-10)
                  .reverse()
                  .map((item) => (
                    <tr key={item.bucket_start}>
                      <td className={tdClass}>
                        {fmtShortDate(item.bucket_start)}
                      </td>
                      <td className={tdClass}>
                        {fmtNumber(item.total_requests)}
                      </td>
                      <td className={tdClass}>
                        {fmtNumber(item.input_tokens)}
                      </td>
                      <td className={tdClass}>
                        {fmtNumber(item.cached_tokens)}
                      </td>
                      <td className={tdClass}>
                        {fmtNumber(item.output_tokens)}
                      </td>
                      <td className={tdClass}>
                        {fmtNumber(item.reasoning_tokens)}
                      </td>
                      <td className={`${tdClass} font-semibold`}>
                        {fmtNumber(item.total_tokens)}
                      </td>
                      <td className={tdClass}>
                        {fmtCost(item.estimated_cost_usd)}
                      </td>
                    </tr>
                  ))}
                {dailyBuckets.every(
                  (item) => asInt(item.total_requests, 0) === 0
                ) ? (
                  <tr>
                    <td
                      colSpan={8}
                      className="px-4 py-10 text-center text-sm text-[#9aa39e]"
                    >
                      暂无 30 天调用聚合
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </div>
      </SurfacePanel>

      <SurfacePanel variant="admin" className="p-5 sm:p-6">
        <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-[#1f2d25]">最近调用</h2>
            <div className="mt-1 text-sm text-[#7b8780]">
              24 小时内最近 {usageItems.length} 条 / 共 {usageTotal} 条。
            </div>
          </div>
        </div>

        <div className={tableWrapClass}>
          <div className="overflow-auto">
            <table className={`${tableClass} min-w-[1080px]`}>
              <thead>
                <tr>
                  <th className={thClass}>时间</th>
                  <th className={thClass}>凭据</th>
                  <th className={thClass}>接口</th>
                  <th className={thClass}>模型</th>
                  <th className={thClass}>状态</th>
                  <th className={thClass}>Token</th>
                  <th className={thClass}>费用估算</th>
                  <th className={thClass}>耗时</th>
                  <th className={thClass}>错误</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#e7efe9] bg-white">
                {usageItems.length > 0 ? (
                  usageItems.map((item) => (
                    <tr key={String(item.id)} className="align-top">
                      <td className={tdClass}>{fmtTs(item.created_at)}</td>
                      <td
                        className={`${tdClass} whitespace-nowrap font-mono text-xs`}
                      >
                        {item.api_key_prefix || '-'}
                      </td>
                      <td className={tdClass}>{item.endpoint || item.path}</td>
                      <td
                        className={`${tdClass} whitespace-nowrap font-mono text-xs`}
                      >
                        {item.model || '-'}
                      </td>
                      <td className={tdClass}>
                        <StatusBadge
                          active={!!item.success}
                          trueText={`HTTP ${item.status_code}`}
                          falseText={`HTTP ${item.status_code}`}
                          falseTone="danger"
                        />
                      </td>
                      <td className={tdClass}>
                        {fmtNumber(item.total_tokens)}
                        <div className="mt-1 text-xs text-[#9aa39e]">
                          {fmtNumber(item.input_tokens)} /{' '}
                          {fmtNumber(item.output_tokens)}
                        </div>
                      </td>
                      <td className={tdClass}>
                        {fmtCost(item.estimated_cost_usd)}
                      </td>
                      <td className={tdClass}>
                        {fmtNumber(item.duration_ms)} ms
                      </td>
                      <td className={tdClass}>{item.error_type || '-'}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td
                      colSpan={9}
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
    </AdminFrame>
  )
}
