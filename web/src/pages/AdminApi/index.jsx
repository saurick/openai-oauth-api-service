import React, { useEffect, useId, useMemo, useRef, useState } from 'react'
import AdminFrame from '@/common/components/layout/AdminFrame'
import SurfacePanel from '@/common/components/layout/SurfacePanel'
import { AUTH_SCOPE } from '@/common/auth/auth'
import { ADMIN_BASE_PATH } from '@/common/utils/adminRpc'
import { getActionErrorMessage } from '@/common/utils/errorMessage'
import {
  gatewayErrorTypeFilterLabel,
  gatewayErrorTypeLabel,
  gatewayErrorTypeTitle,
  GATEWAY_ERROR_TYPE_HELP,
} from '@/common/utils/gatewayErrorTypes'
import { JsonRpc } from '@/common/utils/jsonRpc'
import {
  getTableSelectionAfterClick,
  isInteractiveTableTarget,
  TABLE_ROW_INTERACTION_TITLE,
  toggleTablePageSelection,
  toggleTableSelection,
} from '@/common/utils/tableInteraction'
import {
  DAY_SECONDS,
  DEFAULT_DAILY_USAGE_TIME_RANGE,
  DEFAULT_USAGE_TIME_RANGE,
  getUsageTimeRange,
  getUsageTimeWindow,
  USAGE_TIME_RANGE_OPTIONS,
} from '@/common/utils/usageTimeRange'

const PAGE_SIZE = 30
const DASHBOARD_USAGE_SIZE = 8
const DEFAULT_TABLE_PAGE_SIZE = 8
const TABLE_PAGE_SIZE_OPTIONS = [8, 10, 20, 50, 100]
const TABLE_PAGE_SIZE_SELECT_OPTIONS = TABLE_PAGE_SIZE_OPTIONS.map((value) => ({
  label: `${value} 条/页`,
  value,
}))
const MAX_TABLE_FETCH_SIZE = 200
const KEY_TOKEN_WINDOWS = [
  { key: 'today', label: '今天' },
  { key: '24h', label: '24h', seconds: DAY_SECONDS },
  { key: '7d', label: '7 天', seconds: 7 * DAY_SECONDS },
  { key: '30d', label: '30 天', seconds: 30 * DAY_SECONDS },
  { key: '180d', label: '180 天', seconds: 180 * DAY_SECONDS },
  { key: '360d', label: '360 天', seconds: 360 * DAY_SECONDS },
  { key: '1y', label: '1 年', seconds: 365 * DAY_SECONDS },
  { key: '3y', label: '3 年', seconds: 3 * 365 * DAY_SECONDS },
  { key: '5y', label: '5 年', seconds: 5 * 365 * DAY_SECONDS },
]

function getKeyTokenTimeWindow(windowItem, now) {
  if (windowItem.key === 'today') {
    return getUsageTimeWindow('today', now)
  }
  return {
    endTime: now,
    startTime: now - windowItem.seconds,
  }
}

const tableWrapClass = 'overflow-hidden rounded-lg border border-[#dde8df]'
const tableClass = 'admin-data-table text-left text-sm text-[#1f2d25]'
const keyTableClass =
  'admin-data-table admin-key-table text-left text-sm text-[#1f2d25]'
const thClass =
  'whitespace-nowrap bg-[#f5fbf7] px-4 py-3 font-semibold text-[#66736b]'
const tdClass = 'px-4 py-4 text-[#1f2d25]'
const selectionThClass =
  'w-12 whitespace-nowrap bg-[#f5fbf7] px-3 py-3 text-center'
const selectionTdClass = 'w-12 px-3 py-4 text-center'
const inputClass =
  'rounded-md border border-[#d6ded8] bg-white px-3 py-2.5 text-sm text-[#1f2d25] outline-none transition placeholder:text-[#9aa39e] focus:border-[#238a43] focus:ring-2 focus:ring-[#238a43]/15'
const fieldClass = 'grid gap-1.5 text-sm font-medium text-[#365141]'
const fieldHintClass = 'text-xs font-normal leading-5 text-[#7b8780]'
const primaryButtonClass = 'admin-button admin-button-primary'
const secondaryButtonClass = 'admin-button admin-button-default'
const dangerButtonClass = 'admin-button admin-button-danger'
const tableActionButtonClass = 'admin-button admin-button-compact'
const tableDangerButtonClass = 'admin-button admin-button-compact-danger'
const tablePrimaryButtonClass = 'admin-button admin-button-compact-primary'
const toolbarClass = 'admin-module-toolbar'
const filterGroupClass = 'admin-module-filter-group'
const primaryActionsClass = 'admin-module-primary-actions'
const selectionRowClass =
  'admin-module-toolbar-row admin-module-toolbar-row-compact'
const selectionBlockClass = 'admin-module-selection-block'
const selectionActionsClass = 'admin-module-selection-actions'
const selectionTagClass = 'admin-selection-tag'

const MODEL_LIMIT_OPTIONS = [{ label: '允许全部模型', value: '' }]
const KEY_STATUS_FILTER_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '启用', value: 'enabled' },
  { label: '禁用', value: 'disabled' },
]
const USAGE_SUCCESS_FILTER_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '成功', value: 'true' },
  { label: '未完成 / 失败', value: 'false' },
]
const USAGE_STATUS_CODE_FILTER_OPTIONS = [
  { label: '全部状态码', value: '' },
  { label: '200 OK', value: '200' },
  { label: '400 Bad Request', value: '400' },
  { label: '401 Unauthorized', value: '401' },
  { label: '403 Forbidden', value: '403' },
  { label: '413 Payload Too Large', value: '413' },
  { label: '429 Too Many Requests', value: '429' },
  { label: '499 Client Closed Request', value: '499' },
  { label: '500 Internal Server Error', value: '500' },
  { label: '502 Bad Gateway', value: '502' },
  { label: '503 Service Unavailable', value: '503' },
  { label: '504 Gateway Timeout', value: '504' },
]
const CODEX_UPSTREAM_MODE_OPTIONS = [
  { label: 'Backend', value: 'codex_backend' },
  { label: '强制 CLI', value: 'codex_cli' },
]
const CODEX_UPSTREAM_STRATEGY_OPTIONS = [
  {
    label: 'Backend 直连',
    value: 'backend_only',
    description: '失败直接返回上游错误',
  },
  {
    label: 'Backend + CLI 兜底',
    value: 'backend_with_cli_fallback',
    description: '仅纯文本 / 图片可临时降级',
  },
  {
    label: '强制 CLI',
    value: 'codex_cli',
    description: '每次都走服务端 codex exec',
  },
]
const KEY_UPSTREAM_STRATEGY_OPTIONS = [
  {
    label: '继承全局默认',
    value: '',
    description: '跟随上游策略页当前配置',
  },
  ...CODEX_UPSTREAM_STRATEGY_OPTIONS,
]
const DEFAULT_REASONING_EFFORT_OPTIONS = [
  {
    label: '关闭',
    value: '',
    description: '客户端未传 reasoning_effort 时不注入默认档位',
  },
  { label: 'Fast', value: 'low', description: '默认使用 low，优先减少延迟' },
  {
    label: 'Medium',
    value: 'medium',
    description: '默认使用 medium，兼顾速度和推理深度',
  },
  { label: 'High', value: 'high', description: '默认使用 high' },
  { label: 'Deep', value: 'xhigh', description: '默认使用 xhigh' },
]
const KEY_DEFAULT_REASONING_EFFORT_OPTIONS = [
  {
    label: '继承全局默认',
    value: '',
    description: '跟随上游策略页的默认推理档位',
  },
  {
    label: '关闭默认',
    value: 'none',
    description: '即使全局开启，也不为该 key 注入默认 effort',
  },
  ...DEFAULT_REASONING_EFFORT_OPTIONS.filter((option) => option.value),
]
const MODEL_CONTEXT_HELP = {
  window:
    '表格显示当前生效值；弹窗保存的是模型覆盖值。上下文窗口是客户端可看到的模型窗口，默认按 Codex 使用体验取 400K；留空或 0 表示继承，不是无限制。',
  compact:
    '表格显示当前生效值；弹窗保存的是模型覆盖值。开始压缩是转发前先摘要历史；硬拦截是压缩后仍过大则返回 413。留空或 0 表示继承默认 260K / 380K，不是关闭压缩。',
  bytes:
    '表格显示当前生效值；弹窗保存的是模型覆盖值。字节阈值用于兜底 JSON、工具输出和附件文本导致的超大请求。留空或 0 表示继承，不是无限制；它不是计费 token。',
  keep: '表格显示当前生效值；弹窗保存的是模型覆盖值。保留条数是压缩时至少保留的最近 messages / input items 数；留空或 0 表示继承默认值。',
  units:
    '阈值可填整数、K 或 M，例如 260K、0.38M；保留条数只能填整数。留空或 0 表示不写模型覆盖，继续走运维覆盖或内置推荐，不代表无限制。',
}
const USAGE_UPSTREAM_FILTER_OPTIONS = [
  { label: '全部上游', value: '' },
  ...CODEX_UPSTREAM_MODE_OPTIONS,
]
const USAGE_CLIENT_TYPE_OPTIONS = [
  { label: '全部客户端', value: '' },
  { label: 'Codex', value: 'codex' },
  { label: 'OpenCode', value: 'opencode' },
  { label: '其他', value: 'other' },
]
const USAGE_ERROR_FILTER_BASE_OPTIONS = [
  { label: '客户端 / 入口代理取消', value: 'client_canceled' },
  { value: 'context_length_exceeded' },
  { value: 'gateway_api_key_invalid' },
  { value: 'gateway_api_key_disabled' },
  { value: 'gateway_model_disabled' },
  { value: 'gateway_model_not_allowed' },
  { value: 'gateway_rate_limited' },
  { value: 'gateway_quota_exceeded' },
  { value: 'gateway_reasoning_effort_invalid' },
  { value: 'gateway_error' },
  { value: 'codex_backend_auth_failed' },
  { value: 'codex_backend_rate_limited' },
  { value: 'codex_backend_http_5xx' },
  { value: 'codex_backend_timeout' },
  { value: 'codex_backend_response_failed' },
  { value: 'codex_backend_response_incomplete' },
  { value: 'codex_backend_stream_error' },
  { value: 'codex_backend_stream_interrupted' },
  { value: 'codex_backend_http_error' },
  { value: 'codex_backend_upstream_failed' },
  { value: 'codex_cli_timeout' },
  { value: 'codex_cli_not_found' },
  { value: 'codex_cli_empty_prompt' },
  { value: 'codex_cli_empty_answer' },
  { value: 'codex_cli_upstream_failed' },
]
const USAGE_ERROR_FILTER_OPTIONS = [
  { label: '全部错误 / 中断类型', value: '' },
  ...USAGE_ERROR_FILTER_BASE_OPTIONS.map((option) => ({
    label: gatewayErrorTypeFilterLabel(option.value, option.label),
    value: option.value,
  })),
]
const CODEX_REASONING_EFFORT_OPTIONS = [
  { label: 'Low', value: 'low' },
  { label: 'Medium', value: 'medium' },
  { label: 'High', value: 'high' },
  { label: 'XHigh', value: 'xhigh' },
]
const USAGE_REASONING_EFFORT_FILTER_OPTIONS = [
  { label: '全部 Effort', value: '' },
  ...CODEX_REASONING_EFFORT_OPTIONS,
]
const USAGE_TAB_OPTIONS = [
  { key: 'details', label: '调用明细' },
  { key: 'errors', label: '异常请求' },
  { key: 'sessions', label: '会话聚合' },
  { key: 'keys', label: '凭据统计' },
  { key: 'daily', label: '每日模型' },
]
const DEFAULT_USAGE_TAB = 'details'
const TOKEN_LIMIT_UNIT = 1_000_000
const KEY_REMARK_MAX_LENGTH = 80
const KEY_REMARK_PATTERN = '[A-Za-z0-9]*'
const CODEX_MODEL_CATALOG = [
  {
    cached_input_usd_per_million: 0.5,
    input_usd_per_million: 5,
    model_id: 'gpt-5.5',
    output_usd_per_million: 30,
  },
  {
    cached_input_usd_per_million: 0.25,
    input_usd_per_million: 2.5,
    model_id: 'gpt-5.4',
    output_usd_per_million: 15,
  },
  {
    cached_input_usd_per_million: 0.075,
    input_usd_per_million: 0.75,
    model_id: 'gpt-5.4-mini',
    output_usd_per_million: 4.5,
  },
  {
    cached_input_usd_per_million: 0.175,
    input_usd_per_million: 1.75,
    model_id: 'gpt-5.3-codex',
    output_usd_per_million: 14,
  },
  {
    model_id: 'gpt-5.3-codex-spark',
    price_note: 'research preview，价格未定',
  },
  {
    cached_input_usd_per_million: 0.175,
    input_usd_per_million: 1.75,
    model_id: 'gpt-5.2',
    output_usd_per_million: 14,
  },
]
const CODEX_MODEL_IDS = new Set(
  CODEX_MODEL_CATALOG.map((item) => item.model_id)
)

const INITIAL_KEY_FORM = {
  remark: '',
  allowedModels: '',
  upstreamStrategy: '',
  defaultReasoningEffort: '',
  dailyTokenLimit: '',
  weeklyTokenLimit: '',
  dailyInputTokenLimit: '',
  weeklyInputTokenLimit: '',
  dailyOutputTokenLimit: '',
  weeklyOutputTokenLimit: '',
  dailyBillableInputTokenLimit: '',
  weeklyBillableInputTokenLimit: '',
}
const INITIAL_MODEL_CONTEXT_FORM = {
  contextWindowTokens: '',
  contextCompactTokens: '',
  contextHardTokens: '',
  contextCompactBytes: '',
  contextHardBytes: '',
  contextKeepItems: '',
}

function normalizeKeyRemarkInput(value) {
  return String(value || '')
    .replace(/[^A-Za-z0-9]/g, '')
    .slice(0, KEY_REMARK_MAX_LENGTH)
}

const INITIAL_USAGE_FILTERS = {
  keyIds: [],
  model: '',
  reasoningEffort: '',
  success: '',
  statusCode: '',
  upstreamMode: '',
  clientType: '',
  errorType: '',
  timeRange: DEFAULT_USAGE_TIME_RANGE,
}

const VIEW_CONFIG = {
  dashboard: {
    title: '业务看板',
    description:
      '汇总 API 转发、Token 用量、费用估算和最近异常线索，不承载配置操作。',
  },
  keys: {
    section: '转发配置',
    title: 'API 凭据',
    description: '生成、搜索、启停和删除给客户端调用本服务使用的 ogw_ 凭据。',
  },
  models: {
    section: '转发配置',
    title: '模型管理',
    description: '维护 `/v1/models` 返回项，并控制请求是否允许使用对应模型。',
  },
  upstream: {
    section: '转发配置',
    title: '上游策略',
    description:
      '切换 Codex backend 直连、CLI 兜底或强制 CLI，影响后续 API 转发请求。',
  },
  analytics: {
    section: '用量统计',
    title: '用量统计',
    description:
      '先按凭据维度汇总 Token 窗口，后续可继续扩展模型、趋势、服务错误率和延迟分析。',
  },
  usage: {
    section: '用量统计',
    title: '用量日志',
    description:
      '优先查看调用明细、异常请求、会话聚合、凭据统计和每日模型，排查 Token、费用、耗时与错误类型。',
  },
}

function asInt(v, fallback = 0) {
  const n = Number(v)
  return Number.isFinite(n) ? Math.trunc(n) : fallback
}

function fmtNumber(v) {
  return new Intl.NumberFormat().format(asInt(v, 0))
}

function fmtOptionalNumber(v) {
  const n = asInt(v, 0)
  return n > 0 ? fmtNumber(n) : '未配置'
}

function fmtEffectiveContextValue(item, key, suffix = '') {
  const n = asInt(item?.[key], 0)
  if (n <= 0) return '未配置'
  return `${fmtNumber(n)}${suffix ? ` ${suffix}` : ''}`
}

function fmtContextInputValue(value, { allowUnit = true } = {}) {
  const n = asInt(value, 0)
  if (n <= 0) return ''
  if (!allowUnit) return String(n)
  if (n >= 1_000_000 && n % 10_000 === 0) {
    return `${Number((n / 1_000_000).toFixed(2))}M`
  }
  if (n >= 1_000 && n % 1_000 === 0) {
    return `${n / 1_000}K`
  }
  return String(n)
}

function limitInputToNumber(value, { allowUnit = true } = {}) {
  const raw = String(value || '').trim()
  if (!raw) return 0
  const normalized = raw.replace(/\s+/g, '').toLowerCase()
  const match = normalized.match(/^(\d+(?:\.\d+)?)([km])?$/)
  if (!match) return null
  if (match[2] && !allowUnit) return null
  const multiplier = match[2] === 'm' ? 1_000_000 : match[2] === 'k' ? 1_000 : 1
  const n = Number(match[1]) * multiplier
  if (!Number.isFinite(n)) return null
  return Math.trunc(n)
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

function fmtDecimalNumber(v) {
  const n = Number(v)
  return new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 2,
  }).format(Number.isFinite(n) ? n : 0)
}

function tokenLimitInputToTokens(value) {
  const n = Number(value)
  if (!Number.isFinite(n) || n <= 0) return 0
  return Math.round(n * TOKEN_LIMIT_UNIT)
}

function tokenLimitTokensToInput(value) {
  const tokens = asInt(value, 0)
  if (tokens <= 0) return ''
  return String(tokens / TOKEN_LIMIT_UNIT)
}

function fmtTokenLimit(value) {
  const tokens = asInt(value, 0)
  if (tokens <= 0) return '不限'
  return `${fmtDecimalNumber(tokens / TOKEN_LIMIT_UNIT)} 百万`
}

function renderTokenLimitPair(label, dailyValue, weeklyValue) {
  const hasLimit = asInt(dailyValue, 0) > 0 || asInt(weeklyValue, 0) > 0
  if (!hasLimit) return null
  return (
    <div className="whitespace-nowrap">
      <span className="text-[#7b8780]">{label}：</span>
      <span>
        日 {fmtTokenLimit(dailyValue)} / 周 {fmtTokenLimit(weeklyValue)}
      </span>
    </div>
  )
}

function fmtTs(ts) {
  if (!ts) return '-'
  const d = new Date(Number(ts) * 1000)
  if (Number.isNaN(d.getTime())) return String(ts)
  return d.toLocaleString()
}

function fmtDate(ts) {
  if (!ts) return '-'
  const d = new Date(Number(ts) * 1000)
  if (Number.isNaN(d.getTime())) return String(ts)
  return d.toLocaleDateString()
}

function fmtCost(v) {
  if (v == null || v === '') return '未配置价格'
  const n = Number(v)
  if (!Number.isFinite(n)) return '未配置价格'
  return `$${n.toFixed(4)}`
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

function clientTypeLabel(value) {
  const item = USAGE_CLIENT_TYPE_OPTIONS.find(
    (option) => option.value === String(value || '')
  )
  return item?.label || '其他'
}

function upstreamStrategyLabel(value) {
  const item = KEY_UPSTREAM_STRATEGY_OPTIONS.find(
    (option) => option.value === String(value || '')
  )
  return item?.label || '继承全局默认'
}

function defaultReasoningEffortLabel(
  value,
  options = KEY_DEFAULT_REASONING_EFFORT_OPTIONS
) {
  const item = options.find((option) => option.value === String(value || ''))
  return item?.label || '继承全局默认'
}

function reasoningEffortLabel(value) {
  const item = CODEX_REASONING_EFFORT_OPTIONS.find(
    (option) => option.value === value
  )
  return item?.label || '未记录'
}

function isClientCanceledUsage(item) {
  return String(item?.error_type || '') === 'client_canceled'
}

function renderUpstreamStats(item) {
  return (
    <div className="text-xs leading-5">
      <div>Backend {fmtNumber(item?.backend_requests)}</div>
      <div className="text-[#9aa39e]">
        CLI {fmtNumber(item?.cli_requests)}
        <span className="mx-1 text-[#c0c9c4]">/</span>
        fallback {fmtNumber(item?.fallback_requests)}
      </div>
    </div>
  )
}

function renderClientStats(item) {
  return (
    <div className="text-xs leading-5">
      <div>Codex {fmtNumber(item?.codex_requests)}</div>
      <div className="text-[#9aa39e]">
        OpenCode {fmtNumber(item?.opencode_requests)}
        <span className="mx-1 text-[#c0c9c4]">/</span>
        其他 {fmtNumber(item?.other_client_requests)}
      </div>
    </div>
  )
}

function createInitialPagination() {
  return {
    current: 1,
    pageSize: DEFAULT_TABLE_PAGE_SIZE,
  }
}

function getTotalPages(total, pageSize) {
  return Math.max(1, Math.ceil(asInt(total, 0) / Math.max(1, pageSize)))
}

function clampPagination(pagination, total) {
  const pageSize = Math.max(
    1,
    asInt(pagination?.pageSize, DEFAULT_TABLE_PAGE_SIZE)
  )
  const totalPages = getTotalPages(total, pageSize)
  const current = Math.min(
    Math.max(1, asInt(pagination?.current, 1)),
    totalPages
  )
  return { current, pageSize }
}

function paginateItems(items, pagination) {
  const list = Array.isArray(items) ? items : []
  const { current, pageSize } = clampPagination(pagination, list.length)
  const start = (current - 1) * pageSize
  return list.slice(start, start + pageSize)
}

function getPaginationOffset(pagination) {
  const pageSize = Math.max(
    1,
    asInt(pagination?.pageSize, DEFAULT_TABLE_PAGE_SIZE)
  )
  const current = Math.max(1, asInt(pagination?.current, 1))
  return (current - 1) * pageSize
}

function getVisiblePaginationItems(current, totalPages) {
  if (totalPages <= 5) {
    return Array.from({ length: totalPages }, (_, index) => index + 1)
  }

  if (current <= 3) {
    return [1, 2, 3, 'next-gap', totalPages]
  }

  if (current >= totalPages - 2) {
    return [1, 'prev-gap', totalPages - 2, totalPages - 1, totalPages]
  }

  return [
    1,
    'prev-gap',
    current - 1,
    current,
    current + 1,
    'next-gap',
    totalPages,
  ]
}

function usageFiltersForTab(filters, tab) {
  if (tab === 'errors') {
    return {
      ...filters,
      success: 'false',
      excludeErrorType:
        filters.errorType || filters.statusCode === '499'
          ? ''
          : 'client_canceled',
    }
  }
  return filters
}

function applyUsageTabDefaultTimeRange(filters, tab, timeRangeTouched) {
  if (timeRangeTouched) {
    return filters
  }
  if (tab === 'daily' && filters.timeRange === DEFAULT_USAGE_TIME_RANGE) {
    return {
      ...filters,
      timeRange: DEFAULT_DAILY_USAGE_TIME_RANGE,
    }
  }
  if (tab !== 'daily' && filters.timeRange === DEFAULT_DAILY_USAGE_TIME_RANGE) {
    return {
      ...filters,
      timeRange: DEFAULT_USAGE_TIME_RANGE,
    }
  }
  return filters
}

function splitModels(value) {
  return String(value || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function isCodexModelId(modelId) {
  return CODEX_MODEL_IDS.has(String(modelId || '').trim())
}

function normalizeCodexModelPrices(items) {
  const pricesByModelID = new Map(
    CODEX_MODEL_CATALOG.map((item) => [item.model_id, item])
  )
  for (const item of Array.isArray(items) ? items : []) {
    if (item?.model_id && isCodexModelId(item.model_id)) {
      pricesByModelID.set(item.model_id, item)
    }
  }
  return Array.from(pricesByModelID.values())
}

function normalizeCodexModels(items) {
  const modelsByID = new Map()
  for (const item of CODEX_MODEL_CATALOG) {
    modelsByID.set(item.model_id, {
      enabled: true,
      model_id: item.model_id,
      owned_by: 'openai',
      source: 'seed',
    })
  }
  for (const item of Array.isArray(items) ? items : []) {
    if (item?.model_id && isCodexModelId(item.model_id)) {
      modelsByID.set(item.model_id, item)
    }
  }
  return Array.from(modelsByID.values())
}

function getModelOptions(models, currentValue = '', officialPrices = []) {
  const values = []
  const officialModelIDs = new Set(
    (Array.isArray(officialPrices) ? officialPrices : [])
      .map((item) => item?.model_id)
      .filter(Boolean)
  )
  for (const item of Array.isArray(models) ? models : []) {
    if (
      item?.model_id &&
      (officialModelIDs.size === 0 || officialModelIDs.has(item.model_id))
    ) {
      values.push(item.model_id)
    }
  }
  for (const item of Array.isArray(officialPrices) ? officialPrices : []) {
    if (item?.model_id) values.push(item.model_id)
  }
  if (
    currentValue &&
    (officialModelIDs.size === 0 || officialModelIDs.has(currentValue))
  ) {
    values.push(currentValue)
  }
  return Array.from(new Set(values.filter(Boolean)))
}

function mapPricesByModelID(items) {
  return new Map(
    (Array.isArray(items) ? items : [])
      .filter((item) => item?.model_id)
      .map((item) => [item.model_id, item])
  )
}

function fmtPricePerMillion(value) {
  if (value == null || value === '') return '未配置'
  const n = Number(value)
  if (!Number.isFinite(n)) return '未配置'
  if (n === 0) return '$0'
  return `$${new Intl.NumberFormat(undefined, {
    maximumFractionDigits: 4,
  }).format(n)}`
}

function modelContextFormFromItem(item) {
  return {
    contextWindowTokens: String(asInt(item?.context_window_tokens, 0) || ''),
    contextCompactTokens: String(asInt(item?.context_compact_tokens, 0) || ''),
    contextHardTokens: String(asInt(item?.context_hard_tokens, 0) || ''),
    contextCompactBytes: String(asInt(item?.context_compact_bytes, 0) || ''),
    contextHardBytes: String(asInt(item?.context_hard_bytes, 0) || ''),
    contextKeepItems: String(asInt(item?.context_keep_items, 0) || ''),
  }
}

function fmtPriceTriplet(price) {
  if (!price) return '未配置官方标准价'
  if (price.price_note) return price.price_note
  return `输入 ${fmtPricePerMillion(price.input_usd_per_million)} / 缓存 ${fmtPricePerMillion(price.cached_input_usd_per_million)} / 输出 ${fmtPricePerMillion(price.output_usd_per_million)}`
}

function normalizeSelectOptions(options) {
  return (Array.isArray(options) ? options : [])
    .map((option) => {
      if (typeof option === 'string' || typeof option === 'number') {
        return {
          label: String(option),
          value: String(option),
        }
      }
      return {
        label: String(option?.label ?? option?.value ?? ''),
        value: String(option?.value ?? ''),
      }
    })
    .filter((option) => option.label)
}

function normalizeSelectedValues(value) {
  return Array.from(
    new Set(
      (Array.isArray(value) ? value : [])
        .map((item) => String(item || '').trim())
        .filter(Boolean)
    )
  )
}

function SearchableSelect({
  ariaLabel,
  className = inputClass,
  disabled = false,
  menuPlacement = 'bottom',
  onChange,
  options,
  placeholder = '输入筛选',
  value,
}) {
  const listboxId = useId()
  const rootRef = useRef(null)
  const inputRef = useRef(null)
  const normalizedOptions = useMemo(
    () => normalizeSelectOptions(options),
    [options]
  )
  const selectedOption = useMemo(
    () =>
      normalizedOptions.find(
        (option) => String(option.value) === String(value)
      ) || null,
    [normalizedOptions, value]
  )
  const selectedLabel = selectedOption?.label || ''
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState(selectedLabel)

  useEffect(() => {
    if (!open) {
      setQuery(selectedLabel)
    }
  }, [open, selectedLabel])

  useEffect(() => {
    if (!open) return undefined

    const handlePointerDown = (event) => {
      if (rootRef.current?.contains(event.target)) return
      setOpen(false)
      setQuery(selectedLabel)
    }

    window.addEventListener('pointerdown', handlePointerDown)
    return () => window.removeEventListener('pointerdown', handlePointerDown)
  }, [open, selectedLabel])

  const activeQuery = query === selectedLabel ? '' : query.trim().toLowerCase()
  const filteredOptions = activeQuery
    ? normalizedOptions.filter((option) => {
        const label = option.label.toLowerCase()
        const optionValue = String(option.value).toLowerCase()
        return label.includes(activeQuery) || optionValue.includes(activeQuery)
      })
    : normalizedOptions

  const selectOption = (option) => {
    onChange?.(option.value)
    setQuery(option.label)
    setOpen(false)
  }

  const resetToSelected = () => {
    setOpen(false)
    setQuery(selectedLabel)
  }

  return (
    <div
      ref={rootRef}
      className="admin-searchable-select"
      data-menu-placement={menuPlacement}
      data-open={open ? 'true' : 'false'}
    >
      <input
        ref={inputRef}
        type="text"
        role="combobox"
        aria-autocomplete="list"
        aria-controls={listboxId}
        aria-expanded={open}
        aria-label={ariaLabel}
        autoComplete="off"
        disabled={disabled}
        value={query}
        onChange={(e) => {
          setQuery(e.target.value)
          setOpen(true)
        }}
        onFocus={() => {
          setOpen(true)
          window.requestAnimationFrame(() => inputRef.current?.select())
        }}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            e.preventDefault()
            resetToSelected()
            return
          }
          if (e.key === 'Enter' && open) {
            e.preventDefault()
            if (filteredOptions.length > 0) {
              selectOption(filteredOptions[0])
            }
          }
        }}
        onBlur={(e) => {
          if (!rootRef.current?.contains(e.relatedTarget)) {
            resetToSelected()
          }
        }}
        className={`${className} admin-searchable-select-input`}
        placeholder={placeholder}
      />
      <div className="admin-searchable-select-caret" aria-hidden="true" />
      {open && !disabled ? (
        <div
          id={listboxId}
          role="listbox"
          className="admin-searchable-select-menu"
        >
          {filteredOptions.length > 0 ? (
            filteredOptions.map((option) => {
              const selected = String(option.value) === String(value)
              return (
                <button
                  key={`${option.value}-${option.label}`}
                  type="button"
                  role="option"
                  aria-selected={selected}
                  className={`admin-searchable-select-option ${
                    selected ? 'admin-searchable-select-option-active' : ''
                  }`}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => selectOption(option)}
                >
                  {option.label}
                </button>
              )
            })
          ) : (
            <div className="admin-searchable-select-empty">无匹配选项</div>
          )}
        </div>
      ) : null}
    </div>
  )
}

function SearchableMultiSelect({
  ariaLabel,
  className = inputClass,
  disabled = false,
  menuPlacement = 'bottom',
  onChange,
  options,
  placeholder = '输入筛选',
  summaryLabel,
  value,
}) {
  const listboxId = useId()
  const rootRef = useRef(null)
  const inputRef = useRef(null)
  const normalizedOptions = useMemo(
    () => normalizeSelectOptions(options),
    [options]
  )
  const selectedValues = useMemo(() => normalizeSelectedValues(value), [value])
  const selectedValueSet = useMemo(
    () => new Set(selectedValues),
    [selectedValues]
  )
  const selectedLabels = useMemo(
    () =>
      normalizedOptions
        .filter((option) => selectedValueSet.has(String(option.value)))
        .map((option) => option.label),
    [normalizedOptions, selectedValueSet]
  )
  const selectedSummary =
    selectedLabels.length > 0
      ? selectedLabels.length === 1
        ? selectedLabels[0]
        : `已选 ${selectedLabels.length} 个${summaryLabel || '选项'}`
      : ''
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState(selectedSummary)

  useEffect(() => {
    if (!open) {
      setQuery(selectedSummary)
    }
  }, [open, selectedSummary])

  useEffect(() => {
    if (!open) return undefined

    const handlePointerDown = (event) => {
      if (rootRef.current?.contains(event.target)) return
      setOpen(false)
      setQuery(selectedSummary)
    }

    window.addEventListener('pointerdown', handlePointerDown)
    return () => window.removeEventListener('pointerdown', handlePointerDown)
  }, [open, selectedSummary])

  const activeQuery =
    query === selectedSummary ? '' : query.trim().toLowerCase()
  const filteredOptions = activeQuery
    ? normalizedOptions.filter((option) => {
        const label = option.label.toLowerCase()
        const optionValue = String(option.value).toLowerCase()
        return label.includes(activeQuery) || optionValue.includes(activeQuery)
      })
    : normalizedOptions

  const toggleOption = (option) => {
    const optionValue = String(option.value)
    const nextValues = selectedValueSet.has(optionValue)
      ? selectedValues.filter((item) => item !== optionValue)
      : [...selectedValues, optionValue]
    onChange?.(nextValues)
    setQuery('')
    setOpen(true)
    window.requestAnimationFrame(() => inputRef.current?.focus())
  }

  const clearSelection = () => {
    onChange?.([])
    setQuery('')
    setOpen(true)
    window.requestAnimationFrame(() => inputRef.current?.focus())
  }

  const resetToSelected = () => {
    setOpen(false)
    setQuery(selectedSummary)
  }

  return (
    <div
      ref={rootRef}
      className="admin-searchable-select admin-searchable-multi-select"
      data-menu-placement={menuPlacement}
      data-open={open ? 'true' : 'false'}
    >
      <input
        ref={inputRef}
        type="text"
        role="combobox"
        aria-autocomplete="list"
        aria-controls={listboxId}
        aria-expanded={open}
        aria-label={ariaLabel}
        autoComplete="off"
        disabled={disabled}
        value={query}
        onChange={(e) => {
          setQuery(e.target.value)
          setOpen(true)
        }}
        onFocus={() => {
          setOpen(true)
          window.requestAnimationFrame(() => inputRef.current?.select())
        }}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            e.preventDefault()
            resetToSelected()
            return
          }
          if (e.key === 'Enter' && open) {
            e.preventDefault()
            if (filteredOptions.length > 0) {
              toggleOption(filteredOptions[0])
            }
          }
        }}
        onBlur={(e) => {
          if (!rootRef.current?.contains(e.relatedTarget)) {
            resetToSelected()
          }
        }}
        className={`${className} admin-searchable-select-input`}
        placeholder={placeholder}
      />
      {selectedValues.length > 0 ? (
        <button
          type="button"
          className="admin-searchable-select-clear"
          onMouseDown={(e) => e.preventDefault()}
          onClick={clearSelection}
          aria-label={`清空${ariaLabel}`}
        >
          ×
        </button>
      ) : (
        <div className="admin-searchable-select-caret" aria-hidden="true" />
      )}
      {open && !disabled ? (
        <div
          id={listboxId}
          role="listbox"
          aria-multiselectable="true"
          className="admin-searchable-select-menu"
        >
          {filteredOptions.length > 0 ? (
            filteredOptions.map((option) => {
              const selected = selectedValueSet.has(String(option.value))
              return (
                <button
                  key={`${option.value}-${option.label}`}
                  type="button"
                  role="option"
                  aria-selected={selected}
                  className={`admin-searchable-select-option admin-searchable-multi-option ${
                    selected ? 'admin-searchable-select-option-active' : ''
                  }`}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => toggleOption(option)}
                >
                  <span
                    className="admin-searchable-multi-check"
                    aria-hidden="true"
                  >
                    {selected ? '✓' : ''}
                  </span>
                  <span>{option.label}</span>
                </button>
              )
            })
          ) : (
            <div className="admin-searchable-select-empty">无匹配选项</div>
          )}
        </div>
      ) : null}
    </div>
  )
}

function mapStatsByKeyID(items) {
  return new Map(
    (Array.isArray(items) ? items : []).map((item) => [
      asInt(item.api_key_id, 0),
      item,
    ])
  )
}

function mergeKeyTokenStats(keys, statsByWindow) {
  return (Array.isArray(keys) ? keys : []).map((key) => {
    const keyID = asInt(key.id, 0)
    const tokens = {}
    const upstream = {}
    for (const windowItem of KEY_TOKEN_WINDOWS) {
      const stat = statsByWindow?.[windowItem.key]?.get(keyID)
      tokens[windowItem.key] = asInt(stat?.total_tokens, 0)
      upstream[windowItem.key] = {
        backend_requests: asInt(stat?.backend_requests, 0),
        cli_requests: asInt(stat?.cli_requests, 0),
        fallback_requests: asInt(stat?.fallback_requests, 0),
      }
    }
    return {
      disabled: Boolean(key.disabled),
      id: keyID,
      name: key.name || '无备注',
      prefix: key.key_prefix || '-',
      tokens,
      upstream,
    }
  })
}

function SummaryCard({ label, value, sub, help, helpAlign = 'start' }) {
  return (
    <SurfacePanel variant="admin" className="admin-summary-card p-4">
      <div className="text-xs font-medium uppercase tracking-[0.18em] text-[#7b8780]">
        <HeaderWithHelp help={help} align={helpAlign}>
          {label}
        </HeaderWithHelp>
      </div>
      <div className="mt-3 text-2xl font-semibold text-[#1f2d25]">{value}</div>
      {sub ? <div className="mt-1 text-sm text-[#7b8780]">{sub}</div> : null}
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

function HeaderWithHelp({ children, help, align }) {
  if (!help) return children
  return (
    <span className="admin-th-help-wrap">
      <span>{children}</span>
      <button
        type="button"
        className="admin-th-help"
        aria-label={`说明：${help}`}
        data-tooltip={help}
        data-tooltip-align={align || undefined}
      >
        ?
      </button>
    </span>
  )
}

function ErrorTypeCell({ value }) {
  const code = String(value || '').trim()
  if (!code) return '-'
  const label = gatewayErrorTypeLabel(code)
  return (
    <div className="max-w-[240px]" title={gatewayErrorTypeTitle(code)}>
      <div className="break-all font-mono text-xs leading-5">{code}</div>
      {label ? (
        <div className="mt-1 text-xs leading-5 text-[#7b8780]">{label}</div>
      ) : null}
    </div>
  )
}

function DiagnosticCell({ item }) {
  const diagnostic = item?.diagnostic || {}
  const summary = String(item?.diagnostic_summary || '').trim()
  const chips = []
  if (diagnostic.backend_only) chips.push('backend-only')
  if (diagnostic.fallback_blocked) chips.push('fallback blocked')
  else if (diagnostic.fallback_enabled) chips.push('fallback enabled')
  if (diagnostic.context_compacted) chips.push('context compacted')
  if (diagnostic.upstream_http_status) {
    chips.push(`上游 HTTP ${diagnostic.upstream_http_status}`)
  }
  const body = String(diagnostic.upstream_body || '').trim()
  const compactSummary = String(
    diagnostic.context_compaction_summary || ''
  ).trim()

  if (!summary && chips.length === 0 && !body && !compactSummary) return '-'

  return (
    <div
      className="max-w-[280px] text-xs leading-5"
      title={summary || body || compactSummary}
    >
      {chips.length > 0 ? (
        <div className="flex flex-wrap gap-1">
          {chips.map((chip) => (
            <span
              key={chip}
              className="rounded-full bg-amber-50 px-2 py-0.5 font-semibold text-amber-700"
            >
              {chip}
            </span>
          ))}
        </div>
      ) : null}
      <div className="mt-1 text-[#7b8780]">
        请求 {fmtNumber(diagnostic.request_bytes ?? item?.request_bytes)}B
        <span className="mx-1 text-[#c0c9c4]">/</span>
        响应 {fmtNumber(diagnostic.response_bytes ?? item?.response_bytes)}B
      </div>
      {diagnostic.context_compacted ? (
        <div className="mt-1 text-[#7b8780]">
          压缩 {fmtNumber(diagnostic.context_original_estimated_tokens)}
          {' -> '}
          {fmtNumber(diagnostic.context_compacted_estimated_tokens)} tokens
          <span className="mx-1 text-[#c0c9c4]">/</span>
          {fmtNumber(diagnostic.context_original_bytes)}
          {' -> '}
          {fmtNumber(diagnostic.context_compacted_bytes)}B
        </div>
      ) : null}
      {body ? (
        <div className="mt-1 line-clamp-2 break-all font-mono text-[#9aa39e]">
          {body}
        </div>
      ) : null}
      {compactSummary ? (
        <div className="mt-1 line-clamp-2 break-all text-[#7b8780]">
          {compactSummary}
        </div>
      ) : null}
    </div>
  )
}

function CopyButton({ value, label = '复制' }) {
  const [copied, setCopied] = useState(false)

  const copy = async () => {
    if (!value) return
    try {
      await navigator.clipboard.writeText(value)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1400)
    } catch (error) {
      console.warn('复制凭据失败', error)
    }
  }

  return (
    <button
      type="button"
      onClick={copy}
      disabled={!value}
      className={tableActionButtonClass}
    >
      {copied ? '已复制' : label}
    </button>
  )
}

function apiKeyRemark(item) {
  return item?.api_key_name || item?.name || '无备注'
}

function apiKeyPlainText(item) {
  return item?.plain_key || ''
}

function apiKeyDisplayText(item) {
  const plainKey = apiKeyPlainText(item)
  if (plainKey) return plainKey
  if (item?.key_prefix && item?.key_last4) {
    return `${item.key_prefix}…${item.key_last4}`
  }
  return item?.key_prefix || '-'
}

function ApiKeyUsageCell({ item }) {
  return (
    <div className="min-w-[160px]">
      <div className="max-w-[260px] truncate font-medium text-[#1f2d25]">
        {apiKeyRemark(item)}
      </div>
      <div className="mt-1 font-mono text-xs text-[#7b8780]">
        {item?.api_key_prefix || item?.prefix || '-'}
      </div>
    </div>
  )
}

function TablePagination({ total, pagination, onChange, disabled = false }) {
  const currentState = clampPagination(pagination, total)
  const totalPages = getTotalPages(total, currentState.pageSize)
  const paginationItems = getVisiblePaginationItems(
    currentState.current,
    totalPages
  )

  const setCurrent = (current) => {
    onChange?.({
      current: Math.min(Math.max(1, current), totalPages),
      pageSize: currentState.pageSize,
    })
  }

  const setPageSize = (pageSize) => {
    onChange?.({
      current: 1,
      pageSize,
    })
  }

  return (
    <div className="admin-table-pagination" aria-label="表格分页">
      <div className="admin-table-pagination-controls">
        <div className="admin-table-pagination-summary">
          共 {fmtNumber(total)} 条
        </div>
        <button
          type="button"
          onClick={() => setCurrent(currentState.current - 1)}
          disabled={disabled || currentState.current <= 1}
          className="admin-page-button admin-page-button-arrow"
          aria-label="上一页"
        >
          <span aria-hidden="true">‹</span>
        </button>
        <div className="admin-table-page-numbers" aria-label="页码">
          {paginationItems.map((item) =>
            typeof item === 'number' ? (
              <button
                key={item}
                type="button"
                className={`admin-page-button ${
                  item === currentState.current
                    ? 'admin-page-button-current'
                    : ''
                }`}
                disabled={disabled || item === currentState.current}
                aria-current={
                  item === currentState.current ? 'page' : undefined
                }
                aria-label={`第 ${item} 页`}
                onClick={() => setCurrent(item)}
              >
                {item}
              </button>
            ) : (
              <span
                key={item}
                className="admin-table-pagination-ellipsis"
                aria-hidden="true"
              >
                …
              </span>
            )
          )}
        </div>
        <button
          type="button"
          onClick={() => setCurrent(currentState.current + 1)}
          disabled={disabled || currentState.current >= totalPages}
          className="admin-page-button admin-page-button-arrow"
          aria-label="下一页"
        >
          <span aria-hidden="true">›</span>
        </button>
        <label className="admin-table-page-size">
          <SearchableSelect
            value={currentState.pageSize}
            onChange={(nextValue) =>
              setPageSize(asInt(nextValue, DEFAULT_TABLE_PAGE_SIZE))
            }
            disabled={disabled}
            ariaLabel="每页条数"
            className="admin-table-page-size-input"
            menuPlacement="top"
            options={TABLE_PAGE_SIZE_SELECT_OPTIONS}
          />
        </label>
      </div>
    </div>
  )
}

export default function AdminApiPage({ view = 'dashboard' }) {
  const currentView = VIEW_CONFIG[view] ? view : 'dashboard'
  const currentConfig = VIEW_CONFIG[currentView]
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
  const [keyTotal, setKeyTotal] = useState(0)
  const [keyTokenStatsByWindow, setKeyTokenStatsByWindow] = useState({})
  const [models, setModels] = useState([])
  const [modelTotal, setModelTotal] = useState(0)
  const [officialModelPrices, setOfficialModelPrices] = useState([])
  const [usageItems, setUsageItems] = useState([])
  const [usageTotal, setUsageTotal] = useState(0)
  const [usageSessionItems, setUsageSessionItems] = useState([])
  const [usageSessionTotal, setUsageSessionTotal] = useState(0)
  const [usageBuckets, setUsageBuckets] = useState([])
  const [gatewayUpstreamStrategy, setGatewayUpstreamStrategy] =
    useState('backend_only')
  const [gatewayDefaultReasoningEffort, setGatewayDefaultReasoningEffort] =
    useState('')
  const [gatewayUpstreamSaving, setGatewayUpstreamSaving] = useState(false)
  const [usageTab, setUsageTab] = useState(DEFAULT_USAGE_TAB)
  const [selectedUsageBucket, setSelectedUsageBucket] = useState(null)
  const [selectedUsageBucketItems, setSelectedUsageBucketItems] = useState([])
  const [selectedUsageBucketTotal, setSelectedUsageBucketTotal] = useState(0)
  const [usageBucketDetailPagination, setUsageBucketDetailPagination] =
    useState(createInitialPagination)
  const [usageBucketDetailLoading, setUsageBucketDetailLoading] =
    useState(false)
  const [selectedUsageSession, setSelectedUsageSession] = useState(null)
  const [selectedUsageSessionItems, setSelectedUsageSessionItems] = useState([])
  const [usageSessionDetailLoading, setUsageSessionDetailLoading] =
    useState(false)
  const [newKey, setNewKey] = useState(null)
  const [newKeyBatch, setNewKeyBatch] = useState([])
  const [editingKeyId, setEditingKeyId] = useState(null)
  const [resettingKey, setResettingKey] = useState(false)
  const [confirmDialog, setConfirmDialog] = useState(null)
  const [confirmSubmitting, setConfirmSubmitting] = useState(false)
  const [keyModalOpen, setKeyModalOpen] = useState(false)
  const [keyForm, setKeyForm] = useState(INITIAL_KEY_FORM)
  const [keySearchInput, setKeySearchInput] = useState('')
  const [keySearch, setKeySearch] = useState('')
  const [keyModelFilter, setKeyModelFilter] = useState('')
  const [keyStatusFilter, setKeyStatusFilter] = useState('')
  const [keyPagination, setKeyPagination] = useState(createInitialPagination)
  const [keyStatsPagination, setKeyStatsPagination] = useState(
    createInitialPagination
  )
  const [modelPagination, setModelPagination] = useState(
    createInitialPagination
  )
  const [modelContextModalOpen, setModelContextModalOpen] = useState(false)
  const [editingModelContext, setEditingModelContext] = useState(null)
  const [modelContextForm, setModelContextForm] = useState(
    INITIAL_MODEL_CONTEXT_FORM
  )
  const [usagePagination, setUsagePagination] = useState(
    createInitialPagination
  )
  const [dailyUsagePagination, setDailyUsagePagination] = useState(
    createInitialPagination
  )
  const [selectedKeyIds, setSelectedKeyIds] = useState([])
  const [usageFilters, setUsageFilters] = useState(INITIAL_USAGE_FILTERS)
  const [appliedUsageFilters, setAppliedUsageFilters] = useState(
    INITIAL_USAGE_FILTERS
  )
  const [usageTimeRangeTouched, setUsageTimeRangeTouched] = useState(false)
  const pageKeySelectionRef = useRef(null)
  const selectedKeyIdSet = useMemo(
    () => new Set(selectedKeyIds),
    [selectedKeyIds]
  )
  const modelOptions = useMemo(
    () => getModelOptions(models, keyForm.allowedModels, officialModelPrices),
    [keyForm.allowedModels, models, officialModelPrices]
  )
  const officialModelPriceByID = useMemo(
    () => mapPricesByModelID(officialModelPrices),
    [officialModelPrices]
  )
  const selectedKey = useMemo(() => {
    if (selectedKeyIds.length !== 1) return null
    return (
      keys.find((item) => String(item.id) === String(selectedKeyIds[0])) || null
    )
  }, [keys, selectedKeyIds])
  const selectedKeyText = selectedKey
    ? selectedKey.name || selectedKey.key_prefix || `凭据 ${selectedKey.id}`
    : selectedKeyIds.length > 1
      ? `已选 ${selectedKeyIds.length} 个凭据`
      : '请先单击或勾选一条凭据'
  const filteredKeys = useMemo(
    () =>
      keys.filter((item) => {
        const modelMatched = keyModelFilter
          ? Array.isArray(item.allowed_models) &&
            item.allowed_models.includes(keyModelFilter)
          : true
        const statusMatched =
          keyStatusFilter === 'enabled'
            ? !item.disabled
            : keyStatusFilter === 'disabled'
              ? !!item.disabled
              : true
        return modelMatched && statusMatched
      }),
    [keyModelFilter, keyStatusFilter, keys]
  )
  const hasActiveKeyFilters = Boolean(
    keySearch || keySearchInput || keyModelFilter || keyStatusFilter
  )
  const keyTokenStatsRows = useMemo(
    () => mergeKeyTokenStats(filteredKeys, keyTokenStatsByWindow),
    [filteredKeys, keyTokenStatsByWindow]
  )
  const paginatedKeys = useMemo(
    () => paginateItems(filteredKeys, keyPagination),
    [filteredKeys, keyPagination]
  )
  const paginatedKeyIds = useMemo(
    () => paginatedKeys.map((item) => item.id),
    [paginatedKeys]
  )
  const selectedPaginatedKeyCount = useMemo(
    () => paginatedKeyIds.filter((id) => selectedKeyIdSet.has(id)).length,
    [paginatedKeyIds, selectedKeyIdSet]
  )
  const isPaginatedKeysAllSelected =
    paginatedKeyIds.length > 0 &&
    selectedPaginatedKeyCount === paginatedKeyIds.length
  const isPaginatedKeysPartiallySelected =
    selectedPaginatedKeyCount > 0 &&
    selectedPaginatedKeyCount < paginatedKeyIds.length
  const paginatedKeyTokenStatsRows = useMemo(
    () => paginateItems(keyTokenStatsRows, keyStatsPagination),
    [keyStatsPagination, keyTokenStatsRows]
  )
  const paginatedModels = useMemo(
    () => paginateItems(models, modelPagination),
    [modelPagination, models]
  )
  const dailyUsageRows = useMemo(
    () => [...usageBuckets].reverse(),
    [usageBuckets]
  )
  const paginatedDailyUsageRows = useMemo(
    () => paginateItems(dailyUsageRows, dailyUsagePagination),
    [dailyUsagePagination, dailyUsageRows]
  )

  const setKeyListState = (res) => {
    const items = Array.isArray(res?.data?.items) ? res.data.items : []
    setKeys(items)
    setKeyTotal(asInt(res?.data?.total, items.length))
    const itemIds = new Set(items.map((item) => item.id))
    setSelectedKeyIds((current) => current.filter((id) => itemIds.has(id)))
  }

  const setModelListState = (res) => {
    const items = normalizeCodexModels(res?.data?.items)
    setModels(items)
    setModelTotal(items.length)
  }

  const setOfficialModelPriceState = (res) => {
    setOfficialModelPrices(normalizeCodexModelPrices(res?.data?.items))
  }

  const setUsageListState = (res) => {
    setUsageItems(Array.isArray(res?.data?.items) ? res.data.items : [])
    setUsageTotal(asInt(res?.data?.total, 0))
    if (res?.data?.summary) {
      setSummary(res.data.summary)
    }
  }

  const setUsageBucketsState = (res) => {
    setUsageBuckets(Array.isArray(res?.data?.items) ? res.data.items : [])
  }

  const setUsageSessionState = (res) => {
    setUsageSessionItems(Array.isArray(res?.data?.items) ? res.data.items : [])
    setUsageSessionTotal(asInt(res?.data?.total, 0))
  }

  const setGatewayUpstreamState = (res) => {
    const nextStrategy = res?.data?.strategy
    if (
      CODEX_UPSTREAM_STRATEGY_OPTIONS.some(
        (option) => option.value === nextStrategy
      )
    ) {
      setGatewayUpstreamStrategy(nextStrategy)
    }
    const nextDefaultReasoningEffort = String(
      res?.data?.default_reasoning_effort || ''
    )
    if (
      DEFAULT_REASONING_EFFORT_OPTIONS.some(
        (option) => option.value === nextDefaultReasoningEffort
      )
    ) {
      setGatewayDefaultReasoningEffort(nextDefaultReasoningEffort)
    }
  }

  const buildUsageWindowParams = (filters, now, timeWindowOverride) => {
    const timeWindow =
      timeWindowOverride || getUsageTimeWindow(filters.timeRange, now)
    const params = {
      end_time: timeWindow.endTime,
      start_time: timeWindow.startTime,
    }
    const keyIds = normalizeSelectedValues(filters.keyIds)
      .map((item) => asInt(item, 0))
      .filter((item) => item > 0)
    if (keyIds.length > 0) params.key_ids = keyIds
    if (filters.model) params.model = filters.model
    if (filters.reasoningEffort) {
      params.reasoning_effort = filters.reasoningEffort
    }
    if (filters.success) params.success = filters.success === 'true'
    if (filters.statusCode) params.status_code = asInt(filters.statusCode, 0)
    if (filters.upstreamMode) params.upstream_mode = filters.upstreamMode
    if (filters.clientType) params.client_type = filters.clientType
    if (filters.errorType) {
      params.error_type = filters.errorType
    }
    if (filters.excludeErrorType) {
      params.exclude_error_type = filters.excludeErrorType
    }
    return params
  }

  const buildUsageParams = (
    filters = appliedUsageFilters,
    pagination = usagePagination,
    now = Math.floor(Date.now() / 1000)
  ) => ({
    ...buildUsageWindowParams(filters, now),
    limit: pagination.pageSize || PAGE_SIZE,
    offset: getPaginationOffset(pagination),
  })

  const loadAll = async ({
    usageFilterOverride,
    usagePaginationOverride,
    usageTabOverride,
  } = {}) => {
    setLoading(true)
    setErrMsg('')
    try {
      const now = Math.floor(Date.now() / 1000)
      const startTime = now - DAY_SECONDS
      const activeUsageFilters = usageFilterOverride || appliedUsageFilters
      const activeUsagePagination = usagePaginationOverride || usagePagination
      const activeUsageTab = usageTabOverride || usageTab
      const effectiveUsageFilters = usageFiltersForTab(
        activeUsageFilters,
        activeUsageTab
      )

      if (currentView === 'dashboard') {
        const [summaryRes, keysRes, modelsRes, usageRes] = await Promise.all([
          apiRpc.call('summary', { start_time: startTime }),
          apiRpc.call('key_list', { limit: MAX_TABLE_FETCH_SIZE, offset: 0 }),
          apiRpc.call('model_list', { limit: MAX_TABLE_FETCH_SIZE, offset: 0 }),
          apiRpc.call('usage_list', {
            limit: DASHBOARD_USAGE_SIZE,
            offset: 0,
            start_time: startTime,
          }),
        ])
        setSummary(summaryRes?.data?.summary || {})
        setKeyListState(keysRes)
        setModelListState(modelsRes)
        setUsageListState(usageRes)
        return
      }

      if (currentView === 'keys') {
        const [keysRes, modelsRes] = await Promise.all([
          apiRpc.call('key_list', {
            limit: MAX_TABLE_FETCH_SIZE,
            offset: 0,
            search: keySearch,
          }),
          apiRpc.call('model_list', { limit: MAX_TABLE_FETCH_SIZE, offset: 0 }),
        ])
        setKeyListState(keysRes)
        setModelListState(modelsRes)
        return
      }

      if (currentView === 'analytics') {
        const [keysRes, modelsRes, ...keyTokenStatsResults] = await Promise.all(
          [
            apiRpc.call('key_list', {
              limit: MAX_TABLE_FETCH_SIZE,
              offset: 0,
              search: keySearch,
            }),
            apiRpc.call('model_list', {
              limit: MAX_TABLE_FETCH_SIZE,
              offset: 0,
            }),
            ...KEY_TOKEN_WINDOWS.map((windowItem) => {
              const window = getKeyTokenTimeWindow(windowItem, now)
              return apiRpc.call('usage_key_summaries', {
                end_time: window.endTime,
                limit: MAX_TABLE_FETCH_SIZE,
                offset: 0,
                start_time: window.startTime,
              })
            }),
          ]
        )
        setKeyListState(keysRes)
        setModelListState(modelsRes)
        setKeyTokenStatsByWindow(
          Object.fromEntries(
            KEY_TOKEN_WINDOWS.map((windowItem, index) => [
              windowItem.key,
              mapStatsByKeyID(keyTokenStatsResults[index]?.data?.items),
            ])
          )
        )
        return
      }

      if (currentView === 'models') {
        const [modelsRes, officialPricesRes] = await Promise.all([
          apiRpc.call('model_list', {
            limit: MAX_TABLE_FETCH_SIZE,
            offset: 0,
          }),
          apiRpc.call('official_model_price_list', {}),
        ])
        setModelListState(modelsRes)
        setOfficialModelPriceState(officialPricesRes)
        return
      }

      if (currentView === 'upstream') {
        const upstreamRes = await apiRpc.call('gateway_upstream_get', {})
        setGatewayUpstreamState(upstreamRes)
        return
      }

      const [
        upstreamRes,
        keysRes,
        modelsRes,
        usageRes,
        bucketsRes,
        sessionRes,
        ...keyTokenStatsResults
      ] = await Promise.all([
        apiRpc.call('gateway_upstream_get', {}),
        apiRpc.call('key_list', { limit: MAX_TABLE_FETCH_SIZE, offset: 0 }),
        apiRpc.call('model_list', { limit: MAX_TABLE_FETCH_SIZE, offset: 0 }),
        apiRpc.call(
          'usage_list',
          buildUsageParams(effectiveUsageFilters, activeUsagePagination, now)
        ),
        apiRpc.call('usage_buckets', {
          ...buildUsageWindowParams(effectiveUsageFilters, now),
          group_by: 'day_model',
        }),
        apiRpc.call(
          'usage_session_summaries',
          buildUsageParams(effectiveUsageFilters, activeUsagePagination, now)
        ),
        ...KEY_TOKEN_WINDOWS.map((windowItem) =>
          apiRpc.call('usage_key_summaries', {
            ...buildUsageWindowParams(
              effectiveUsageFilters,
              now,
              getKeyTokenTimeWindow(windowItem, now)
            ),
            limit: MAX_TABLE_FETCH_SIZE,
            offset: 0,
          })
        ),
      ])
      setGatewayUpstreamState(upstreamRes)
      setKeyListState(keysRes)
      setModelListState(modelsRes)
      setUsageListState(usageRes)
      setUsageBucketsState(bucketsRes)
      setUsageSessionState(sessionRes)
      setKeyTokenStatsByWindow(
        Object.fromEntries(
          KEY_TOKEN_WINDOWS.map((windowItem, index) => [
            windowItem.key,
            mapStatsByKeyID(keyTokenStatsResults[index]?.data?.items),
          ])
        )
      )
    } catch (e) {
      setErrMsg(getActionErrorMessage(e, '加载 API 数据'))
    } finally {
      setLoading(false)
    }
  }

  const changeGatewayUpstreamStrategy = async (strategy) => {
    if (strategy === gatewayUpstreamStrategy || gatewayUpstreamSaving) return
    setErrMsg('')
    setGatewayUpstreamSaving(true)
    try {
      const res = await apiRpc.call('gateway_upstream_set', {
        strategy,
        default_reasoning_effort: gatewayDefaultReasoningEffort,
      })
      setGatewayUpstreamState(res)
      await loadAll()
    } catch (e) {
      setErrMsg(getActionErrorMessage(e, '切换 Codex 上游策略'))
    } finally {
      setGatewayUpstreamSaving(false)
    }
  }

  const changeGatewayDefaultReasoningEffort = async (
    defaultReasoningEffort
  ) => {
    if (
      defaultReasoningEffort === gatewayDefaultReasoningEffort ||
      gatewayUpstreamSaving
    ) {
      return
    }
    setErrMsg('')
    setGatewayUpstreamSaving(true)
    try {
      const res = await apiRpc.call('gateway_upstream_set', {
        strategy: gatewayUpstreamStrategy,
        default_reasoning_effort: defaultReasoningEffort,
      })
      setGatewayUpstreamState(res)
      await loadAll()
    } catch (e) {
      setErrMsg(getActionErrorMessage(e, '切换默认推理档位'))
    } finally {
      setGatewayUpstreamSaving(false)
    }
  }

  useEffect(() => {
    loadAll()
  }, [
    currentView,
    keySearch,
    usagePagination.current,
    usagePagination.pageSize,
  ])

  useEffect(() => {
    setKeyPagination((current) => ({ ...current, current: 1 }))
    setKeyStatsPagination((current) => ({ ...current, current: 1 }))
  }, [keySearch, keyModelFilter, keyStatusFilter])

  useEffect(() => {
    setKeyPagination((current) => {
      const next = clampPagination(current, filteredKeys.length)
      return next.current === current.current &&
        next.pageSize === current.pageSize
        ? current
        : next
    })
  }, [filteredKeys.length])

  useEffect(() => {
    setKeyStatsPagination((current) => {
      const next = clampPagination(current, keyTokenStatsRows.length)
      return next.current === current.current &&
        next.pageSize === current.pageSize
        ? current
        : next
    })
  }, [keyTokenStatsRows.length])

  useEffect(() => {
    setModelPagination((current) => {
      const next = clampPagination(current, models.length)
      return next.current === current.current &&
        next.pageSize === current.pageSize
        ? current
        : next
    })
  }, [models.length])

  useEffect(() => {
    setUsagePagination((current) => {
      const next = clampPagination(current, usageTotal)
      return next.current === current.current &&
        next.pageSize === current.pageSize
        ? current
        : next
    })
  }, [usageTotal])

  useEffect(() => {
    setDailyUsagePagination((current) => {
      const next = clampPagination(current, dailyUsageRows.length)
      return next.current === current.current &&
        next.pageSize === current.pageSize
        ? current
        : next
    })
  }, [dailyUsageRows.length])

  useEffect(() => {
    if (pageKeySelectionRef.current) {
      pageKeySelectionRef.current.indeterminate =
        isPaginatedKeysPartiallySelected
    }
  }, [isPaginatedKeysPartiallySelected])

  useEffect(() => {
    const nextSearch = keySearchInput.trim()
    const timer = window.setTimeout(() => {
      setKeySearch((current) => (current === nextSearch ? current : nextSearch))
    }, 300)
    return () => window.clearTimeout(timer)
  }, [keySearchInput])

  const saveKey = async (e) => {
    e.preventDefault()
    setErrMsg('')
    setNewKey(null)
    setNewKeyBatch([])
    try {
      if (editingKeyId) {
        const currentKey = keys.find((item) => item.id === editingKeyId)
        await apiRpc.call('key_update', {
          key_id: editingKeyId,
          name: keyForm.remark.trim(),
          quota_daily_tokens: tokenLimitInputToTokens(keyForm.dailyTokenLimit),
          quota_weekly_tokens: tokenLimitInputToTokens(
            keyForm.weeklyTokenLimit
          ),
          quota_daily_input_tokens: tokenLimitInputToTokens(
            keyForm.dailyInputTokenLimit
          ),
          quota_weekly_input_tokens: tokenLimitInputToTokens(
            keyForm.weeklyInputTokenLimit
          ),
          quota_daily_output_tokens: tokenLimitInputToTokens(
            keyForm.dailyOutputTokenLimit
          ),
          quota_weekly_output_tokens: tokenLimitInputToTokens(
            keyForm.weeklyOutputTokenLimit
          ),
          quota_daily_billable_input_tokens: tokenLimitInputToTokens(
            keyForm.dailyBillableInputTokenLimit
          ),
          quota_weekly_billable_input_tokens: tokenLimitInputToTokens(
            keyForm.weeklyBillableInputTokenLimit
          ),
          allowed_models: splitModels(keyForm.allowedModels),
          upstream_strategy: keyForm.upstreamStrategy,
          default_reasoning_effort: keyForm.defaultReasoningEffort,
          disabled: !!currentKey?.disabled,
        })
      } else {
        const result = await apiRpc.call('key_create', {
          name: keyForm.remark.trim(),
          quota_daily_tokens: tokenLimitInputToTokens(keyForm.dailyTokenLimit),
          quota_weekly_tokens: tokenLimitInputToTokens(
            keyForm.weeklyTokenLimit
          ),
          quota_daily_input_tokens: tokenLimitInputToTokens(
            keyForm.dailyInputTokenLimit
          ),
          quota_weekly_input_tokens: tokenLimitInputToTokens(
            keyForm.weeklyInputTokenLimit
          ),
          quota_daily_output_tokens: tokenLimitInputToTokens(
            keyForm.dailyOutputTokenLimit
          ),
          quota_weekly_output_tokens: tokenLimitInputToTokens(
            keyForm.weeklyOutputTokenLimit
          ),
          quota_daily_billable_input_tokens: tokenLimitInputToTokens(
            keyForm.dailyBillableInputTokenLimit
          ),
          quota_weekly_billable_input_tokens: tokenLimitInputToTokens(
            keyForm.weeklyBillableInputTokenLimit
          ),
          allowed_models: splitModels(keyForm.allowedModels),
          upstream_strategy: keyForm.upstreamStrategy,
          default_reasoning_effort: keyForm.defaultReasoningEffort,
        })
        setNewKey(result?.data || null)
      }
      setKeyForm(INITIAL_KEY_FORM)
      setEditingKeyId(null)
      setKeyModalOpen(false)
      await loadAll()
    } catch (err) {
      setErrMsg(
        getActionErrorMessage(
          err,
          editingKeyId ? '更新 API key' : '创建 API key'
        )
      )
    }
  }

  const openCreateKey = () => {
    setNewKey(null)
    setNewKeyBatch([])
    setEditingKeyId(null)
    setKeyForm(INITIAL_KEY_FORM)
    setKeyModalOpen(true)
  }

  const startEditKey = (item) => {
    const allowedModel =
      Array.isArray(item.allowed_models) && item.allowed_models.length > 0
        ? item.allowed_models[0]
        : ''
    setNewKey(null)
    setNewKeyBatch([])
    setEditingKeyId(item.id)
    setKeyForm({
      remark: item.name || '',
      allowedModels:
        allowedModel && officialModelPriceByID.has(allowedModel)
          ? allowedModel
          : '',
      upstreamStrategy: item.upstream_strategy || '',
      defaultReasoningEffort: item.default_reasoning_effort || '',
      dailyTokenLimit: tokenLimitTokensToInput(item.quota_daily_tokens),
      weeklyTokenLimit: tokenLimitTokensToInput(item.quota_weekly_tokens),
      dailyInputTokenLimit: tokenLimitTokensToInput(
        item.quota_daily_input_tokens
      ),
      weeklyInputTokenLimit: tokenLimitTokensToInput(
        item.quota_weekly_input_tokens
      ),
      dailyOutputTokenLimit: tokenLimitTokensToInput(
        item.quota_daily_output_tokens
      ),
      weeklyOutputTokenLimit: tokenLimitTokensToInput(
        item.quota_weekly_output_tokens
      ),
      dailyBillableInputTokenLimit: tokenLimitTokensToInput(
        item.quota_daily_billable_input_tokens
      ),
      weeklyBillableInputTokenLimit: tokenLimitTokensToInput(
        item.quota_weekly_billable_input_tokens
      ),
    })
    setKeyModalOpen(true)
  }

  const cancelEditKey = () => {
    setEditingKeyId(null)
    setKeyForm(INITIAL_KEY_FORM)
    setKeyModalOpen(false)
  }

  const openConfirmDialog = (config) => {
    setErrMsg('')
    setConfirmDialog({
      cancelText: '取消',
      tone: 'danger',
      ...config,
    })
  }

  const closeConfirmDialog = () => {
    if (confirmSubmitting || resettingKey) return
    setConfirmDialog(null)
  }

  const submitConfirmDialog = async () => {
    if (!confirmDialog?.onConfirm || confirmSubmitting) return
    setConfirmSubmitting(true)
    try {
      await confirmDialog.onConfirm()
      setConfirmDialog(null)
    } finally {
      setConfirmSubmitting(false)
    }
  }

  const performResetKeySecret = async (keyId) => {
    setErrMsg('')
    setNewKey(null)
    setNewKeyBatch([])
    setResettingKey(true)
    try {
      const result = await apiRpc.call('key_reset_secret', {
        key_id: keyId,
      })
      setNewKey(result?.data || null)
      setNewKeyBatch([])
      setEditingKeyId(null)
      setKeyForm(INITIAL_KEY_FORM)
      setKeyModalOpen(false)
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '重置 API key'))
    } finally {
      setResettingKey(false)
    }
  }

  const resetKeySecret = () => {
    if (!editingKeyId || resettingKey) return
    const keyId = editingKeyId
    const currentKey = keys.find((item) => item.id === keyId)
    const label = currentKey?.name || currentKey?.key_prefix || `ID ${keyId}`
    openConfirmDialog({
      title: '重置 API key',
      description: `确认重置 API 凭据「${label}」吗？`,
      detail: '旧 key 会立即失效，需要把新生成的完整 key 同步到客户端。',
      confirmText: '重置 API key',
      pendingText: '重置中...',
      onConfirm: () => performResetKeySecret(keyId),
    })
  }

  const performResetSelectedKeys = async (keyIds) => {
    setErrMsg('')
    setNewKey(null)
    setNewKeyBatch([])
    setResettingKey(true)

    const generated = []
    try {
      for (const keyId of keyIds) {
        const result = await apiRpc.call('key_reset_secret', {
          key_id: keyId,
        })
        if (result?.data?.plain_key) {
          generated.push(result.data)
        }
      }
      setNewKeyBatch(generated)
      setSelectedKeyIds([])
      if (keyIds.includes(editingKeyId)) {
        cancelEditKey()
      }
      await loadAll()
    } catch (err) {
      if (generated.length > 0) {
        setNewKeyBatch(generated)
      }
      setErrMsg(getActionErrorMessage(err, '批量重置 API key'))
      await loadAll()
    } finally {
      setResettingKey(false)
    }
  }

  const resetSelectedKeys = () => {
    if (selectedKeyIds.length === 0 || resettingKey) return
    const keyIds = [...selectedKeyIds]
    const count = keyIds.length
    openConfirmDialog({
      title: '批量重置 API key',
      description: `确认重置选中的 ${count} 个 API 凭据吗？`,
      detail: '旧 key 会立即失效，需要把新生成的完整 key 同步到对应客户端。',
      confirmText: `重置 ${count} 个 key`,
      pendingText: '重置中...',
      onConfirm: () => performResetSelectedKeys(keyIds),
    })
  }

  const handleKeySearchInputChange = (e) => {
    setErrMsg('')
    setSelectedKeyIds([])
    setKeySearchInput(e.target.value)
  }

  const clearKeySearch = () => {
    setErrMsg('')
    setSelectedKeyIds([])
    setKeySearchInput('')
    setKeySearch('')
    setKeyModelFilter('')
    setKeyStatusFilter('')
  }

  const performDeleteKey = async (item) => {
    const keyId = item?.id
    setErrMsg('')
    setNewKeyBatch([])
    try {
      await apiRpc.call('key_delete', { key_id: keyId })
      if (editingKeyId === keyId) cancelEditKey()
      setSelectedKeyIds((current) => current.filter((id) => id !== keyId))
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '删除 API key'))
    }
  }

  const deleteKey = (item) => {
    const keyId = item?.id
    const label = item?.name || item?.key_prefix || `ID ${keyId}`
    openConfirmDialog({
      title: '删除 API 凭据',
      description: `确认删除 API 凭据「${label}」吗？`,
      detail: '删除后不可恢复，历史调用记录会保留。',
      confirmText: '删除',
      pendingText: '删除中...',
      onConfirm: () => performDeleteKey(item),
    })
  }

  const performDeleteSelectedKeys = async (keyIds) => {
    setErrMsg('')
    setNewKey(null)
    setNewKeyBatch([])
    try {
      await apiRpc.call('key_delete_batch', { key_ids: keyIds })
      if (keyIds.includes(editingKeyId)) cancelEditKey()
      setSelectedKeyIds([])
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '批量删除 API key'))
    }
  }

  const deleteSelectedKeys = () => {
    if (selectedKeyIds.length === 0) return
    const keyIds = [...selectedKeyIds]
    openConfirmDialog({
      title: '批量删除 API 凭据',
      description: `确认删除选中的 ${keyIds.length} 个 API 凭据吗？`,
      detail: '删除后不可恢复，历史调用记录会保留。',
      confirmText: `删除 ${keyIds.length} 个凭据`,
      pendingText: '删除中...',
      onConfirm: () => performDeleteSelectedKeys(keyIds),
    })
  }

  const performDisableAllKeys = async () => {
    setErrMsg('')
    setNewKey(null)
    setNewKeyBatch([])
    try {
      await apiRpc.call('key_disable_all')
      setSelectedKeyIds([])
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '禁用全部 API key'))
    }
  }

  const disableAllKeys = () => {
    if (keyTotal === 0) return
    const activeCount = keys.filter((item) => !item.disabled).length
    openConfirmDialog({
      title: '禁用全部 API key',
      description: '确认禁用全站所有 API 凭据吗？',
      detail: `这会立即拦截所有下游客户端调用，不限于当前页或当前筛选；历史调用记录和 key 本身会保留。当前已加载列表里有 ${activeCount} 个启用凭据。`,
      confirmText: '禁用全部 key',
      pendingText: '禁用中...',
      onConfirm: performDisableAllKeys,
    })
  }

  const performEnableAllKeys = async () => {
    setErrMsg('')
    setNewKey(null)
    setNewKeyBatch([])
    try {
      await apiRpc.call('key_enable_all')
      setSelectedKeyIds([])
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '启用全部 API key'))
    }
  }

  const enableAllKeys = () => {
    if (keyTotal === 0) return
    const disabledCount = keys.filter((item) => item.disabled).length
    openConfirmDialog({
      title: '启用全部 API key',
      description: '确认启用全站所有 API 凭据吗？',
      detail: `这会立即恢复所有下游客户端调用，不限于当前页或当前筛选；历史调用记录和 key 本身会保留。当前已加载列表里有 ${disabledCount} 个禁用凭据。`,
      confirmText: '启用全部 key',
      pendingText: '启用中...',
      tone: 'primary',
      onConfirm: performEnableAllKeys,
    })
  }

  const selectKeyRow = (keyId) => {
    setSelectedKeyIds((current) => getTableSelectionAfterClick(current, keyId))
  }

  const toggleKeySelection = (keyId, checked) => {
    setSelectedKeyIds((current) =>
      toggleTableSelection(current, keyId, checked)
    )
  }

  const toggleCurrentPageKeySelection = (checked) => {
    setSelectedKeyIds((current) =>
      toggleTablePageSelection(current, paginatedKeyIds, checked)
    )
  }

  const handleKeyRowClick = (event, keyId) => {
    if (isInteractiveTableTarget(event.target)) {
      return
    }
    selectKeyRow(keyId)
  }

  const handleKeyRowDoubleClick = (event, item) => {
    if (isInteractiveTableTarget(event.target)) {
      return
    }
    startEditKey(item)
  }

  const setKeyDisabled = async (keyId, disabled) => {
    setErrMsg('')
    try {
      await apiRpc.call('key_set_disabled', {
        key_id: keyId,
        disabled,
      })
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '更新 API key'))
    }
  }

  const setModelEnabled = async (modelId, enabled) => {
    setErrMsg('')
    try {
      await apiRpc.call('model_set_enabled', {
        id: modelId,
        enabled,
      })
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '更新模型状态'))
    }
  }

  const openModelContextModal = (item) => {
    setErrMsg('')
    setEditingModelContext(item)
    setModelContextForm(modelContextFormFromItem(item))
    setModelContextModalOpen(true)
  }

  const saveModelContext = async (e) => {
    e.preventDefault()
    if (!editingModelContext?.id) return
    const payload = {
      context_window_tokens: limitInputToNumber(
        modelContextForm.contextWindowTokens
      ),
      context_compact_tokens: limitInputToNumber(
        modelContextForm.contextCompactTokens
      ),
      context_hard_tokens: limitInputToNumber(
        modelContextForm.contextHardTokens
      ),
      context_compact_bytes: limitInputToNumber(
        modelContextForm.contextCompactBytes
      ),
      context_hard_bytes: limitInputToNumber(modelContextForm.contextHardBytes),
      context_keep_items: limitInputToNumber(
        modelContextForm.contextKeepItems,
        {
          allowUnit: false,
        }
      ),
    }
    if (Object.values(payload).some((value) => value == null)) {
      setErrMsg(
        '上下文策略只支持数字，阈值可使用 K / M 单位，保留条数只能填整数。'
      )
      return
    }
    setErrMsg('')
    try {
      await apiRpc.call('model_context_update', {
        id: editingModelContext.id,
        ...payload,
      })
      setModelContextModalOpen(false)
      setEditingModelContext(null)
      setModelContextForm(INITIAL_MODEL_CONTEXT_FORM)
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '更新模型上下文策略'))
    }
  }

  const switchUsageTab = (nextTab) => {
    if (nextTab === usageTab) return
    const nextFilters = applyUsageTabDefaultTimeRange(
      appliedUsageFilters,
      nextTab,
      usageTimeRangeTouched
    )
    const nextPagination = {
      ...usagePagination,
      current: 1,
    }
    setUsageTab(nextTab)
    setUsageFilters(nextFilters)
    setAppliedUsageFilters(nextFilters)
    setUsagePagination(nextPagination)
    setDailyUsagePagination((current) => ({ ...current, current: 1 }))
    loadAll({
      usageFilterOverride: nextFilters,
      usagePaginationOverride: nextPagination,
      usageTabOverride: nextTab,
    })
  }

  const openUsageSessionDetail = async (sessionItem) => {
    if (!sessionItem?.session_id) return
    setSelectedUsageSession(sessionItem)
    setSelectedUsageSessionItems([])
    setUsageSessionDetailLoading(true)
    try {
      const now = Math.floor(Date.now() / 1000)
      const effectiveUsageFilters = usageFiltersForTab(
        appliedUsageFilters,
        usageTab
      )
      const res = await apiRpc.call('usage_list', {
        ...buildUsageWindowParams(effectiveUsageFilters, now),
        limit: 100,
        offset: 0,
        session_id: sessionItem.session_id,
      })
      setSelectedUsageSessionItems(
        Array.isArray(res?.data?.items) ? res.data.items : []
      )
    } catch (e) {
      setErrMsg(getActionErrorMessage(e, '加载会话详情'))
    } finally {
      setUsageSessionDetailLoading(false)
    }
  }

  const loadUsageBucketDetail = async (bucketItem, pagination) => {
    if (!bucketItem?.bucket_start || !bucketItem?.model) return
    setUsageBucketDetailLoading(true)
    try {
      const startTime = asInt(bucketItem.bucket_start, 0)
      const res = await apiRpc.call('usage_list', {
        ...buildUsageParams(appliedUsageFilters, pagination, 0),
        end_time: startTime + DAY_SECONDS,
        limit: pagination.pageSize,
        model: bucketItem.model,
        offset: (pagination.current - 1) * pagination.pageSize,
        start_time: startTime,
      })
      const items = Array.isArray(res?.data?.items) ? res.data.items : []
      setSelectedUsageBucketItems(items)
      setSelectedUsageBucketTotal(asInt(res?.data?.total, items.length))
    } catch (e) {
      setErrMsg(getActionErrorMessage(e, '加载每日模型详情'))
    } finally {
      setUsageBucketDetailLoading(false)
    }
  }

  const openUsageBucketDetail = async (bucketItem) => {
    if (!bucketItem?.bucket_start || !bucketItem?.model) return
    const nextPagination = createInitialPagination()
    setSelectedUsageBucket(bucketItem)
    setSelectedUsageBucketItems([])
    setSelectedUsageBucketTotal(0)
    setUsageBucketDetailPagination(nextPagination)
    await loadUsageBucketDetail(bucketItem, nextPagination)
  }

  const changeUsageBucketDetailPagination = async (nextPagination) => {
    if (!selectedUsageBucket) return
    const safePagination = clampPagination(
      nextPagination,
      selectedUsageBucketTotal
    )
    setUsageBucketDetailPagination(safePagination)
    await loadUsageBucketDetail(selectedUsageBucket, safePagination)
  }

  const renderUsageTable = (compact = false) => (
    <div className={tableWrapClass}>
      <div className="overflow-auto">
        <table
          className={`${tableClass} ${compact ? 'min-w-[900px]' : 'min-w-[2280px]'}`}
        >
          <thead>
            <tr>
              <th className={thClass}>时间</th>
              {!compact ? <th className={thClass}>请求</th> : null}
              <th className={thClass}>凭据</th>
              <th className={thClass}>接口</th>
              <th className={thClass}>模型</th>
              {!compact ? <th className={thClass}>Effort</th> : null}
              {!compact ? <th className={thClass}>上游</th> : null}
              {!compact ? <th className={thClass}>客户端</th> : null}
              <th className={thClass}>状态</th>
              <th className={thClass}>
                <HeaderWithHelp help="总 Token = 输入 Token + 输出 Token；非缓存输入 = 输入 Token - 缓存输入。">
                  Token
                </HeaderWithHelp>
              </th>
              {!compact ? (
                <th className={thClass}>
                  <HeaderWithHelp help="缓存输入是命中上下文缓存的输入 Token；推理输出是模型内部 reasoning 输出 Token。">
                    缓存输入 / 推理输出
                  </HeaderWithHelp>
                </th>
              ) : null}
              {!compact ? (
                <th className={thClass}>
                  <HeaderWithHelp help="按当前模型价格口径估算；未配置价格时显示未配置。">
                    费用估算
                  </HeaderWithHelp>
                </th>
              ) : null}
              {!compact ? (
                <th className={thClass}>
                  <HeaderWithHelp help="网关从收到请求到返回响应的耗时，单位毫秒。">
                    耗时
                  </HeaderWithHelp>
                </th>
              ) : null}
              {!compact ? (
                <th className={thClass}>
                  <HeaderWithHelp help="请求字节 / 响应字节，用于判断单次调用的数据体大小。">
                    字节
                  </HeaderWithHelp>
                </th>
              ) : null}
              {!compact ? (
                <th className={thClass}>
                  <HeaderWithHelp help={GATEWAY_ERROR_TYPE_HELP}>
                    错误
                  </HeaderWithHelp>
                </th>
              ) : null}
              {!compact ? (
                <th className={thClass}>
                  <HeaderWithHelp help="诊断字段只保存请求 / 响应大小、fallback 状态、backend-only 标记和脱敏上游摘要，不保存 prompt 或模型输出正文。">
                    诊断
                  </HeaderWithHelp>
                </th>
              ) : null}
            </tr>
          </thead>
          <tbody className="divide-y divide-[#e7efe9] bg-white">
            {usageItems.length > 0 ? (
              usageItems.map((item) => (
                <tr key={String(item.id)} className="align-top">
                  <td className={tdClass}>{fmtTs(item.created_at)}</td>
                  {!compact ? (
                    <td className={`${tdClass} min-w-[220px]`}>
                      <div className="font-mono text-xs">
                        {item.request_id || '-'}
                      </div>
                      <div className="mt-1 break-all text-xs text-[#9aa39e]">
                        Session：{item.session_id || '未传入'}
                      </div>
                      <div className="mt-1 break-all text-xs text-[#9aa39e]">
                        IP：{item.client_ip || '未记录'}
                      </div>
                    </td>
                  ) : null}
                  <td className={tdClass}>
                    <ApiKeyUsageCell item={item} />
                  </td>
                  <td className={tdClass}>
                    {item.endpoint || item.path}
                    {!compact ? (
                      <div className="mt-1 text-xs text-[#9aa39e]">
                        {item.method || '-'}
                      </div>
                    ) : null}
                  </td>
                  <td className={`${tdClass} font-mono text-xs`}>
                    {item.model || '-'}
                  </td>
                  {!compact ? (
                    <td className={`${tdClass} whitespace-nowrap text-xs`}>
                      {reasoningEffortLabel(item.reasoning_effort)}
                    </td>
                  ) : null}
                  {!compact ? (
                    <td className={tdClass}>
                      <div className="whitespace-nowrap text-xs font-semibold">
                        {upstreamModeLabel(item.upstream_mode)}
                      </div>
                      <div className="mt-1 text-xs text-[#9aa39e]">
                        {item.upstream_fallback ? 'fallback' : 'direct'}
                      </div>
                    </td>
                  ) : null}
                  {!compact ? (
                    <td className={`${tdClass} whitespace-nowrap text-xs`}>
                      {clientTypeLabel(item.client_type)}
                    </td>
                  ) : null}
                  <td className={tdClass}>
                    <StatusBadge
                      active={!!item.success}
                      trueText={`HTTP ${item.status_code}`}
                      falseText={
                        isClientCanceledUsage(item)
                          ? `已取消 HTTP ${item.status_code}`
                          : `HTTP ${item.status_code}`
                      }
                      falseTone={
                        isClientCanceledUsage(item) ? 'neutral' : 'danger'
                      }
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
                  {!compact ? (
                    <td className={tdClass}>
                      <div className="text-xs leading-5">
                        缓存输入 {fmtNumber(item.cached_tokens)}
                      </div>
                      <div className="mt-1 text-xs leading-5 text-[#9aa39e]">
                        推理输出 {fmtNumber(item.reasoning_tokens)}
                      </div>
                    </td>
                  ) : null}
                  {!compact ? (
                    <td className={`${tdClass} whitespace-nowrap`}>
                      {fmtCost(item.estimated_cost_usd)}
                    </td>
                  ) : null}
                  {!compact ? (
                    <td className={tdClass}>
                      {fmtNumber(item.duration_ms)} ms
                    </td>
                  ) : null}
                  {!compact ? (
                    <td className={tdClass}>
                      <div className="text-xs leading-5">
                        请求 {fmtNumber(item.request_bytes)}
                      </div>
                      <div className="mt-1 text-xs leading-5 text-[#9aa39e]">
                        响应 {fmtNumber(item.response_bytes)}
                      </div>
                    </td>
                  ) : null}
                  {!compact ? (
                    <td className={tdClass}>
                      <ErrorTypeCell value={item.error_type} />
                    </td>
                  ) : null}
                  {!compact ? (
                    <td className={tdClass}>
                      <DiagnosticCell item={item} />
                    </td>
                  ) : null}
                </tr>
              ))
            ) : (
              <tr>
                <td
                  colSpan={compact ? 6 : 16}
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
  )

  const renderDashboard = () => {
    const activeKeys = keys.filter((item) => !item.disabled).length
    const enabledModels = models.filter((item) => item.enabled).length

    return (
      <div className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <SummaryCard
            label="24h 请求"
            value={fmtNumber(summary.total_requests)}
            sub={`${fmtNumber(summary.success_requests)} 成功 / ${fmtNumber(summary.failed_requests)} 服务错误 / ${fmtNumber(summary.client_canceled_requests)} 取消`}
          />
          <SummaryCard
            label="24h Token"
            value={fmtNumber(summary.total_tokens)}
            sub={`${fmtNumber(summary.input_tokens)} 输入（非缓存 ${fmtNumber(billableInputTokens(summary))}） / ${fmtNumber(summary.output_tokens)} 输出`}
          />
          <SummaryCard
            label="API 凭据"
            value={fmtNumber(keyTotal)}
            sub={`${fmtNumber(activeKeys)} 个启用`}
          />
          <SummaryCard
            label="模型"
            value={fmtNumber(modelTotal)}
            sub={`${fmtNumber(enabledModels)} 个启用`}
          />
        </div>

        <SurfacePanel variant="admin" className="p-5 sm:p-6">
          <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-[#1f2d25]">
                最近调用预览
              </h2>
              <div className="mt-1 text-sm text-[#7b8780]">
                24 小时内最近 {usageItems.length} 条 / 共 {usageTotal} 条。
              </div>
            </div>
          </div>
          {renderUsageTable(true)}
        </SurfacePanel>
      </div>
    )
  }

  const renderKeyTokenStats = ({
    title = '凭据 Token 统计',
    description = '按当前凭据列表汇总各时间窗口的总 Token；无调用显示 0。',
    showFilters = false,
  } = {}) => (
    <SurfacePanel variant="admin" className="p-5 sm:p-6">
      <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-[#1f2d25]">{title}</h2>
          <div className="mt-1 text-sm text-[#7b8780]">{description}</div>
        </div>
        <div className="text-sm text-[#7b8780]">
          {fmtNumber(keyTokenStatsRows.length)} 个凭据
        </div>
      </div>

      {showFilters ? (
        <div className={`${toolbarClass} mb-5`}>
          <div className="admin-module-toolbar-row">
            <div className={filterGroupClass}>
              <label className="sr-only" htmlFor="analytics-key-search">
                搜索凭据统计
              </label>
              <input
                id="analytics-key-search"
                value={keySearchInput}
                onChange={handleKeySearchInputChange}
                className={inputClass}
                placeholder="搜索备注、前缀或后四位"
              />
              <SearchableSelect
                value={keyModelFilter}
                onChange={(nextValue) => {
                  setSelectedKeyIds([])
                  setKeyModelFilter(nextValue)
                }}
                ariaLabel="按模型筛选"
                options={[
                  { label: '全部模型', value: '' },
                  ...modelOptions.map((modelId) => ({
                    label: modelId,
                    value: modelId,
                  })),
                ]}
                placeholder="输入模型筛选"
              />
              <SearchableSelect
                value={keyStatusFilter}
                onChange={(nextValue) => {
                  setSelectedKeyIds([])
                  setKeyStatusFilter(nextValue)
                }}
                ariaLabel="按状态筛选"
                options={KEY_STATUS_FILTER_OPTIONS}
                placeholder="输入状态筛选"
              />
            </div>
            <div className={primaryActionsClass}>
              {hasActiveKeyFilters ? (
                <button
                  type="button"
                  onClick={clearKeySearch}
                  disabled={loading}
                  className={secondaryButtonClass}
                >
                  重置
                </button>
              ) : null}
            </div>
          </div>
        </div>
      ) : null}

      <div className={tableWrapClass}>
        <div className="overflow-auto">
          <table className={`${tableClass} min-w-[1240px]`}>
            <thead>
              <tr>
                <th className={thClass}>备注</th>
                <th className={thClass}>状态</th>
                {KEY_TOKEN_WINDOWS.map((windowItem) => (
                  <th key={windowItem.key} className={thClass}>
                    {windowItem.label} Token
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-[#e7efe9] bg-white">
              {keyTokenStatsRows.length > 0 ? (
                paginatedKeyTokenStatsRows.map((item) => (
                  <tr key={String(item.id)}>
                    <td className={tdClass}>
                      <div className="max-w-[260px] truncate font-medium text-[#1f2d25]">
                        {item.name}
                      </div>
                      <div className="mt-1 font-mono text-xs text-[#7b8780]">
                        {item.prefix}
                      </div>
                    </td>
                    <td className={tdClass}>
                      <StatusBadge active={!item.disabled} />
                    </td>
                    {KEY_TOKEN_WINDOWS.map((windowItem) => (
                      <td
                        key={windowItem.key}
                        className={`${tdClass} whitespace-nowrap font-semibold`}
                      >
                        {fmtNumber(item.tokens[windowItem.key])}
                        <div className="mt-1 text-xs font-normal text-[#9aa39e]">
                          B{' '}
                          {fmtNumber(
                            item.upstream[windowItem.key]?.backend_requests
                          )}
                          <span className="mx-1 text-[#c0c9c4]">/</span>
                          CLI{' '}
                          {fmtNumber(
                            item.upstream[windowItem.key]?.cli_requests
                          )}
                        </div>
                      </td>
                    ))}
                  </tr>
                ))
              ) : (
                <tr>
                  <td
                    colSpan={KEY_TOKEN_WINDOWS.length + 2}
                    className="px-4 py-10 text-center text-sm text-[#9aa39e]"
                  >
                    {hasActiveKeyFilters
                      ? '没有匹配的 API 凭据统计'
                      : loading
                        ? '加载中...'
                        : '暂无 API 凭据统计'}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
      <TablePagination
        total={keyTokenStatsRows.length}
        pagination={keyStatsPagination}
        onChange={setKeyStatsPagination}
        disabled={loading}
      />
    </SurfacePanel>
  )

  const renderKeys = () => (
    <div className="space-y-5">
      <SurfacePanel variant="admin" className="p-5 sm:p-6">
        <div className="space-y-5">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h2 className="text-lg font-semibold text-[#1f2d25]">API 凭据</h2>
              <div className="mt-1 text-sm text-[#7b8780]">
                共 {keyTotal} 个；用于客户端调用本服务的 OpenAI 兼容接口。
              </div>
            </div>
          </div>

          <div className={toolbarClass}>
            <div className="admin-module-toolbar-row">
              <div className={filterGroupClass}>
                <label className="sr-only" htmlFor="key-search">
                  搜索凭据
                </label>
                <input
                  id="key-search"
                  value={keySearchInput}
                  onChange={handleKeySearchInputChange}
                  className={inputClass}
                  placeholder="搜索备注、前缀或后四位"
                />
                <SearchableSelect
                  value={keyModelFilter}
                  onChange={(nextValue) => {
                    setSelectedKeyIds([])
                    setKeyModelFilter(nextValue)
                  }}
                  ariaLabel="按模型筛选"
                  options={[
                    { label: '全部模型', value: '' },
                    ...modelOptions.map((modelId) => ({
                      label: modelId,
                      value: modelId,
                    })),
                  ]}
                  placeholder="输入模型筛选"
                />
                <SearchableSelect
                  value={keyStatusFilter}
                  onChange={(nextValue) => {
                    setSelectedKeyIds([])
                    setKeyStatusFilter(nextValue)
                  }}
                  ariaLabel="按状态筛选"
                  options={KEY_STATUS_FILTER_OPTIONS}
                  placeholder="输入状态筛选"
                />
              </div>
              <div className={primaryActionsClass}>
                {hasActiveKeyFilters ? (
                  <button
                    type="button"
                    onClick={clearKeySearch}
                    disabled={loading}
                    className={secondaryButtonClass}
                  >
                    重置
                  </button>
                ) : null}
                <button
                  type="button"
                  onClick={openCreateKey}
                  disabled={loading}
                  className={primaryButtonClass}
                >
                  新建 API 凭据
                </button>
                <button
                  type="button"
                  onClick={enableAllKeys}
                  disabled={loading || keyTotal === 0}
                  className={secondaryButtonClass}
                >
                  启用全部 key
                </button>
                <button
                  type="button"
                  onClick={disableAllKeys}
                  disabled={loading || keyTotal === 0}
                  className={dangerButtonClass}
                >
                  禁用全部 key
                </button>
              </div>
            </div>
            <div className={selectionRowClass}>
              <div className={selectionBlockClass}>
                <span className="font-semibold text-[#1f2d25]">当前操作</span>
                <span
                  className={`${selectionTagClass} ${
                    selectedKey ? 'admin-selection-tag-active' : ''
                  }`}
                >
                  {selectedKeyText}
                </span>
              </div>
              <div className={selectionActionsClass}>
                {selectedKeyIds.length > 0 ? (
                  <button
                    type="button"
                    onClick={() => setSelectedKeyIds([])}
                    className="admin-link-button"
                  >
                    清空已选
                  </button>
                ) : null}
                <button
                  type="button"
                  onClick={() => selectedKey && startEditKey(selectedKey)}
                  disabled={loading || !selectedKey}
                  className={tableActionButtonClass}
                >
                  编辑
                </button>
                <button
                  type="button"
                  onClick={() =>
                    selectedKey &&
                    setKeyDisabled(selectedKey.id, !selectedKey.disabled)
                  }
                  disabled={loading || !selectedKey}
                  className={tablePrimaryButtonClass}
                >
                  {selectedKey?.disabled ? '启用' : '禁用'}
                </button>
                <button
                  type="button"
                  onClick={resetSelectedKeys}
                  disabled={
                    loading || resettingKey || selectedKeyIds.length === 0
                  }
                  className={dangerButtonClass}
                >
                  {resettingKey ? '重置中...' : '重置 API key'}
                </button>
                <button
                  type="button"
                  onClick={deleteSelectedKeys}
                  disabled={
                    loading || resettingKey || selectedKeyIds.length === 0
                  }
                  className={dangerButtonClass}
                >
                  删除
                </button>
              </div>
            </div>
          </div>

          <div className={tableWrapClass}>
            <div className="overflow-auto">
              <table className={`${keyTableClass} min-w-[1500px]`}>
                <colgroup>
                  <col className="admin-key-table-selection-col" />
                  <col className="admin-key-table-remark-col" />
                  <col className="admin-key-table-date-col" />
                  <col className="admin-key-table-date-col" />
                  <col className="admin-key-table-key-col" />
                  <col className="admin-key-table-model-col" />
                  <col className="admin-key-table-upstream-col" />
                  <col className="admin-key-table-effort-col" />
                  <col className="admin-key-table-quota-col" />
                  <col className="admin-key-table-status-col" />
                </colgroup>
                <thead>
                  <tr>
                    <th className={selectionThClass}>
                      <input
                        ref={pageKeySelectionRef}
                        type="checkbox"
                        checked={isPaginatedKeysAllSelected}
                        onChange={(event) =>
                          toggleCurrentPageKeySelection(event.target.checked)
                        }
                        disabled={paginatedKeyIds.length === 0}
                        aria-label="选择当前页 API 凭据"
                        title="选择当前页"
                        className="admin-checkbox"
                      />
                    </th>
                    <th className={thClass}>备注</th>
                    <th className={thClass}>创建时间</th>
                    <th className={thClass}>更新时间</th>
                    <th className={thClass}>完整凭据</th>
                    <th className={thClass}>模型限制</th>
                    <th className={thClass}>上游策略</th>
                    <th className={thClass}>默认 Effort</th>
                    <th className={thClass}>Token 日 / 周限制（百万）</th>
                    <th className={thClass}>状态</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-[#e7efe9] bg-white">
                  {filteredKeys.length > 0 ? (
                    paginatedKeys.map((item) => {
                      const isSelected = selectedKeyIdSet.has(item.id)
                      return (
                        <tr
                          key={String(item.id)}
                          className={`admin-table-row align-top ${
                            isSelected ? 'admin-table-row-selected' : ''
                          }`}
                          onClick={(event) => handleKeyRowClick(event, item.id)}
                          onDoubleClick={(event) =>
                            handleKeyRowDoubleClick(event, item)
                          }
                          aria-selected={isSelected}
                          title={TABLE_ROW_INTERACTION_TITLE}
                        >
                          <td className={selectionTdClass}>
                            <input
                              type="checkbox"
                              checked={isSelected}
                              onChange={(e) =>
                                toggleKeySelection(item.id, e.target.checked)
                              }
                              onClick={(e) => e.stopPropagation()}
                              aria-label={`选择 ${item.name || item.key_prefix || item.id}`}
                              className="admin-checkbox"
                            />
                          </td>
                          <td className={`${tdClass} font-medium`}>
                            {item.name || '无备注'}
                            <div className="mt-1 text-xs text-[#9aa39e]">
                              最近使用：{fmtTs(item.last_used_at)}
                            </div>
                          </td>
                          <td
                            className={`${tdClass} whitespace-nowrap text-sm`}
                          >
                            {fmtTs(item.created_at)}
                          </td>
                          <td
                            className={`${tdClass} whitespace-nowrap text-sm`}
                          >
                            {fmtTs(item.updated_at)}
                          </td>
                          <td className={`${tdClass} font-mono text-xs`}>
                            <div className="admin-key-value-cell">
                              <span className="admin-key-value-text">
                                {apiKeyDisplayText(item)}
                              </span>
                              {apiKeyPlainText(item) ? (
                                <CopyButton
                                  value={apiKeyPlainText(item)}
                                  label="复制完整凭据"
                                />
                              ) : null}
                            </div>
                          </td>
                          <td className={tdClass}>
                            {Array.isArray(item.allowed_models) &&
                            item.allowed_models.length > 0
                              ? item.allowed_models.join(', ')
                              : '不限'}
                          </td>
                          <td className={`${tdClass} whitespace-nowrap`}>
                            {upstreamStrategyLabel(item.upstream_strategy)}
                          </td>
                          <td className={`${tdClass} whitespace-nowrap`}>
                            {defaultReasoningEffortLabel(
                              item.default_reasoning_effort
                            )}
                          </td>
                          <td className={tdClass}>
                            <div className="grid gap-1 text-sm">
                              {renderTokenLimitPair(
                                '总量',
                                item.quota_daily_tokens,
                                item.quota_weekly_tokens
                              ) || (
                                <span className="whitespace-nowrap">
                                  总量：不限
                                </span>
                              )}
                              {renderTokenLimitPair(
                                '输入',
                                item.quota_daily_input_tokens,
                                item.quota_weekly_input_tokens
                              )}
                              {renderTokenLimitPair(
                                '输出',
                                item.quota_daily_output_tokens,
                                item.quota_weekly_output_tokens
                              )}
                              {renderTokenLimitPair(
                                '非缓存输入',
                                item.quota_daily_billable_input_tokens,
                                item.quota_weekly_billable_input_tokens
                              )}
                            </div>
                          </td>
                          <td className={`${tdClass} whitespace-nowrap`}>
                            <StatusBadge
                              active={!item.disabled}
                              trueText="启用"
                              falseText="禁用"
                            />
                          </td>
                        </tr>
                      )
                    })
                  ) : (
                    <tr>
                      <td
                        colSpan={10}
                        className="px-4 py-10 text-center text-sm text-[#9aa39e]"
                      >
                        {hasActiveKeyFilters
                          ? '没有匹配的 API 凭据'
                          : '暂无 API 凭据'}
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
          <TablePagination
            total={filteredKeys.length}
            pagination={keyPagination}
            onChange={setKeyPagination}
            disabled={loading}
          />
        </div>
      </SurfacePanel>
    </div>
  )

  const renderAnalytics = () => {
    const activeKeys = keyTokenStatsRows.filter((item) => !item.disabled).length
    const total24hTokens = keyTokenStatsRows.reduce(
      (sum, item) => sum + asInt(item.tokens['24h'], 0),
      0
    )
    const total30dTokens = keyTokenStatsRows.reduce(
      (sum, item) => sum + asInt(item.tokens['30d'], 0),
      0
    )

    return (
      <div className="space-y-5">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <SummaryCard
            label="凭据数"
            value={fmtNumber(keyTokenStatsRows.length)}
            sub={`${fmtNumber(activeKeys)} 个启用`}
          />
          <SummaryCard
            label="24h Token"
            value={fmtNumber(total24hTokens)}
            sub="按当前筛选后的凭据汇总"
          />
          <SummaryCard
            label="30 天 Token"
            value={fmtNumber(total30dTokens)}
            sub="按当前筛选后的凭据汇总"
          />
          <SummaryCard
            label="统计维度"
            value="凭据"
            sub="模型、趋势和服务错误率后续扩展"
          />
        </div>

        {renderKeyTokenStats({
          title: '凭据维度',
          description: '按当前凭据列表汇总各时间窗口的总 Token；无调用显示 0。',
          showFilters: true,
        })}
      </div>
    )
  }

  const renderModels = () => (
    <SurfacePanel variant="admin" className="p-5 sm:p-6">
      <div className="space-y-5">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold text-[#1f2d25]">模型管理</h2>
            <div className="mt-1 text-sm text-[#7b8780]">
              共 {modelTotal} 个；列表会进入 `/v1/models`
              返回项，并参与请求启停校验。
            </div>
          </div>
        </div>

        <div className={toolbarClass}>
          <div className="admin-module-toolbar-row">
            <div className={selectionBlockClass}>
              <span className="font-semibold text-[#1f2d25]">模型操作</span>
              <span className={selectionTagClass}>
                模型列表随代码固定维护；上下文策略保存后立即影响后续请求
              </span>
            </div>
          </div>
        </div>

        <div className={tableWrapClass}>
          <div className="overflow-auto">
            <table className={`${tableClass} min-w-[1380px]`}>
              <thead>
                <tr>
                  <th className={thClass}>模型</th>
                  <th className={thClass}>来源</th>
                  <th className={thClass}>输入 $/1M</th>
                  <th className={thClass}>缓存输入 $/1M</th>
                  <th className={thClass}>输出 $/1M</th>
                  <th className={thClass}>
                    <HeaderWithHelp help={MODEL_CONTEXT_HELP.window}>
                      上下文窗口
                    </HeaderWithHelp>
                  </th>
                  <th className={thClass}>
                    <HeaderWithHelp help={MODEL_CONTEXT_HELP.compact}>
                      压缩阈值
                    </HeaderWithHelp>
                  </th>
                  <th className={thClass}>
                    <HeaderWithHelp help={MODEL_CONTEXT_HELP.bytes}>
                      字节阈值
                    </HeaderWithHelp>
                  </th>
                  <th className={thClass}>
                    <HeaderWithHelp help={MODEL_CONTEXT_HELP.keep}>
                      保留
                    </HeaderWithHelp>
                  </th>
                  <th className={thClass}>状态</th>
                  <th className={thClass}>操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#e7efe9] bg-white">
                {models.length > 0 ? (
                  paginatedModels.map((item) => {
                    const price = officialModelPriceByID.get(item.model_id)
                    return (
                      <tr key={String(item.id)} className="align-top">
                        <td className={`${tdClass} font-mono`}>
                          {item.model_id}
                          <div className="mt-1 text-xs text-[#9aa39e]">
                            {item.owned_by || '-'}
                          </div>
                        </td>
                        <td className={tdClass}>{item.source || '-'}</td>
                        <td className={tdClass}>
                          {fmtPricePerMillion(price?.input_usd_per_million)}
                        </td>
                        <td className={tdClass}>
                          {fmtPricePerMillion(
                            price?.cached_input_usd_per_million
                          )}
                        </td>
                        <td className={tdClass}>
                          {fmtPricePerMillion(price?.output_usd_per_million)}
                        </td>
                        <td className={tdClass}>
                          {fmtOptionalNumber(
                            item.effective_context_window_tokens
                          )}{' '}
                          tokens
                        </td>
                        <td className={tdClass}>
                          <div className="whitespace-nowrap">
                            {fmtOptionalNumber(
                              item.effective_context_compact_tokens
                            )}
                            {' / '}
                            {fmtOptionalNumber(
                              item.effective_context_hard_tokens
                            )}
                          </div>
                        </td>
                        <td className={tdClass}>
                          <div className="whitespace-nowrap">
                            {fmtOptionalNumber(
                              item.effective_context_compact_bytes
                            )}
                            {' / '}
                            {fmtOptionalNumber(
                              item.effective_context_hard_bytes
                            )}
                          </div>
                        </td>
                        <td className={tdClass}>
                          {fmtOptionalNumber(item.effective_context_keep_items)}
                        </td>
                        <td className={tdClass}>
                          <StatusBadge
                            active={!!item.enabled}
                            trueText="启用"
                            falseText="禁用"
                          />
                        </td>
                        <td className={tdClass}>
                          <div className="flex flex-wrap gap-2">
                            <button
                              type="button"
                              onClick={() => openModelContextModal(item)}
                              disabled={loading}
                              className={tableActionButtonClass}
                            >
                              上下文
                            </button>
                            <button
                              type="button"
                              onClick={() =>
                                setModelEnabled(item.id, !item.enabled)
                              }
                              disabled={loading}
                              className={
                                item.enabled
                                  ? tableDangerButtonClass
                                  : tablePrimaryButtonClass
                              }
                            >
                              {item.enabled ? '禁用' : '启用'}
                            </button>
                          </div>
                        </td>
                      </tr>
                    )
                  })
                ) : (
                  <tr>
                    <td
                      colSpan={11}
                      className="px-4 py-10 text-center text-sm text-[#9aa39e]"
                    >
                      暂无模型
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
        <TablePagination
          total={models.length}
          pagination={modelPagination}
          onChange={setModelPagination}
          disabled={loading}
        />
      </div>
    </SurfacePanel>
  )

  const renderKeyLimitInput = (name, label, hint) => (
    <label className={fieldClass}>
      {label}
      <input
        type="number"
        min="0"
        step="0.1"
        value={keyForm[name]}
        onChange={(e) =>
          setKeyForm((current) => ({
            ...current,
            [name]: e.target.value,
          }))
        }
        className={inputClass}
        placeholder="0 表示不限，1 = 100 万 token"
      />
      <span className={fieldHintClass}>{hint}</span>
    </label>
  )

  const renderConfirmDialog = () => {
    if (!confirmDialog) return null

    const isBusy = confirmSubmitting || resettingKey
    const confirmClass =
      confirmDialog.tone === 'primary' ? primaryButtonClass : dangerButtonClass

    return (
      <div className="admin-modal-backdrop admin-confirm-backdrop">
        <button
          type="button"
          className="admin-modal-overlay"
          aria-label="关闭确认弹窗"
          onClick={closeConfirmDialog}
          disabled={isBusy}
        />
        <div
          className="admin-modal-panel admin-confirm-panel"
          role="dialog"
          aria-modal="true"
          aria-labelledby="admin-confirm-title"
          aria-describedby="admin-confirm-description"
        >
          <div className="admin-modal-header">
            <div>
              <h2 id="admin-confirm-title" className="admin-modal-title">
                {confirmDialog.title}
              </h2>
              <p
                id="admin-confirm-description"
                className="admin-modal-description"
              >
                {confirmDialog.description}
              </p>
            </div>
            <button
              type="button"
              onClick={closeConfirmDialog}
              className="admin-modal-close"
              aria-label="关闭弹窗"
              disabled={isBusy}
            >
              ×
            </button>
          </div>
          <div className="admin-confirm-body">
            <div className="admin-confirm-detail">{confirmDialog.detail}</div>
          </div>
          <div className="admin-modal-footer admin-confirm-footer">
            <button
              type="button"
              onClick={closeConfirmDialog}
              disabled={isBusy}
              className={secondaryButtonClass}
            >
              {confirmDialog.cancelText}
            </button>
            <button
              type="button"
              onClick={submitConfirmDialog}
              disabled={isBusy}
              className={confirmClass}
            >
              {isBusy
                ? confirmDialog.pendingText || '处理中...'
                : confirmDialog.confirmText}
            </button>
          </div>
        </div>
      </div>
    )
  }

  const renderKeyModal = () => {
    if (!keyModalOpen) return null

    return (
      <div className="admin-modal-backdrop">
        <button
          type="button"
          className="admin-modal-overlay"
          aria-label="关闭弹窗"
          onClick={cancelEditKey}
        />
        <div
          className="admin-modal-panel"
          role="dialog"
          aria-modal="true"
          aria-labelledby="key-modal-title"
        >
          <div className="admin-modal-header">
            <div>
              <h2 id="key-modal-title" className="admin-modal-title">
                {editingKeyId ? '编辑 API 凭据' : '新建 API 凭据'}
              </h2>
              <p className="admin-modal-description">
                设置备注、模型限制、上游策略、默认推理档位和 token 日 / 周额度。
              </p>
            </div>
            <button
              type="button"
              onClick={cancelEditKey}
              className="admin-modal-close"
              aria-label="关闭弹窗"
            >
              ×
            </button>
          </div>
          <form onSubmit={saveKey} className="admin-modal-form" noValidate>
            <label className={fieldClass}>
              备注名称
              <input
                value={keyForm.remark}
                onChange={(e) =>
                  setKeyForm((current) => ({
                    ...current,
                    remark: normalizeKeyRemarkInput(e.target.value),
                  }))
                }
                maxLength={KEY_REMARK_MAX_LENGTH}
                pattern={KEY_REMARK_PATTERN}
                inputMode="text"
                className={inputClass}
                placeholder="例如 team1"
              />
              <span className={fieldHintClass}>
                仅支持字母和数字；留空时使用默认备注。保存备注、额度、模型、上游策略或默认推理档位不会重新生成
                API key。
              </span>
            </label>
            {editingKeyId ? (
              <div className="grid gap-2 rounded-lg border border-[#e4ece6] bg-[#f7fbf8] p-3">
                <div>
                  <div className="text-sm font-semibold text-[#1f2d25]">
                    重置 API key
                  </div>
                  <div className={fieldHintClass}>
                    如果该 key 已泄密，可以点击重置 API key。重置后旧 key
                    会立即失效，请把新生成的完整 key 同步到客户端。
                  </div>
                </div>
                <div>
                  <button
                    type="button"
                    onClick={resetKeySecret}
                    disabled={resettingKey || loading}
                    className={dangerButtonClass}
                  >
                    {resettingKey ? '重置中...' : '重置 API key'}
                  </button>
                </div>
              </div>
            ) : null}
            <label className={fieldClass}>
              允许模型
              <SearchableSelect
                value={keyForm.allowedModels}
                onChange={(nextValue) =>
                  setKeyForm((current) => ({
                    ...current,
                    allowedModels: nextValue,
                  }))
                }
                ariaLabel="允许模型"
                options={[
                  ...MODEL_LIMIT_OPTIONS,
                  ...modelOptions.map((modelId) => ({
                    label: `仅允许 ${modelId}`,
                    value: modelId,
                  })),
                ]}
                placeholder="输入模型筛选"
              />
              <span className={fieldHintClass}>
                选项来自模型管理页，避免填入不存在的模型。
              </span>
            </label>
            <label className={fieldClass}>
              上游策略
              <SearchableSelect
                value={keyForm.upstreamStrategy}
                onChange={(nextValue) =>
                  setKeyForm((current) => ({
                    ...current,
                    upstreamStrategy: nextValue,
                  }))
                }
                ariaLabel="上游策略"
                options={KEY_UPSTREAM_STRATEGY_OPTIONS}
                placeholder="输入上游策略"
              />
              <span className={fieldHintClass}>
                默认继承全局；仅对该 API 凭据后续请求生效。
              </span>
            </label>
            <label className={fieldClass}>
              默认推理档位
              <SearchableSelect
                value={keyForm.defaultReasoningEffort}
                onChange={(nextValue) =>
                  setKeyForm((current) => ({
                    ...current,
                    defaultReasoningEffort: nextValue,
                  }))
                }
                ariaLabel="默认推理档位"
                options={KEY_DEFAULT_REASONING_EFFORT_OPTIONS}
                placeholder="输入默认推理档位"
              />
              <span className={fieldHintClass}>
                该设置会覆盖客户端传入的
                reasoning_effort；关闭默认时保留客户端原始档位。
              </span>
            </label>
            <div className="grid gap-3 rounded-lg border border-[#e4ece6] bg-[#f7fbf8] p-3">
              <div>
                <div className="text-sm font-semibold text-[#1f2d25]">
                  总 Token 限制
                </div>
                <div className={fieldHintClass}>
                  按已落库 usage 的总 Token 判断；达到任一额度后返回 429。
                </div>
              </div>
              <div className="grid gap-3 md:grid-cols-2">
                {renderKeyLimitInput(
                  'dailyTokenLimit',
                  '每日总 Token（百万）',
                  '按自然日统计；0 表示不限。'
                )}
                {renderKeyLimitInput(
                  'weeklyTokenLimit',
                  '每周总 Token（百万）',
                  '按自然周统计；0 表示不限。'
                )}
              </div>
            </div>
            <div className="grid gap-3 rounded-lg border border-[#e4ece6] bg-[#f7fbf8] p-3">
              <div>
                <div className="text-sm font-semibold text-[#1f2d25]">
                  细分 Token 限制
                </div>
                <div className={fieldHintClass}>
                  输入、输出和非缓存输入分别按日 / 周独立判断；留空或 0
                  表示不限。
                </div>
              </div>
              <div className="grid gap-3 md:grid-cols-2">
                {renderKeyLimitInput(
                  'dailyInputTokenLimit',
                  '每日输入 Token（百万）',
                  '按 input_tokens 统计，包含缓存输入。'
                )}
                {renderKeyLimitInput(
                  'weeklyInputTokenLimit',
                  '每周输入 Token（百万）',
                  '按 input_tokens 统计，包含缓存输入。'
                )}
                {renderKeyLimitInput(
                  'dailyBillableInputTokenLimit',
                  '每日非缓存输入（百万）',
                  '按 input_tokens - cached_tokens 统计，不伪造缺失缓存值。'
                )}
                {renderKeyLimitInput(
                  'weeklyBillableInputTokenLimit',
                  '每周非缓存输入（百万）',
                  '按 input_tokens - cached_tokens 统计，不伪造缺失缓存值。'
                )}
                {renderKeyLimitInput(
                  'dailyOutputTokenLimit',
                  '每日输出 Token（百万）',
                  '按 output_tokens 统计，reasoning 已包含在输出口径中。'
                )}
                {renderKeyLimitInput(
                  'weeklyOutputTokenLimit',
                  '每周输出 Token（百万）',
                  '按 output_tokens 统计，reasoning 已包含在输出口径中。'
                )}
              </div>
            </div>
            <div className="admin-modal-footer">
              <button
                type="button"
                onClick={cancelEditKey}
                className={secondaryButtonClass}
              >
                取消
              </button>
              <button
                type="submit"
                disabled={loading}
                className={primaryButtonClass}
              >
                {editingKeyId ? '保存凭据' : '生成 API 凭据'}
              </button>
            </div>
          </form>
        </div>
      </div>
    )
  }

  const renderModelContextInput = (
    name,
    label,
    hint,
    help,
    effectiveKey,
    { effectiveSuffix = '', placeholder = '留空=继承', allowUnit = true } = {}
  ) => {
    const effectiveRawValue = asInt(editingModelContext?.[effectiveKey], 0)
    const effectiveValue = fmtEffectiveContextValue(
      editingModelContext,
      effectiveKey,
      effectiveSuffix
    )
    const fillEffectiveValue = () => {
      const nextValue = fmtContextInputValue(effectiveRawValue, { allowUnit })
      if (!nextValue) return
      setModelContextForm((current) => ({
        ...current,
        [name]: nextValue,
      }))
    }
    return (
      <div className="admin-model-context-field">
        <div className="admin-model-context-field-head">
          <span className="admin-model-context-field-title">
            <HeaderWithHelp help={help}>{label}</HeaderWithHelp>
          </span>
          <button
            type="button"
            className="admin-button admin-button-compact admin-model-context-fill"
            onClick={fillEffectiveValue}
            disabled={effectiveRawValue <= 0}
          >
            填入当前值
          </button>
        </div>
        <input
          type="text"
          aria-label={label}
          inputMode="decimal"
          value={modelContextForm[name]}
          onChange={(e) =>
            setModelContextForm((current) => ({
              ...current,
              [name]: e.target.value,
            }))
          }
          className={inputClass}
          placeholder={placeholder}
        />
        <span className={fieldHintClass}>{hint}</span>
        <span className={fieldHintClass}>
          当前生效：{effectiveValue}；输入框留空或填 0
          表示继承该值，不是无限制。
        </span>
      </div>
    )
  }

  const closeModelContextModal = () => {
    setModelContextModalOpen(false)
    setEditingModelContext(null)
    setModelContextForm(INITIAL_MODEL_CONTEXT_FORM)
  }

  const renderModelContextModal = () => {
    if (!modelContextModalOpen || !editingModelContext) return null

    return (
      <div className="admin-modal-backdrop">
        <button
          type="button"
          className="admin-modal-overlay"
          aria-label="关闭模型上下文策略"
          onClick={closeModelContextModal}
        />
        <div
          className="admin-modal-panel admin-model-context-modal"
          role="dialog"
          aria-modal="true"
          aria-labelledby="model-context-modal-title"
        >
          <div className="admin-modal-header">
            <div>
              <h2 id="model-context-modal-title" className="admin-modal-title">
                模型上下文策略
              </h2>
              <p className="admin-modal-description">
                {editingModelContext.model_id}{' '}
                的压缩阈值；保存后仅影响后续请求。留空或 0
                表示继承当前生效值，不是无限制。
              </p>
            </div>
            <button
              type="button"
              onClick={closeModelContextModal}
              className="admin-modal-close"
              aria-label="关闭弹窗"
            >
              ×
            </button>
          </div>
          <form
            onSubmit={saveModelContext}
            className="admin-modal-form"
            noValidate
          >
            <div className="admin-model-context-section">
              <div>
                <div className="admin-model-context-section-title">
                  Token 阈值
                </div>
                <div className={fieldHintClass}>
                  硬拦截必须大于开始压缩；0 使用推荐值；支持 K / M 单位。默认按
                  Codex 体验控制在 400K 窗口内。
                </div>
              </div>
              <div className="admin-model-context-grid">
                {renderModelContextInput(
                  'contextWindowTokens',
                  '上下文窗口 tokens',
                  '用于 /v1/models 元数据和硬阈值比例展示。',
                  MODEL_CONTEXT_HELP.window,
                  'effective_context_window_tokens',
                  {
                    effectiveSuffix: 'tokens',
                    placeholder: '留空=继承，如需覆盖填 400K',
                  }
                )}
                {renderModelContextInput(
                  'contextCompactTokens',
                  '开始压缩 tokens',
                  '请求粗估 token 达到该值时压缩。',
                  MODEL_CONTEXT_HELP.compact,
                  'effective_context_compact_tokens',
                  {
                    effectiveSuffix: 'tokens',
                    placeholder: '留空=继承，如需覆盖填 260K',
                  }
                )}
                {renderModelContextInput(
                  'contextHardTokens',
                  '硬拦截 tokens',
                  '压缩后仍达到该值时返回 413。',
                  MODEL_CONTEXT_HELP.compact,
                  'effective_context_hard_tokens',
                  {
                    effectiveSuffix: 'tokens',
                    placeholder: '留空=继承，如需覆盖填 380K',
                  }
                )}
              </div>
            </div>
            <div className="admin-model-context-section">
              <div>
                <div className="admin-model-context-section-title">
                  字节阈值
                </div>
                <div className={fieldHintClass}>
                  用于保护 JSON、工具输出和附件文本导致的超大请求体；支持 K / M
                  单位，不等同于计费 token。
                </div>
              </div>
              <div className="admin-model-context-grid">
                {renderModelContextInput(
                  'contextCompactBytes',
                  '开始压缩 bytes',
                  '请求体达到该字节数时压缩。',
                  MODEL_CONTEXT_HELP.bytes,
                  'effective_context_compact_bytes',
                  {
                    effectiveSuffix: 'bytes',
                    placeholder: '留空=继承，如需覆盖填 1.04M',
                  }
                )}
                {renderModelContextInput(
                  'contextHardBytes',
                  '硬拦截 bytes',
                  '压缩后仍达到该字节数时返回 413。',
                  MODEL_CONTEXT_HELP.bytes,
                  'effective_context_hard_bytes',
                  {
                    effectiveSuffix: 'bytes',
                    placeholder: '留空=继承，如需覆盖填 1.9M',
                  }
                )}
                {renderModelContextInput(
                  'contextKeepItems',
                  '保留最近条数',
                  '压缩时保留最近 messages / input items。',
                  MODEL_CONTEXT_HELP.keep,
                  'effective_context_keep_items',
                  {
                    effectiveSuffix: '条',
                    placeholder: '留空=继承，如需覆盖填 8',
                    allowUnit: false,
                  }
                )}
              </div>
            </div>
            <div className="admin-modal-footer">
              <button
                type="button"
                onClick={closeModelContextModal}
                className={secondaryButtonClass}
              >
                取消
              </button>
              <button
                type="submit"
                disabled={loading}
                className={primaryButtonClass}
              >
                保存策略
              </button>
            </div>
          </form>
        </div>
      </div>
    )
  }

  const renderUsageBucketDetailModal = () => {
    if (!selectedUsageBucket) return null

    return (
      <div className="admin-modal-backdrop">
        <button
          type="button"
          className="admin-modal-overlay"
          aria-label="关闭每日模型详情"
          onClick={() => setSelectedUsageBucket(null)}
        />
        <div
          className="admin-modal-panel admin-usage-day-model-modal"
          role="dialog"
          aria-modal="true"
          aria-labelledby="usage-day-model-detail-title"
        >
          <div className="admin-modal-header admin-usage-day-model-header">
            <div>
              <h2
                id="usage-day-model-detail-title"
                className="admin-modal-title"
              >
                {selectedUsageBucket.model || '-'}
              </h2>
              <p className="admin-modal-description">
                {fmtDate(selectedUsageBucket.bucket_start)}，上一页 /
                下一页只在当天该模型的请求内分页。
              </p>
            </div>
            <button
              type="button"
              onClick={() => setSelectedUsageBucket(null)}
              className="admin-modal-close"
              aria-label="关闭弹窗"
            >
              ×
            </button>
          </div>
          <div className="admin-usage-day-model-body">
            <div className={tableWrapClass}>
              <div className="overflow-auto">
                <table className={`${tableClass} min-w-[1380px]`}>
                  <thead>
                    <tr>
                      <th className={thClass}>时间</th>
                      <th className={thClass}>凭据备注</th>
                      <th className={thClass}>上游</th>
                      <th className={thClass}>Effort</th>
                      <th className={thClass}>输入 Tokens</th>
                      <th className={thClass}>非缓存输入</th>
                      <th className={thClass}>输出 Tokens</th>
                      <th className={thClass}>缓存 Tokens</th>
                      <th className={thClass}>Reasoning Tokens</th>
                      <th className={thClass}>总 Tokens</th>
                      <th className={thClass}>价格</th>
                      <th className={thClass}>成功</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#e7efe9] bg-white">
                    {selectedUsageBucketItems.length > 0 ? (
                      selectedUsageBucketItems.map((item) => (
                        <tr key={String(item.id)} className="align-top">
                          <td className={`${tdClass} whitespace-nowrap`}>
                            {fmtTs(item.created_at)}
                          </td>
                          <td className={tdClass}>
                            <ApiKeyUsageCell item={item} />
                          </td>
                          <td className={tdClass}>
                            <div className="whitespace-nowrap text-xs">
                              {upstreamModeLabel(item.upstream_mode)}
                            </div>
                            <div className="mt-1 text-xs text-[#9aa39e]">
                              {item.upstream_fallback ? 'fallback' : 'direct'}
                            </div>
                          </td>
                          <td
                            className={`${tdClass} whitespace-nowrap text-xs`}
                          >
                            {reasoningEffortLabel(item.reasoning_effort)}
                          </td>
                          <td className={tdClass}>
                            {fmtNumber(item.input_tokens)}
                          </td>
                          <td className={tdClass}>
                            {fmtNumber(billableInputTokens(item))}
                          </td>
                          <td className={tdClass}>
                            {fmtNumber(item.output_tokens)}
                          </td>
                          <td className={tdClass}>
                            {fmtNumber(item.cached_tokens)}
                          </td>
                          <td className={tdClass}>
                            {fmtNumber(item.reasoning_tokens)}
                          </td>
                          <td className={`${tdClass} font-semibold`}>
                            {fmtNumber(item.total_tokens)}
                          </td>
                          <td className={`${tdClass} whitespace-nowrap`}>
                            {fmtCost(item.estimated_cost_usd)}
                          </td>
                          <td className={tdClass}>
                            <StatusBadge
                              active={!!item.success}
                              trueText="是"
                              falseText="否"
                              falseTone="danger"
                            />
                          </td>
                        </tr>
                      ))
                    ) : (
                      <tr>
                        <td
                          colSpan={12}
                          className="px-4 py-10 text-center text-sm text-[#9aa39e]"
                        >
                          {usageBucketDetailLoading
                            ? '加载中...'
                            : '暂无请求明细'}
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
          <div className="admin-usage-day-model-footer">
            <TablePagination
              total={selectedUsageBucketTotal}
              pagination={usageBucketDetailPagination}
              onChange={changeUsageBucketDetailPagination}
              disabled={usageBucketDetailLoading}
            />
          </div>
        </div>
      </div>
    )
  }

  const renderUsageSessionDetailModal = () => {
    if (!selectedUsageSession) return null

    const summaryRows = [
      ['会话 ID', selectedUsageSession.session_id || '-'],
      ['凭据备注', apiKeyRemark(selectedUsageSession)],
      ['凭据前缀', selectedUsageSession.api_key_prefix || '-'],
      ['首次调用', fmtTs(selectedUsageSession.first_seen_at)],
      ['最近调用', fmtTs(selectedUsageSession.last_seen_at)],
      ['请求数', fmtNumber(selectedUsageSession.total_requests)],
      [
        '成功 / 服务错误 / 取消',
        `${fmtNumber(selectedUsageSession.success_requests)} / ${fmtNumber(
          selectedUsageSession.failed_requests
        )} / ${fmtNumber(selectedUsageSession.client_canceled_requests)}`,
      ],
      [
        'Backend / CLI',
        `${fmtNumber(selectedUsageSession.backend_requests)} / ${fmtNumber(
          selectedUsageSession.cli_requests
        )}`,
      ],
      [
        'Codex / OpenCode / 其他',
        `${fmtNumber(selectedUsageSession.codex_requests)} / ${fmtNumber(
          selectedUsageSession.opencode_requests
        )} / ${fmtNumber(selectedUsageSession.other_client_requests)}`,
      ],
      ['Fallback', fmtNumber(selectedUsageSession.fallback_requests)],
      ['输入 Token', fmtNumber(selectedUsageSession.input_tokens)],
      [
        '非缓存输入 Token',
        fmtNumber(billableInputTokens(selectedUsageSession)),
      ],
      ['输出 Token', fmtNumber(selectedUsageSession.output_tokens)],
      ['缓存 Token', fmtNumber(selectedUsageSession.cached_tokens)],
      ['Reasoning Token', fmtNumber(selectedUsageSession.reasoning_tokens)],
      ['总 Token', fmtNumber(selectedUsageSession.total_tokens)],
      ['平均耗时', `${fmtNumber(selectedUsageSession.average_duration_ms)} ms`],
      ['费用估算', fmtCost(selectedUsageSession.estimated_cost_usd)],
    ]

    return (
      <div className="admin-modal-backdrop">
        <button
          type="button"
          className="admin-modal-overlay"
          aria-label="关闭会话详情"
          onClick={() => setSelectedUsageSession(null)}
        />
        <div
          className="admin-modal-panel admin-usage-session-modal"
          role="dialog"
          aria-modal="true"
          aria-labelledby="usage-session-detail-title"
        >
          <div className="admin-modal-header">
            <div>
              <h2 id="usage-session-detail-title" className="admin-modal-title">
                会话详情
              </h2>
              <p className="admin-modal-description">
                按同一个 session_id 聚合 usage；详情只展开请求级排障字段。
              </p>
            </div>
            <button
              type="button"
              onClick={() => setSelectedUsageSession(null)}
              className="admin-modal-close"
              aria-label="关闭弹窗"
            >
              ×
            </button>
          </div>
          <div className="admin-usage-detail-grid">
            {summaryRows.map(([label, value]) => (
              <div key={label} className="admin-usage-detail-item">
                <div className="admin-usage-detail-label">{label}</div>
                <div className="admin-usage-detail-value">{value}</div>
              </div>
            ))}
          </div>
          <div className="admin-usage-session-calls">
            <div className="mb-3 text-sm font-semibold text-[#365141]">
              请求明细
            </div>
            <div className={tableWrapClass}>
              <div className="overflow-auto">
                <table className={`${tableClass} min-w-[1340px]`}>
                  <thead>
                    <tr>
                      <th className={thClass}>时间</th>
                      <th className={thClass}>请求 ID</th>
                      <th className={thClass}>客户端 IP</th>
                      <th className={thClass}>接口</th>
                      <th className={thClass}>模型</th>
                      <th className={thClass}>Effort</th>
                      <th className={thClass}>上游</th>
                      <th className={thClass}>状态</th>
                      <th className={thClass}>Token</th>
                      <th className={thClass}>费用估算</th>
                      <th className={thClass}>耗时</th>
                      <th className={thClass}>
                        <HeaderWithHelp help={GATEWAY_ERROR_TYPE_HELP}>
                          错误
                        </HeaderWithHelp>
                      </th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[#e7efe9] bg-white">
                    {selectedUsageSessionItems.length > 0 ? (
                      selectedUsageSessionItems.map((item) => (
                        <tr key={String(item.id)} className="align-top">
                          <td className={tdClass}>{fmtTs(item.created_at)}</td>
                          <td className={`${tdClass} font-mono text-xs`}>
                            {item.request_id || '-'}
                          </td>
                          <td className={`${tdClass} font-mono text-xs`}>
                            {item.client_ip || '-'}
                          </td>
                          <td className={tdClass}>
                            {item.endpoint || item.path || '-'}
                          </td>
                          <td className={`${tdClass} font-mono text-xs`}>
                            {item.model || '-'}
                          </td>
                          <td
                            className={`${tdClass} whitespace-nowrap text-xs`}
                          >
                            {reasoningEffortLabel(item.reasoning_effort)}
                          </td>
                          <td className={tdClass}>
                            <div className="whitespace-nowrap text-xs">
                              {upstreamModeLabel(item.upstream_mode)}
                            </div>
                            <div className="mt-1 text-xs text-[#9aa39e]">
                              {item.upstream_fallback ? 'fallback' : 'direct'}
                            </div>
                          </td>
                          <td className={tdClass}>
                            <StatusBadge
                              active={!!item.success}
                              trueText={`HTTP ${item.status_code}`}
                              falseText={
                                isClientCanceledUsage(item)
                                  ? `已取消 HTTP ${item.status_code}`
                                  : `HTTP ${item.status_code}`
                              }
                              falseTone={
                                isClientCanceledUsage(item)
                                  ? 'neutral'
                                  : 'danger'
                              }
                            />
                          </td>
                          <td className={tdClass}>
                            {fmtNumber(item.total_tokens)}
                            <div className="mt-1 text-xs text-[#9aa39e]">
                              {fmtNumber(item.input_tokens)} /{' '}
                              {fmtNumber(item.output_tokens)}
                            </div>
                            <div className="mt-1 text-xs text-[#9aa39e]">
                              非缓存输入 {fmtNumber(billableInputTokens(item))}
                            </div>
                          </td>
                          <td className={`${tdClass} whitespace-nowrap`}>
                            {fmtCost(item.estimated_cost_usd)}
                          </td>
                          <td className={tdClass}>
                            {fmtNumber(item.duration_ms)} ms
                          </td>
                          <td className={tdClass}>
                            <ErrorTypeCell value={item.error_type} />
                          </td>
                        </tr>
                      ))
                    ) : (
                      <tr>
                        <td
                          colSpan={12}
                          className="px-4 py-8 text-center text-sm text-[#9aa39e]"
                        >
                          {usageSessionDetailLoading
                            ? '加载中...'
                            : '暂无会话请求明细'}
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </div>
      </div>
    )
  }

  const renderUsageSummaryCards = () => (
    <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-6">
      <SummaryCard
        label="请求数"
        value={fmtNumber(summary.total_requests)}
        sub={`${fmtNumber(summary.success_requests)} 成功 / ${fmtNumber(summary.failed_requests)} 服务错误 / ${fmtNumber(summary.client_canceled_requests)} 取消`}
        help="请求数按当前用量日志筛选窗口统计；成功、服务错误和客户端取消分别按 usage 真源归类。"
      />
      <SummaryCard
        label="总 Token"
        value={fmtNumber(summary.total_tokens)}
        sub={`${fmtNumber(summary.input_tokens)} 输入（非缓存 ${fmtNumber(billableInputTokens(summary))}） / ${fmtNumber(summary.output_tokens)} 输出`}
        help="总 Token = 输入 Token + 输出 Token；非缓存输入用于估算实际计费输入。"
      />
      <SummaryCard
        label="费用估算"
        value={fmtCost(summary.estimated_cost_usd)}
        sub="按当前模型价格口径估算"
        help="费用按当前模型价格表估算；未配置价格的模型不会被伪造成真实费用。"
      />
      <SummaryCard
        label="上游分布"
        value={`${fmtNumber(summary.backend_requests)} / ${fmtNumber(summary.cli_requests)}`}
        sub={`${fmtNumber(summary.fallback_requests)} 次 fallback`}
        help="主值依次为 Backend 直连 / CLI 执行次数；fallback 表示从主上游切到兜底上游的次数。"
      />
      <SummaryCard
        label="客户端分布"
        value={`${fmtNumber(summary.codex_requests)} / ${fmtNumber(summary.opencode_requests)}`}
        sub={`${fmtNumber(summary.other_client_requests)} 次其他客户端`}
        help="主值依次为 Codex / OpenCode 请求数；其他客户端单独汇总。"
      />
      <SummaryCard
        label="服务错误率"
        value={fmtRate(summary.failed_requests, summary.total_requests)}
        sub={`${fmtNumber(summary.client_canceled_requests)} 次客户端取消 / ${fmtNumber(summary.average_duration_ms)} ms 平均耗时`}
        help="服务错误率 = 服务端或上游错误请求 / 当前筛选窗口请求总数；客户端取消单独展示。"
        helpAlign="end"
      />
    </div>
  )

  const renderUpstreamModeControl = () => (
    <SurfacePanel variant="admin" className="p-5 sm:p-6">
      <div className="flex flex-col gap-5">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-[#1f2d25]">
              Codex 上游策略
            </h2>
            <div className="mt-1 text-sm text-[#7b8780]">
              Backend 直连失败时直接返回错误；CLI 兜底只作为临时救急； 强制 CLI
              会每次走服务端 codex exec。
            </div>
          </div>
          <div
            className="admin-view-tabs"
            role="tablist"
            aria-label="Codex 上游策略"
          >
            {CODEX_UPSTREAM_STRATEGY_OPTIONS.map((item) => (
              <button
                key={item.value}
                type="button"
                role="tab"
                aria-selected={gatewayUpstreamStrategy === item.value}
                title={item.description}
                onClick={() => changeGatewayUpstreamStrategy(item.value)}
                disabled={loading || gatewayUpstreamSaving}
                className="admin-view-tab"
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>
        <div className="flex flex-col gap-4 border-t border-[#e4ece6] pt-5 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h3 className="text-base font-semibold text-[#1f2d25]">
              全局默认推理档位
            </h3>
            <div className="mt-1 text-sm text-[#7b8780]">
              默认关闭；开启后会覆盖客户端传入的 reasoning_effort，key
              可单独覆盖或关闭。
            </div>
          </div>
          <div
            className="admin-view-tabs"
            role="group"
            aria-label="全局默认推理档位"
          >
            {DEFAULT_REASONING_EFFORT_OPTIONS.map((item) => (
              <button
                key={item.value || 'off'}
                type="button"
                aria-pressed={gatewayDefaultReasoningEffort === item.value}
                title={item.description}
                onClick={() => changeGatewayDefaultReasoningEffort(item.value)}
                disabled={loading || gatewayUpstreamSaving}
                className={`admin-view-tab ${
                  gatewayDefaultReasoningEffort === item.value
                    ? 'admin-view-tab-active'
                    : ''
                }`}
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>
      </div>
    </SurfacePanel>
  )

  const renderDailyUsage = () => {
    const activeTimeRange = getUsageTimeRange(appliedUsageFilters.timeRange)

    return (
      <SurfacePanel variant="admin" className="p-5 sm:p-6">
        <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 className="text-lg font-semibold text-[#1f2d25]">
              每日模型汇总
            </h2>
            <div className="mt-1 text-sm text-[#7b8780]">
              当前 {activeTimeRange.label}{' '}
              窗口按日期和模型聚合请求、Token、费用估算和服务错误率；点击详情后只在当天该模型的请求内分页。
            </div>
          </div>
          <div className="text-sm text-[#7b8780]">
            {fmtNumber(dailyUsageRows.length)} 组
          </div>
        </div>
        <div className={tableWrapClass}>
          <div className="overflow-auto">
            <table className={`${tableClass} min-w-[1420px]`}>
              <thead>
                <tr>
                  <th className={thClass}>日期</th>
                  <th className={thClass}>模型</th>
                  <th className={thClass}>请求</th>
                  <th className={thClass}>上游</th>
                  <th className={thClass}>客户端</th>
                  <th className={thClass}>输入 Tokens</th>
                  <th className={thClass}>非缓存输入</th>
                  <th className={thClass}>输出 Tokens</th>
                  <th className={thClass}>总费用</th>
                  <th className={thClass}>状态</th>
                  <th className={thClass}>详情</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#e7efe9] bg-white">
                {paginatedDailyUsageRows.length > 0 ? (
                  paginatedDailyUsageRows.map((item) => {
                    const failedRequests = asInt(item.failed_requests, 0)
                    const canceledRequests = asInt(
                      item.client_canceled_requests,
                      0
                    )
                    const totalRequests = asInt(item.total_requests, 0)
                    return (
                      <tr
                        key={`${item.bucket_start}-${item.model || '-'}`}
                        className="align-top"
                      >
                        <td className={`${tdClass} whitespace-nowrap`}>
                          {fmtDate(item.bucket_start)}
                        </td>
                        <td className={tdClass}>
                          <span className="admin-model-pill">
                            {item.model || '-'}
                          </span>
                        </td>
                        <td className={`${tdClass} font-semibold`}>
                          {fmtNumber(item.total_requests)}
                          <div className="mt-1 text-xs font-normal text-[#9aa39e]">
                            {fmtNumber(item.success_requests)} 成功 /{' '}
                            {fmtNumber(item.failed_requests)} 服务错误
                            {canceledRequests > 0
                              ? ` / ${fmtNumber(canceledRequests)} 取消`
                              : ''}
                          </div>
                        </td>
                        <td className={tdClass}>{renderUpstreamStats(item)}</td>
                        <td className={tdClass}>{renderClientStats(item)}</td>
                        <td className={tdClass}>
                          {fmtNumber(item.input_tokens)}
                        </td>
                        <td className={tdClass}>
                          {fmtNumber(billableInputTokens(item))}
                        </td>
                        <td className={tdClass}>
                          {fmtNumber(item.output_tokens)}
                        </td>
                        <td
                          className={`${tdClass} whitespace-nowrap font-semibold`}
                        >
                          {fmtCost(item.estimated_cost_usd)}
                        </td>
                        <td className={tdClass}>
                          <span
                            className={
                              failedRequests > 0
                                ? 'admin-usage-status-danger'
                                : 'admin-usage-status-ok'
                            }
                          >
                            {failedRequests > 0 ? '!' : '✓'}{' '}
                            {fmtRate(failedRequests, totalRequests)} 服务错误
                          </span>
                        </td>
                        <td className={tdClass}>
                          <button
                            type="button"
                            onClick={() => openUsageBucketDetail(item)}
                            className={tableActionButtonClass}
                          >
                            详情
                          </button>
                        </td>
                      </tr>
                    )
                  })
                ) : (
                  <tr>
                    <td
                      colSpan={11}
                      className="px-4 py-10 text-center text-sm text-[#9aa39e]"
                    >
                      {loading ? '加载中...' : '暂无每日模型汇总'}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
        <TablePagination
          total={dailyUsageRows.length}
          pagination={dailyUsagePagination}
          onChange={setDailyUsagePagination}
          disabled={loading}
        />
      </SurfacePanel>
    )
  }

  const renderSessionUsage = () => (
    <SurfacePanel variant="admin" className="p-5 sm:p-6">
      <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-[#1f2d25]">会话聚合</h2>
          <div className="mt-1 text-sm text-[#7b8780]">
            按客户端传入的 session_id 聚合同一会话的请求、Token、费用和耗时。
          </div>
        </div>
        <div className="text-sm text-[#7b8780]">
          {fmtNumber(usageSessionTotal)} 个会话
        </div>
      </div>
      <div className={tableWrapClass}>
        <div className="overflow-auto">
          <table className={`${tableClass} min-w-[1740px]`}>
            <thead>
              <tr>
                <th className={thClass}>最近调用</th>
                <th className={thClass}>会话 ID</th>
                <th className={thClass}>凭据</th>
                <th className={thClass}>请求</th>
                <th className={thClass}>上游</th>
                <th className={thClass}>客户端</th>
                <th className={thClass}>Token</th>
                <th className={thClass}>上下文压缩</th>
                <th className={thClass}>费用估算</th>
                <th className={thClass}>平均耗时</th>
                <th className={thClass}>详情</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[#e7efe9] bg-white">
              {usageSessionItems.length > 0 ? (
                usageSessionItems.map((item) => (
                  <tr key={item.session_id} className="align-top">
                    <td className={`${tdClass} whitespace-nowrap`}>
                      {fmtTs(item.last_seen_at)}
                      <div className="mt-1 text-xs text-[#9aa39e]">
                        首次 {fmtTs(item.first_seen_at)}
                      </div>
                    </td>
                    <td
                      className={`${tdClass} max-w-[280px] font-mono text-xs`}
                    >
                      <span className="break-all">
                        {item.session_id || '-'}
                      </span>
                    </td>
                    <td className={tdClass}>
                      <ApiKeyUsageCell item={item} />
                    </td>
                    <td className={`${tdClass} font-semibold`}>
                      {fmtNumber(item.total_requests)}
                      <div className="mt-1 text-xs font-normal text-[#9aa39e]">
                        {fmtNumber(item.success_requests)} 成功 /{' '}
                        {fmtNumber(item.failed_requests)} 服务错误
                        {asInt(item.client_canceled_requests, 0) > 0
                          ? ` / ${fmtNumber(item.client_canceled_requests)} 取消`
                          : ''}
                      </div>
                    </td>
                    <td className={tdClass}>{renderUpstreamStats(item)}</td>
                    <td className={tdClass}>{renderClientStats(item)}</td>
                    <td className={tdClass}>
                      {fmtNumber(item.total_tokens)}
                      <div className="mt-1 text-xs text-[#9aa39e]">
                        {fmtNumber(item.input_tokens)} /{' '}
                        {fmtNumber(item.output_tokens)}
                      </div>
                      <div className="mt-1 text-xs text-[#9aa39e]">
                        非缓存 {fmtNumber(billableInputTokens(item))}
                      </div>
                    </td>
                    <td className={`${tdClass} max-w-[260px]`}>
                      {asInt(item.context_compaction_count, 0) > 0 ? (
                        <div className="text-xs leading-5 text-[#7b8780]">
                          <div className="font-semibold text-[#1f2d25]">
                            {fmtNumber(item.context_compaction_count)} 次压缩
                          </div>
                          <div>
                            {fmtNumber(item.context_original_tokens)}
                            {' -> '}
                            {fmtNumber(item.context_compacted_tokens)} tokens
                          </div>
                          <div>
                            {fmtNumber(item.context_original_bytes)}
                            {' -> '}
                            {fmtNumber(item.context_compacted_bytes)}B
                          </div>
                          {item.context_summary ? (
                            <div className="mt-1 line-clamp-2 break-all">
                              {item.context_summary}
                            </div>
                          ) : null}
                        </div>
                      ) : (
                        '-'
                      )}
                    </td>
                    <td className={`${tdClass} whitespace-nowrap`}>
                      {fmtCost(item.estimated_cost_usd)}
                    </td>
                    <td className={tdClass}>
                      {fmtNumber(item.average_duration_ms)} ms
                    </td>
                    <td className={tdClass}>
                      <button
                        type="button"
                        onClick={() => openUsageSessionDetail(item)}
                        className={tableActionButtonClass}
                      >
                        详情
                      </button>
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td
                    colSpan={11}
                    className="px-4 py-10 text-center text-sm text-[#9aa39e]"
                  >
                    {loading ? '加载中...' : '暂无带 session_id 的会话记录'}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
      <TablePagination
        total={usageSessionTotal}
        pagination={usagePagination}
        onChange={setUsagePagination}
        disabled={loading}
      />
    </SurfacePanel>
  )

  const renderUsage = () => {
    const activeTimeRange = getUsageTimeRange(appliedUsageFilters.timeRange)
    const activePagination = clampPagination(usagePagination, usageTotal)
    const activeSessionPagination = clampPagination(
      usagePagination,
      usageSessionTotal
    )
    const activeTotal = usageTab === 'sessions' ? usageSessionTotal : usageTotal
    const activePage =
      usageTab === 'sessions' ? activeSessionPagination : activePagination
    const usageStart =
      activeTotal > 0 ? (activePage.current - 1) * activePage.pageSize + 1 : 0
    const usageEnd = Math.min(
      activeTotal,
      activePage.current * activePage.pageSize
    )
    const usageUnit = usageTab === 'sessions' ? '会话' : '请求'
    const detailTitle = usageTab === 'errors' ? '异常请求' : '调用明细'
    const detailDescription =
      usageTab === 'errors'
        ? '默认排除客户端取消，仅展示服务 / 上游 / 网关失败；可用错误 / 中断类型或状态码单独查看 499。'
        : '按请求级 usage 真源直接展示状态、Token、缓存、Reasoning、字节、费用估算、耗时和错误类型。'

    return (
      <div className="space-y-5">
        {renderUsageSummaryCards()}

        <SurfacePanel variant="admin" className="p-5 sm:p-6">
          <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-[#1f2d25]">用量日志</h2>
              <div className="mt-1 text-sm text-[#7b8780]">
                {activeTimeRange.label} 范围内第 {usageStart}-{usageEnd} 条 / 共{' '}
                {activeTotal} 条{usageUnit}。
              </div>
            </div>
          </div>

          <div className={`${toolbarClass} mb-5`}>
            <form
              onSubmit={(e) => {
                e.preventDefault()
                const nextFilters = usageFilters
                const nextPagination = {
                  ...usagePagination,
                  current: 1,
                }
                setAppliedUsageFilters(nextFilters)
                setUsagePagination(nextPagination)
                setDailyUsagePagination((current) => ({
                  ...current,
                  current: 1,
                }))
                loadAll({
                  usageFilterOverride: nextFilters,
                  usagePaginationOverride: nextPagination,
                })
              }}
              className="admin-module-toolbar-row"
            >
              <div
                className={`${filterGroupClass} admin-module-filter-group-wide`}
              >
                <label className={fieldClass}>
                  时间范围
                  <SearchableSelect
                    value={usageFilters.timeRange}
                    onChange={(nextValue) => {
                      setUsageTimeRangeTouched(true)
                      setUsageFilters((current) => ({
                        ...current,
                        timeRange: nextValue,
                      }))
                    }}
                    ariaLabel="时间范围"
                    options={USAGE_TIME_RANGE_OPTIONS}
                    placeholder="输入时间范围"
                  />
                </label>
                <label className={fieldClass}>
                  调用凭据
                  <SearchableMultiSelect
                    value={usageFilters.keyIds}
                    onChange={(nextValue) =>
                      setUsageFilters((current) => ({
                        ...current,
                        keyIds: nextValue,
                      }))
                    }
                    ariaLabel="调用凭据"
                    options={keys.map((item) => ({
                      label: item.name || item.key_prefix || `凭据 ${item.id}`,
                      value: String(item.id),
                    }))}
                    placeholder="输入凭据筛选"
                    summaryLabel="凭据"
                  />
                </label>
                <label className={fieldClass}>
                  请求模型
                  <SearchableSelect
                    value={usageFilters.model}
                    onChange={(nextValue) =>
                      setUsageFilters((current) => ({
                        ...current,
                        model: nextValue,
                      }))
                    }
                    ariaLabel="请求模型"
                    options={[
                      { label: '全部模型', value: '' },
                      ...modelOptions.map((modelId) => ({
                        label: modelId,
                        value: modelId,
                      })),
                    ]}
                    placeholder="输入模型筛选"
                  />
                </label>
                <label className={fieldClass}>
                  Effort
                  <SearchableSelect
                    value={usageFilters.reasoningEffort}
                    onChange={(nextValue) =>
                      setUsageFilters((current) => ({
                        ...current,
                        reasoningEffort: nextValue,
                      }))
                    }
                    ariaLabel="Reasoning effort"
                    options={USAGE_REASONING_EFFORT_FILTER_OPTIONS}
                    placeholder="输入 Effort"
                  />
                </label>
                <label className={fieldClass}>
                  请求状态
                  <SearchableSelect
                    value={usageFilters.success}
                    onChange={(nextValue) =>
                      setUsageFilters((current) => ({
                        ...current,
                        success: nextValue,
                      }))
                    }
                    ariaLabel="请求状态"
                    options={USAGE_SUCCESS_FILTER_OPTIONS}
                    placeholder="输入状态筛选"
                  />
                </label>
                <label className={fieldClass}>
                  状态码
                  <SearchableSelect
                    value={usageFilters.statusCode}
                    onChange={(nextValue) =>
                      setUsageFilters((current) => ({
                        ...current,
                        statusCode: nextValue,
                      }))
                    }
                    ariaLabel="HTTP 状态码"
                    options={USAGE_STATUS_CODE_FILTER_OPTIONS}
                    placeholder="输入状态码"
                  />
                </label>
                <label className={fieldClass}>
                  实际上游
                  <SearchableSelect
                    value={usageFilters.upstreamMode}
                    onChange={(nextValue) =>
                      setUsageFilters((current) => ({
                        ...current,
                        upstreamMode: nextValue,
                      }))
                    }
                    ariaLabel="实际执行上游"
                    options={USAGE_UPSTREAM_FILTER_OPTIONS}
                    placeholder="输入实际上游"
                  />
                </label>
                <label className={fieldClass}>
                  调用客户端
                  <SearchableSelect
                    value={usageFilters.clientType}
                    onChange={(nextValue) =>
                      setUsageFilters((current) => ({
                        ...current,
                        clientType: nextValue,
                      }))
                    }
                    ariaLabel="调用客户端"
                    options={USAGE_CLIENT_TYPE_OPTIONS}
                    placeholder="输入客户端"
                  />
                </label>
                <label className={fieldClass}>
                  错误 / 中断类型
                  <SearchableSelect
                    value={usageFilters.errorType}
                    onChange={(nextValue) =>
                      setUsageFilters((current) => ({
                        ...current,
                        errorType: nextValue,
                      }))
                    }
                    ariaLabel="错误或中断类型"
                    options={USAGE_ERROR_FILTER_OPTIONS}
                    placeholder="输入错误或中断类型"
                  />
                </label>
              </div>
              <div className={primaryActionsClass}>
                <button
                  type="submit"
                  disabled={loading}
                  className={primaryButtonClass}
                >
                  应用筛选
                </button>
                <button
                  type="button"
                  onClick={() => {
                    const nextFilters = applyUsageTabDefaultTimeRange(
                      INITIAL_USAGE_FILTERS,
                      usageTab,
                      false
                    )
                    const nextPagination = {
                      ...usagePagination,
                      current: 1,
                    }
                    setUsageTimeRangeTouched(false)
                    setUsageFilters(nextFilters)
                    setAppliedUsageFilters(nextFilters)
                    setUsagePagination(nextPagination)
                    setDailyUsagePagination((current) => ({
                      ...current,
                      current: 1,
                    }))
                    loadAll({
                      usageFilterOverride: nextFilters,
                      usagePaginationOverride: nextPagination,
                    })
                  }}
                  disabled={loading}
                  className={secondaryButtonClass}
                >
                  重置
                </button>
              </div>
            </form>
          </div>

          <div
            className="admin-view-tabs"
            role="tablist"
            aria-label="用量日志视图"
          >
            {USAGE_TAB_OPTIONS.map((item) => (
              <button
                key={item.key}
                type="button"
                role="tab"
                aria-selected={usageTab === item.key}
                onClick={() => switchUsageTab(item.key)}
                className="admin-view-tab"
              >
                {item.label}
              </button>
            ))}
          </div>
        </SurfacePanel>

        {usageTab === 'daily' ? renderDailyUsage() : null}
        {usageTab === 'keys'
          ? renderKeyTokenStats({
              title: '凭据统计',
              description:
                '按当前筛选条件汇总每个 API 凭据的 Token 窗口；无调用显示 0。',
              showFilters: true,
            })
          : null}
        {usageTab === 'sessions' ? renderSessionUsage() : null}
        {usageTab === 'details' || usageTab === 'errors' ? (
          <SurfacePanel variant="admin" className="p-5 sm:p-6">
            <div className="mb-5">
              <h2 className="text-lg font-semibold text-[#1f2d25]">
                {detailTitle}
              </h2>
              <div className="mt-1 text-sm text-[#7b8780]">
                {detailDescription}
              </div>
            </div>
            {renderUsageTable(false)}
            <TablePagination
              total={usageTotal}
              pagination={usagePagination}
              onChange={setUsagePagination}
              disabled={loading}
            />
          </SurfacePanel>
        ) : null}
      </div>
    )
  }

  const renderUpstream = () => (
    <div className="space-y-5">{renderUpstreamModeControl()}</div>
  )

  return (
    <AdminFrame
      breadcrumb={`${currentConfig.section || '配置管理'} / ${currentConfig.title}`}
      title={currentConfig.title}
      description={currentConfig.description}
    >
      <div className="space-y-6">
        {errMsg ? (
          <div className="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
            {errMsg}
          </div>
        ) : null}

        {currentView === 'keys' && newKey?.plain_key ? (
          <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm leading-6 text-amber-800">
            <div className="font-semibold">新 key 已生成</div>
            <div>完整 key 已保存，可在凭据列表继续查看和复制。</div>
            <div className="mt-2 flex flex-col gap-2 sm:flex-row sm:items-start">
              <div className="min-w-0 flex-1 break-all font-mono text-xs text-[#1f2d25] sm:text-sm">
                {newKey.plain_key}
              </div>
              <CopyButton value={newKey.plain_key} label="复制完整凭据" />
            </div>
          </div>
        ) : null}

        {currentView === 'keys' && newKeyBatch.length > 0 ? (
          <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm leading-6 text-amber-800">
            <div className="font-semibold">
              批量重置已完成，共生成 {newKeyBatch.length} 个新 key
            </div>
            <div>旧 key 已立即失效，请把新完整 key 同步到对应客户端。</div>
            <div className="mt-2">
              <CopyButton
                value={newKeyBatch
                  .map((item) => `${item.name || item.id}: ${item.plain_key}`)
                  .join('\n')}
                label="复制全部完整凭据"
              />
            </div>
            <div className="mt-3 grid gap-2">
              {newKeyBatch.map((item) => (
                <div
                  key={item.id}
                  className="flex flex-col gap-2 rounded-md border border-amber-200 bg-white/65 px-3 py-2 sm:flex-row sm:items-start"
                >
                  <div className="min-w-[120px] font-semibold text-amber-900">
                    {item.name || `ID ${item.id}`}
                  </div>
                  <div className="min-w-0 flex-1 break-all font-mono text-xs text-[#1f2d25] sm:text-sm">
                    {item.plain_key}
                  </div>
                  <CopyButton value={item.plain_key} label="复制完整凭据" />
                </div>
              ))}
            </div>
          </div>
        ) : null}

        {currentView === 'dashboard' ? renderDashboard() : null}
        {currentView === 'keys' ? renderKeys() : null}
        {currentView === 'models' ? renderModels() : null}
        {currentView === 'upstream' ? renderUpstream() : null}
        {currentView === 'analytics' ? renderAnalytics() : null}
        {currentView === 'usage' ? renderUsage() : null}
      </div>
      {currentView === 'keys' ? renderKeyModal() : null}
      {currentView === 'models' ? renderModelContextModal() : null}
      {renderUsageBucketDetailModal()}
      {renderUsageSessionDetailModal()}
      {renderConfirmDialog()}
    </AdminFrame>
  )
}
