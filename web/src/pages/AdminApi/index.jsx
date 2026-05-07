import React, { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AppShell from '@/common/components/layout/AppShell'
import SurfacePanel from '@/common/components/layout/SurfacePanel'
import { AUTH_SCOPE } from '@/common/auth/auth'
import { ADMIN_BASE_PATH } from '@/common/utils/adminRpc'
import { getActionErrorMessage } from '@/common/utils/errorMessage'
import { JsonRpc } from '@/common/utils/jsonRpc'

const PAGE_SIZE = 30
const DASHBOARD_USAGE_SIZE = 8
const DAY_SECONDS = 24 * 60 * 60

const VIEW_CONFIG = {
  dashboard: {
    title: '业务看板',
    description: '只展示 API 转发、token 用量和最近异常线索，不承载配置操作。',
  },
  keys: {
    title: '下游 key',
    description: '创建、启停和查看下游调用凭据；明文 key 只在创建时返回一次。',
  },
  models: {
    title: '模型列表',
    description: '维护 `/v1/models` 返回项和请求模型启停状态。',
  },
  usage: {
    title: 'usage 记录',
    description: '查看 24 小时内最近请求，排查状态码、模型、token 和错误类型。',
  },
}

function asInt(v, fallback = 0) {
  const n = Number(v)
  return Number.isFinite(n) ? Math.trunc(n) : fallback
}

function fmtNumber(v) {
  return new Intl.NumberFormat().format(asInt(v, 0))
}

function fmtTs(ts) {
  if (!ts) return '-'
  const d = new Date(Number(ts) * 1000)
  if (Number.isNaN(d.getTime())) return String(ts)
  return d.toLocaleString()
}

function splitModels(value) {
  return String(value || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function SummaryCard({ label, value, sub }) {
  return (
    <SurfacePanel className="p-4">
      <div className="text-xs font-medium uppercase tracking-[0.18em] text-slate-400">
        {label}
      </div>
      <div className="mt-3 text-2xl font-semibold text-slate-50">{value}</div>
      {sub ? <div className="mt-1 text-sm text-slate-400">{sub}</div> : null}
    </SurfacePanel>
  )
}

function StatusBadge({ active, trueText = '启用', falseText = '禁用' }) {
  return (
    <span
      className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold ${
        active
          ? 'bg-emerald-500/15 text-emerald-200'
          : 'bg-zinc-500/15 text-zinc-200'
      }`}
    >
      {active ? trueText : falseText}
    </span>
  )
}

export default function AdminApiPage({ view = 'dashboard' }) {
  const navigate = useNavigate()
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
  const [models, setModels] = useState([])
  const [modelTotal, setModelTotal] = useState(0)
  const [usageItems, setUsageItems] = useState([])
  const [usageTotal, setUsageTotal] = useState(0)
  const [newKey, setNewKey] = useState(null)
  const [keyForm, setKeyForm] = useState({
    name: '',
    allowedModels: '',
    quotaRequests: '',
    quotaTokens: '',
  })
  const [modelForm, setModelForm] = useState({
    modelId: '',
    ownedBy: 'openai',
  })

  const setKeyListState = (res) => {
    const items = Array.isArray(res?.data?.items) ? res.data.items : []
    setKeys(items)
    setKeyTotal(asInt(res?.data?.total, items.length))
  }

  const setModelListState = (res) => {
    const items = Array.isArray(res?.data?.items) ? res.data.items : []
    setModels(items)
    setModelTotal(asInt(res?.data?.total, items.length))
  }

  const setUsageListState = (res) => {
    setUsageItems(Array.isArray(res?.data?.items) ? res.data.items : [])
    setUsageTotal(asInt(res?.data?.total, 0))
    if (res?.data?.summary) {
      setSummary(res.data.summary)
    }
  }

  const loadAll = async () => {
    setLoading(true)
    setErrMsg('')
    try {
      const now = Math.floor(Date.now() / 1000)
      const startTime = now - DAY_SECONDS

      if (currentView === 'dashboard') {
        const [summaryRes, keysRes, modelsRes, usageRes] = await Promise.all([
          apiRpc.call('summary', { start_time: startTime }),
          apiRpc.call('key_list', { limit: 100, offset: 0 }),
          apiRpc.call('model_list', { limit: 200, offset: 0 }),
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
        setKeyListState(await apiRpc.call('key_list', { limit: 100, offset: 0 }))
        return
      }

      if (currentView === 'models') {
        setModelListState(
          await apiRpc.call('model_list', { limit: 200, offset: 0 })
        )
        return
      }

      const usageRes = await apiRpc.call('usage_list', {
        limit: PAGE_SIZE,
        offset: 0,
        start_time: startTime,
      })
      setUsageListState(usageRes)
    } catch (e) {
      setErrMsg(getActionErrorMessage(e, '加载 API 数据'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadAll()
  }, [currentView])

  const createKey = async (e) => {
    e.preventDefault()
    setErrMsg('')
    setNewKey(null)
    try {
      const result = await apiRpc.call('key_create', {
        name: keyForm.name.trim(),
        allowed_models: splitModels(keyForm.allowedModels),
        quota_requests: asInt(keyForm.quotaRequests, 0),
        quota_total_tokens: asInt(keyForm.quotaTokens, 0),
      })
      setNewKey(result?.data || null)
      setKeyForm({
        name: '',
        allowedModels: '',
        quotaRequests: '',
        quotaTokens: '',
      })
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '创建 API key'))
    }
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

  const saveModel = async (e) => {
    e.preventDefault()
    setErrMsg('')
    try {
      await apiRpc.call('model_upsert', {
        model_id: modelForm.modelId.trim(),
        owned_by: modelForm.ownedBy.trim() || 'openai',
        enabled: true,
      })
      setModelForm({ modelId: '', ownedBy: 'openai' })
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '保存模型'))
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

  const renderUsageTable = (compact = false) => (
    <div className="overflow-hidden rounded-3xl border border-white/10">
      <div className="overflow-auto">
        <table
          className={`${compact ? 'min-w-[820px]' : 'min-w-[1080px]'} text-left text-sm text-slate-100`}
        >
          <thead className="bg-white/[0.04] text-slate-300">
            <tr>
              <th className="px-4 py-3 font-medium">时间</th>
              <th className="px-4 py-3 font-medium">key</th>
              <th className="px-4 py-3 font-medium">endpoint</th>
              <th className="px-4 py-3 font-medium">模型</th>
              <th className="px-4 py-3 font-medium">状态</th>
              <th className="px-4 py-3 font-medium">Token</th>
              {!compact ? <th className="px-4 py-3 font-medium">耗时</th> : null}
              {!compact ? <th className="px-4 py-3 font-medium">错误</th> : null}
            </tr>
          </thead>
          <tbody className="divide-white/8 divide-y bg-black/10">
            {usageItems.length > 0 ? (
              usageItems.map((item) => (
                <tr key={String(item.id)} className="align-top">
                  <td className="px-4 py-4 text-slate-300">
                    {fmtTs(item.created_at)}
                  </td>
                  <td className="px-4 py-4 font-mono text-xs text-cyan-100">
                    {item.api_key_prefix || '-'}
                  </td>
                  <td className="px-4 py-4 text-slate-300">
                    {item.endpoint || item.path}
                  </td>
                  <td className="px-4 py-4 font-mono text-xs text-slate-100">
                    {item.model || '-'}
                  </td>
                  <td className="px-4 py-4">
                    <StatusBadge
                      active={!!item.success}
                      trueText={`HTTP ${item.status_code}`}
                      falseText={`HTTP ${item.status_code}`}
                    />
                  </td>
                  <td className="px-4 py-4 text-slate-300">
                    {fmtNumber(item.total_tokens)}
                    <div className="mt-1 text-xs text-slate-500">
                      {fmtNumber(item.input_tokens)} / {fmtNumber(item.output_tokens)}
                    </div>
                  </td>
                  {!compact ? (
                    <td className="px-4 py-4 text-slate-300">
                      {fmtNumber(item.duration_ms)} ms
                    </td>
                  ) : null}
                  {!compact ? (
                    <td className="px-4 py-4 text-slate-300">
                      {item.error_type || '-'}
                    </td>
                  ) : null}
                </tr>
              ))
            ) : (
              <tr>
                <td
                  colSpan={compact ? 6 : 8}
                  className="px-4 py-10 text-center text-sm text-slate-400"
                >
                  暂无 usage 记录
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
            sub={`${fmtNumber(summary.success_requests)} 成功 / ${fmtNumber(summary.failed_requests)} 失败`}
          />
          <SummaryCard
            label="24h Token"
            value={fmtNumber(summary.total_tokens)}
            sub={`${fmtNumber(summary.input_tokens)} 输入 / ${fmtNumber(summary.output_tokens)} 输出`}
          />
          <SummaryCard
            label="下游 key"
            value={fmtNumber(keyTotal)}
            sub={`${fmtNumber(activeKeys)} 个启用`}
          />
          <SummaryCard
            label="模型"
            value={fmtNumber(modelTotal)}
            sub={`${fmtNumber(enabledModels)} 个启用`}
          />
        </div>

        <SurfacePanel className="p-5 sm:p-6">
          <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-slate-50">
                最近 usage 预览
              </h2>
              <div className="mt-1 text-sm text-slate-400">
                24 小时内最近 {usageItems.length} 条 / 共 {usageTotal} 条。
              </div>
            </div>
          </div>
          {renderUsageTable(true)}
        </SurfacePanel>
      </div>
    )
  }

  const renderKeys = () => (
    <SurfacePanel className="p-5 sm:p-6">
      <div className="space-y-5">
        <div className="flex items-center justify-between gap-3">
          <div>
            <h2 className="text-lg font-semibold text-slate-50">下游 API key</h2>
            <div className="mt-1 text-sm text-slate-400">
              共 {keyTotal} 个；明文只在创建时返回，列表只展示前缀和尾号。
            </div>
          </div>
        </div>

        <form onSubmit={createKey} className="grid gap-3 lg:grid-cols-2">
          <input
            value={keyForm.name}
            onChange={(e) =>
              setKeyForm((current) => ({
                ...current,
                name: e.target.value,
              }))
            }
            className="rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-300/50 focus:ring-2 focus:ring-cyan-300/20"
            placeholder="key 名称"
          />
          <input
            value={keyForm.allowedModels}
            onChange={(e) =>
              setKeyForm((current) => ({
                ...current,
                allowedModels: e.target.value,
              }))
            }
            className="rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-300/50 focus:ring-2 focus:ring-cyan-300/20"
            placeholder="限制模型，逗号分隔；留空为不限"
          />
          <input
            value={keyForm.quotaRequests}
            onChange={(e) =>
              setKeyForm((current) => ({
                ...current,
                quotaRequests: e.target.value,
              }))
            }
            inputMode="numeric"
            className="rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-300/50 focus:ring-2 focus:ring-cyan-300/20"
            placeholder="请求配额，0 为不限"
          />
          <input
            value={keyForm.quotaTokens}
            onChange={(e) =>
              setKeyForm((current) => ({
                ...current,
                quotaTokens: e.target.value,
              }))
            }
            inputMode="numeric"
            className="rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-300/50 focus:ring-2 focus:ring-cyan-300/20"
            placeholder="Token 配额，0 为不限"
          />
          <button
            type="submit"
            disabled={loading || !keyForm.name.trim()}
            className="rounded-2xl bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-cyan-200 disabled:cursor-not-allowed disabled:bg-cyan-300/20 disabled:text-slate-400 lg:col-span-2"
          >
            创建 key
          </button>
        </form>

        <div className="overflow-hidden rounded-3xl border border-white/10">
          <div className="overflow-auto">
            <table className="min-w-[760px] text-left text-sm text-slate-100">
              <thead className="bg-white/[0.04] text-slate-300">
                <tr>
                  <th className="px-4 py-3 font-medium">名称</th>
                  <th className="px-4 py-3 font-medium">key</th>
                  <th className="px-4 py-3 font-medium">模型限制</th>
                  <th className="px-4 py-3 font-medium">状态</th>
                  <th className="px-4 py-3 font-medium">操作</th>
                </tr>
              </thead>
              <tbody className="divide-white/8 divide-y bg-black/10">
                {keys.length > 0 ? (
                  keys.map((item) => (
                    <tr key={String(item.id)} className="align-top">
                      <td className="px-4 py-4 font-medium text-slate-50">
                        {item.name}
                        <div className="mt-1 text-xs text-slate-500">
                          最近使用：{fmtTs(item.last_used_at)}
                        </div>
                      </td>
                      <td className="px-4 py-4 font-mono text-xs text-cyan-100">
                        {item.key_prefix}…{item.key_last4}
                      </td>
                      <td className="px-4 py-4 text-slate-300">
                        {Array.isArray(item.allowed_models) &&
                        item.allowed_models.length > 0
                          ? item.allowed_models.join(', ')
                          : '不限'}
                      </td>
                      <td className="px-4 py-4">
                        <StatusBadge
                          active={!item.disabled}
                          trueText="启用"
                          falseText="禁用"
                        />
                      </td>
                      <td className="px-4 py-4">
                        <button
                          type="button"
                          onClick={() => setKeyDisabled(item.id, !item.disabled)}
                          disabled={loading}
                          className={`rounded-full px-4 py-2 text-xs font-semibold transition disabled:cursor-not-allowed disabled:opacity-60 ${
                            item.disabled
                              ? 'bg-emerald-300 text-slate-950 hover:bg-emerald-200'
                              : 'bg-rose-300 text-slate-950 hover:bg-rose-200'
                          }`}
                        >
                          {item.disabled ? '启用' : '禁用'}
                        </button>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td
                      colSpan={5}
                      className="px-4 py-10 text-center text-sm text-slate-400"
                    >
                      暂无 API key
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </SurfacePanel>
  )

  const renderModels = () => (
    <SurfacePanel className="p-5 sm:p-6">
      <div className="space-y-5">
        <div>
          <h2 className="text-lg font-semibold text-slate-50">模型列表</h2>
          <div className="mt-1 text-sm text-slate-400">
            共 {modelTotal} 个；这里控制 `/v1/models` 返回项和请求模型启停。
          </div>
        </div>

        <form onSubmit={saveModel} className="grid gap-3 sm:grid-cols-3">
          <input
            value={modelForm.modelId}
            onChange={(e) =>
              setModelForm((current) => ({
                ...current,
                modelId: e.target.value,
              }))
            }
            className="rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-300/50 focus:ring-2 focus:ring-cyan-300/20 sm:col-span-2"
            placeholder="模型 ID，例如 gpt-5.4"
          />
          <input
            value={modelForm.ownedBy}
            onChange={(e) =>
              setModelForm((current) => ({
                ...current,
                ownedBy: e.target.value,
              }))
            }
            className="rounded-2xl border border-white/10 bg-white/[0.04] px-4 py-3 text-sm text-slate-100 outline-none transition focus:border-cyan-300/50 focus:ring-2 focus:ring-cyan-300/20"
            placeholder="owned_by"
          />
          <button
            type="submit"
            disabled={loading || !modelForm.modelId.trim()}
            className="rounded-2xl bg-cyan-300 px-4 py-3 text-sm font-semibold text-slate-950 transition hover:bg-cyan-200 disabled:cursor-not-allowed disabled:bg-cyan-300/20 disabled:text-slate-400 sm:col-span-3"
          >
            保存模型
          </button>
        </form>

        <div className="overflow-hidden rounded-3xl border border-white/10">
          <div className="overflow-auto">
            <table className="min-w-[680px] text-left text-sm text-slate-100">
              <thead className="bg-white/[0.04] text-slate-300">
                <tr>
                  <th className="px-4 py-3 font-medium">模型</th>
                  <th className="px-4 py-3 font-medium">来源</th>
                  <th className="px-4 py-3 font-medium">状态</th>
                  <th className="px-4 py-3 font-medium">操作</th>
                </tr>
              </thead>
              <tbody className="divide-white/8 divide-y bg-black/10">
                {models.length > 0 ? (
                  models.map((item) => (
                    <tr key={String(item.id)} className="align-top">
                      <td className="px-4 py-4 font-mono text-cyan-100">
                        {item.model_id}
                        <div className="mt-1 text-xs text-slate-500">
                          {item.owned_by || '-'}
                        </div>
                      </td>
                      <td className="px-4 py-4 text-slate-300">
                        {item.source || '-'}
                      </td>
                      <td className="px-4 py-4">
                        <StatusBadge
                          active={!!item.enabled}
                          trueText="启用"
                          falseText="禁用"
                        />
                      </td>
                      <td className="px-4 py-4">
                        <button
                          type="button"
                          onClick={() => setModelEnabled(item.id, !item.enabled)}
                          disabled={loading}
                          className={`rounded-full px-4 py-2 text-xs font-semibold transition disabled:cursor-not-allowed disabled:opacity-60 ${
                            item.enabled
                              ? 'bg-rose-300 text-slate-950 hover:bg-rose-200'
                              : 'bg-emerald-300 text-slate-950 hover:bg-emerald-200'
                          }`}
                        >
                          {item.enabled ? '禁用' : '启用'}
                        </button>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td
                      colSpan={4}
                      className="px-4 py-10 text-center text-sm text-slate-400"
                    >
                      暂无模型
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </SurfacePanel>
  )

  const renderUsage = () => (
    <SurfacePanel className="p-5 sm:p-6">
      <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-slate-50">最近 usage</h2>
          <div className="mt-1 text-sm text-slate-400">
            24 小时内最近 {usageItems.length} 条 / 共 {usageTotal} 条。
          </div>
        </div>
      </div>
      {renderUsageTable(false)}
    </SurfacePanel>
  )

  return (
    <AppShell className="px-4 py-8 sm:px-6 sm:py-10">
      <div className="mx-auto w-full max-w-7xl space-y-6">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
          <div className="space-y-2">
            <div className="inline-flex rounded-full border border-cyan-300/30 bg-cyan-300/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.24em] text-cyan-100">
              OpenAI OAuth API Service
            </div>
            <h1 className="text-2xl font-semibold tracking-tight text-slate-50 sm:text-3xl">
              {currentConfig.title}
            </h1>
            <p className="max-w-3xl text-sm leading-6 text-slate-300 sm:text-base">
              {currentConfig.description}
            </p>
          </div>

          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              onClick={() => navigate('/admin-menu')}
              className="rounded-full border border-white/10 bg-white/[0.04] px-4 py-2 text-sm font-medium text-slate-100 transition hover:bg-white/[0.08]"
            >
              返回控制台
            </button>
            <button
              type="button"
              onClick={loadAll}
              disabled={loading}
              className="rounded-full bg-cyan-300 px-4 py-2 text-sm font-semibold text-slate-950 transition hover:bg-cyan-200 disabled:cursor-not-allowed disabled:bg-cyan-300/20 disabled:text-slate-400"
            >
              {loading ? '刷新中…' : '刷新'}
            </button>
          </div>
        </div>

        {errMsg ? (
          <div className="rounded-2xl border border-rose-400/40 bg-rose-500/10 px-4 py-3 text-sm text-rose-100">
            {errMsg}
          </div>
        ) : null}

        {currentView === 'keys' && newKey?.plain_key ? (
          <div className="rounded-2xl border border-amber-300/40 bg-amber-300/10 px-4 py-3 text-sm leading-6 text-amber-50">
            <div className="font-semibold">新 key 只显示一次</div>
            <div className="mt-2 break-all font-mono text-xs sm:text-sm">
              {newKey.plain_key}
            </div>
          </div>
        ) : null}

        {currentView === 'dashboard' ? renderDashboard() : null}
        {currentView === 'keys' ? renderKeys() : null}
        {currentView === 'models' ? renderModels() : null}
        {currentView === 'usage' ? renderUsage() : null}
      </div>
    </AppShell>
  )
}
