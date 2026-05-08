import React, { useEffect, useMemo, useState } from 'react'
import AdminFrame from '@/common/components/layout/AdminFrame'
import SurfacePanel from '@/common/components/layout/SurfacePanel'
import { AUTH_SCOPE } from '@/common/auth/auth'
import { ADMIN_BASE_PATH } from '@/common/utils/adminRpc'
import { getActionErrorMessage } from '@/common/utils/errorMessage'
import { JsonRpc } from '@/common/utils/jsonRpc'

const PAGE_SIZE = 30
const DASHBOARD_USAGE_SIZE = 8
const DAY_SECONDS = 24 * 60 * 60
const tableWrapClass = 'overflow-hidden rounded-lg border border-[#dde8df]'
const tableClass = 'text-left text-sm text-[#1f2d25]'
const thClass =
  'whitespace-nowrap bg-[#f5fbf7] px-4 py-3 font-semibold text-[#66736b]'
const tdClass = 'px-4 py-4 text-[#1f2d25]'
const inputClass =
  'rounded-md border border-[#d6ded8] bg-white px-3 py-2.5 text-sm text-[#1f2d25] outline-none transition placeholder:text-[#9aa39e] focus:border-[#238a43] focus:ring-2 focus:ring-[#238a43]/15'
const fieldClass = 'grid gap-1.5 text-sm font-medium text-[#365141]'
const fieldHintClass = 'text-xs font-normal leading-5 text-[#7b8780]'
const primaryButtonClass =
  'rounded-md bg-[#238a43] px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-[#1d7538] disabled:cursor-not-allowed disabled:bg-[#cbd8d0] disabled:text-[#7b8780]'
const secondaryButtonClass =
  'rounded-md border border-[#cfd9d2] bg-white px-4 py-2.5 text-sm font-semibold text-[#365141] transition hover:border-[#238a43] hover:text-[#238a43] disabled:cursor-not-allowed disabled:opacity-60'
const dangerButtonClass =
  'rounded-md bg-rose-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-rose-700 disabled:cursor-not-allowed disabled:bg-rose-200 disabled:text-rose-700'
const tableActionButtonClass =
  'rounded-md bg-[#f5fbf7] px-3 py-2 text-xs font-semibold text-[#238a43] transition hover:bg-[#e7efe9] disabled:cursor-not-allowed disabled:opacity-60'

const MODEL_LIMIT_OPTIONS = [{ label: '允许全部模型', value: '' }]

const MODEL_ID_OPTIONS = ['gpt-5.4', 'gpt-5.5']

const INITIAL_KEY_FORM = {
  remark: '',
  allowedModels: '',
  tokenLimit: '',
}

const INITIAL_MODEL_FORM = {
  modelId: 'gpt-5.4',
  ownedBy: 'openai',
  enabled: true,
}

const INITIAL_USAGE_FILTERS = {
  keyId: '',
  model: '',
  success: '',
}

const VIEW_CONFIG = {
  dashboard: {
    title: '业务看板',
    description:
      '汇总 API 转发、Token 用量、费用估算和最近异常线索，不承载配置操作。',
  },
  keys: {
    title: 'API 凭据',
    description: '生成、搜索、启停和删除给客户端调用本服务使用的 ogw_ 凭据。',
  },
  models: {
    title: '模型管理',
    description: '维护 `/v1/models` 返回项，并控制请求是否允许使用对应模型。',
  },
  usage: {
    title: '调用明细',
    description:
      '查看 24 小时内最近请求，按凭据、模型和状态排查 Token、耗时与错误类型。',
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

function getModelOptions(models, currentValue = '') {
  const values = []
  for (const item of Array.isArray(models) ? models : []) {
    if (item?.model_id) values.push(item.model_id)
  }
  values.push(...MODEL_ID_OPTIONS)
  if (currentValue) values.push(currentValue)
  return Array.from(new Set(values.filter(Boolean)))
}

function SummaryCard({ label, value, sub }) {
  return (
    <SurfacePanel variant="admin" className="p-4">
      <div className="text-xs font-medium uppercase tracking-[0.18em] text-[#7b8780]">
        {label}
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
      className={`inline-flex rounded-full px-3 py-1 text-xs font-semibold ${
        active ? 'bg-emerald-50 text-emerald-700' : inactiveClass
      }`}
    >
      {active ? trueText : falseText}
    </span>
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
      className="shrink-0 rounded-md border border-[#cfd9d2] bg-white px-2.5 py-1.5 text-xs font-semibold text-[#365141] transition hover:border-[#238a43] hover:text-[#238a43] disabled:cursor-not-allowed disabled:opacity-60"
    >
      {copied ? '已复制' : label}
    </button>
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
  const [models, setModels] = useState([])
  const [modelTotal, setModelTotal] = useState(0)
  const [usageItems, setUsageItems] = useState([])
  const [usageTotal, setUsageTotal] = useState(0)
  const [newKey, setNewKey] = useState(null)
  const [editingKeyId, setEditingKeyId] = useState(null)
  const [keyForm, setKeyForm] = useState(INITIAL_KEY_FORM)
  const [modelForm, setModelForm] = useState(INITIAL_MODEL_FORM)
  const [keySearchInput, setKeySearchInput] = useState('')
  const [keySearch, setKeySearch] = useState('')
  const [selectedKeyIds, setSelectedKeyIds] = useState([])
  const [usageFilters, setUsageFilters] = useState(INITIAL_USAGE_FILTERS)
  const selectedKeyIdSet = useMemo(
    () => new Set(selectedKeyIds),
    [selectedKeyIds]
  )
  const selectedAllVisibleKeys =
    keys.length > 0 && keys.every((item) => selectedKeyIdSet.has(item.id))
  const modelOptions = useMemo(
    () => getModelOptions(models, keyForm.allowedModels),
    [keyForm.allowedModels, models]
  )

  const setKeyListState = (res) => {
    const items = Array.isArray(res?.data?.items) ? res.data.items : []
    setKeys(items)
    setKeyTotal(asInt(res?.data?.total, items.length))
    const itemIds = new Set(items.map((item) => item.id))
    setSelectedKeyIds((current) => current.filter((id) => itemIds.has(id)))
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

  const buildUsageParams = (startTime, filters = usageFilters) => {
    const params = {
      limit: PAGE_SIZE,
      offset: 0,
      start_time: startTime,
    }
    if (filters.keyId) params.key_id = asInt(filters.keyId, 0)
    if (filters.model) params.model = filters.model
    if (filters.success) params.success = filters.success === 'true'
    return params
  }

  const loadAll = async ({ usageFilterOverride } = {}) => {
    setLoading(true)
    setErrMsg('')
    try {
      const now = Math.floor(Date.now() / 1000)
      const startTime = now - DAY_SECONDS
      const activeUsageFilters = usageFilterOverride || usageFilters

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
        const [keysRes, modelsRes] = await Promise.all([
          apiRpc.call('key_list', {
            limit: 100,
            offset: 0,
            search: keySearch,
          }),
          apiRpc.call('model_list', { limit: 200, offset: 0 }),
        ])
        setKeyListState(keysRes)
        setModelListState(modelsRes)
        return
      }

      if (currentView === 'models') {
        setModelListState(
          await apiRpc.call('model_list', { limit: 200, offset: 0 })
        )
        return
      }

      const [keysRes, modelsRes, usageRes] = await Promise.all([
        apiRpc.call('key_list', { limit: 100, offset: 0 }),
        apiRpc.call('model_list', { limit: 200, offset: 0 }),
        apiRpc.call(
          'usage_list',
          buildUsageParams(startTime, activeUsageFilters)
        ),
      ])
      setKeyListState(keysRes)
      setModelListState(modelsRes)
      setUsageListState(usageRes)
    } catch (e) {
      setErrMsg(getActionErrorMessage(e, '加载 API 数据'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadAll()
  }, [currentView, keySearch])

  const saveKey = async (e) => {
    e.preventDefault()
    setErrMsg('')
    setNewKey(null)
    try {
      if (editingKeyId) {
        const currentKey = keys.find((item) => item.id === editingKeyId)
        await apiRpc.call('key_update', {
          key_id: editingKeyId,
          name: keyForm.remark.trim(),
          quota_total_tokens: asInt(keyForm.tokenLimit, 0),
          allowed_models: splitModels(keyForm.allowedModels),
          disabled: !!currentKey?.disabled,
        })
      } else {
        const result = await apiRpc.call('key_create', {
          name: keyForm.remark.trim(),
          quota_total_tokens: asInt(keyForm.tokenLimit, 0),
          allowed_models: splitModels(keyForm.allowedModels),
        })
        setNewKey(result?.data || null)
      }
      setKeyForm(INITIAL_KEY_FORM)
      setEditingKeyId(null)
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

  const startEditKey = (item) => {
    setNewKey(null)
    setEditingKeyId(item.id)
    setKeyForm({
      remark: item.name || '',
      allowedModels:
        Array.isArray(item.allowed_models) && item.allowed_models.length > 0
          ? item.allowed_models[0]
          : '',
      tokenLimit:
        asInt(item.quota_total_tokens, 0) > 0
          ? String(asInt(item.quota_total_tokens, 0))
          : '',
    })
  }

  const cancelEditKey = () => {
    setEditingKeyId(null)
    setKeyForm(INITIAL_KEY_FORM)
  }

  const submitKeySearch = async (e) => {
    e.preventDefault()
    setErrMsg('')
    setSelectedKeyIds([])
    const nextSearch = keySearchInput.trim()
    if (nextSearch === keySearch) {
      await loadAll()
      return
    }
    setKeySearch(nextSearch)
  }

  const clearKeySearch = () => {
    setErrMsg('')
    setSelectedKeyIds([])
    setKeySearchInput('')
    setKeySearch('')
  }

  const deleteKey = async (item) => {
    const keyId = item?.id
    const label = item?.name || item?.key_prefix || `ID ${keyId}`
    // eslint-disable-next-line no-alert
    const confirmed = window.confirm(
      `确认删除 API 凭据「${label}」吗？删除后不可恢复，历史调用记录会保留。`
    )
    if (!confirmed) {
      return
    }
    setErrMsg('')
    try {
      await apiRpc.call('key_delete', { key_id: keyId })
      if (editingKeyId === keyId) cancelEditKey()
      setSelectedKeyIds((current) => current.filter((id) => id !== keyId))
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '删除 API key'))
    }
  }

  const deleteSelectedKeys = async () => {
    if (selectedKeyIds.length === 0) return
    // eslint-disable-next-line no-alert
    const confirmed = window.confirm(
      `确认删除选中的 ${selectedKeyIds.length} 个 API 凭据吗？删除后不可恢复，历史调用记录会保留。`
    )
    if (!confirmed) {
      return
    }
    setErrMsg('')
    try {
      await apiRpc.call('key_delete_batch', { key_ids: selectedKeyIds })
      if (selectedKeyIds.includes(editingKeyId)) cancelEditKey()
      setSelectedKeyIds([])
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '批量删除 API key'))
    }
  }

  const toggleKeySelection = (keyId, checked) => {
    setSelectedKeyIds((current) => {
      if (checked) {
        return current.includes(keyId) ? current : [...current, keyId]
      }
      return current.filter((id) => id !== keyId)
    })
  }

  const toggleAllVisibleKeys = (checked) => {
    const visibleIds = keys.map((item) => item.id)
    setSelectedKeyIds((current) => {
      if (checked) {
        return Array.from(new Set([...current, ...visibleIds]))
      }
      return current.filter((id) => !visibleIds.includes(id))
    })
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

  const saveModel = async (e) => {
    e.preventDefault()
    const modelId = modelForm.modelId.trim()
    if (!modelId) {
      setErrMsg('请输入模型 ID')
      return
    }

    setErrMsg('')
    try {
      await apiRpc.call('model_upsert', {
        model_id: modelId,
        owned_by: modelForm.ownedBy.trim() || 'openai',
        enabled: modelForm.enabled,
      })
      setModelForm(INITIAL_MODEL_FORM)
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '保存模型'))
    }
  }

  const editModel = (item) => {
    setModelForm({
      modelId: item.model_id || '',
      ownedBy: item.owned_by || 'openai',
      enabled: !!item.enabled,
    })
  }

  const deleteModel = async (item) => {
    // eslint-disable-next-line no-alert
    const confirmed = window.confirm(
      `确认删除模型 ${item.model_id}？删除后会同步清理相关策略、价格和 key 模型限制。`
    )
    if (!confirmed) {
      return
    }
    setErrMsg('')
    try {
      await apiRpc.call('model_delete', { id: item.id })
      await loadAll()
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '删除模型'))
    }
  }

  const renderUsageTable = (compact = false) => (
    <div className={tableWrapClass}>
      <div className="overflow-auto">
        <table
          className={`${tableClass} ${compact ? 'min-w-[820px]' : 'min-w-[1080px]'}`}
        >
          <thead>
            <tr>
              <th className={thClass}>时间</th>
              <th className={thClass}>凭据</th>
              <th className={thClass}>接口</th>
              <th className={thClass}>模型</th>
              <th className={thClass}>状态</th>
              <th className={thClass}>Token</th>
              {!compact ? <th className={thClass}>耗时</th> : null}
              {!compact ? <th className={thClass}>错误</th> : null}
            </tr>
          </thead>
          <tbody className="divide-y divide-[#e7efe9] bg-white">
            {usageItems.length > 0 ? (
              usageItems.map((item) => (
                <tr key={String(item.id)} className="align-top">
                  <td className={tdClass}>{fmtTs(item.created_at)}</td>
                  <td className={`${tdClass} font-mono text-xs`}>
                    {item.api_key_prefix || '-'}
                  </td>
                  <td className={tdClass}>{item.endpoint || item.path}</td>
                  <td className={`${tdClass} font-mono text-xs`}>
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
                  {!compact ? (
                    <td className={tdClass}>
                      {fmtNumber(item.duration_ms)} ms
                    </td>
                  ) : null}
                  {!compact ? (
                    <td className={tdClass}>{item.error_type || '-'}</td>
                  ) : null}
                </tr>
              ))
            ) : (
              <tr>
                <td
                  colSpan={compact ? 6 : 8}
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
            sub={`${fmtNumber(summary.success_requests)} 成功 / ${fmtNumber(summary.failed_requests)} 失败`}
          />
          <SummaryCard
            label="24h Token"
            value={fmtNumber(summary.total_tokens)}
            sub={`${fmtNumber(summary.input_tokens)} 输入 / ${fmtNumber(summary.output_tokens)} 输出`}
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

  const renderKeys = () => (
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

        <form
          onSubmit={saveKey}
          className="grid gap-3 md:grid-cols-2 xl:grid-cols-3"
        >
          <label className={fieldClass}>
            备注名称
            <input
              value={keyForm.remark}
              onChange={(e) =>
                setKeyForm((current) => ({
                  ...current,
                  remark: e.target.value,
                }))
              }
              className={inputClass}
              placeholder="例如内部测试 key"
            />
            <span className={fieldHintClass}>留空时后端会生成默认备注。</span>
          </label>
          <label className={fieldClass}>
            允许模型
            <select
              value={keyForm.allowedModels}
              onChange={(e) =>
                setKeyForm((current) => ({
                  ...current,
                  allowedModels: e.target.value,
                }))
              }
              className={inputClass}
            >
              {MODEL_LIMIT_OPTIONS.map((option) => (
                <option key={option.label} value={option.value}>
                  {option.label}
                </option>
              ))}
              {modelOptions.map((modelId) => (
                <option key={modelId} value={modelId}>
                  仅允许 {modelId}
                </option>
              ))}
            </select>
            <span className={fieldHintClass}>
              选项来自模型管理页，避免填入不存在的模型。
            </span>
          </label>
          <label className={fieldClass}>
            Token 总额度
            <input
              type="number"
              min="0"
              step="1"
              value={keyForm.tokenLimit}
              onChange={(e) =>
                setKeyForm((current) => ({
                  ...current,
                  tokenLimit: e.target.value,
                }))
              }
              className={inputClass}
              placeholder="0 表示不限"
            />
            <span className={fieldHintClass}>
              达到额度后，该凭据的转发请求会返回 429。
            </span>
          </label>
          <button
            type="submit"
            disabled={loading}
            className={primaryButtonClass}
          >
            {editingKeyId ? '保存凭据' : '生成 API 凭据'}
          </button>
          {editingKeyId ? (
            <button
              type="button"
              onClick={cancelEditKey}
              className="rounded-md border border-[#d6ded8] bg-white px-4 py-2.5 text-sm font-semibold text-[#1f2d25] transition hover:border-[#238a43] hover:text-[#238a43]"
            >
              取消编辑
            </button>
          ) : null}
        </form>

        <div className="flex flex-col gap-3 rounded-lg border border-[#e7efe9] bg-[#fbfdfb] p-3 lg:flex-row lg:items-center lg:justify-between">
          <form
            onSubmit={submitKeySearch}
            className="flex flex-1 flex-col gap-2 sm:flex-row"
          >
            <label className="sr-only" htmlFor="key-search">
              搜索凭据
            </label>
            <input
              id="key-search"
              value={keySearchInput}
              onChange={(e) => setKeySearchInput(e.target.value)}
              className={`${inputClass} flex-1`}
              placeholder="搜索备注、完整凭据、前缀或后四位"
            />
            <button
              type="submit"
              disabled={loading}
              className={primaryButtonClass}
            >
              搜索
            </button>
            {keySearch ? (
              <button
                type="button"
                onClick={clearKeySearch}
                disabled={loading}
                className={secondaryButtonClass}
              >
                清空
              </button>
            ) : null}
          </form>
          <button
            type="button"
            onClick={deleteSelectedKeys}
            disabled={loading || selectedKeyIds.length === 0}
            className={dangerButtonClass}
          >
            批量删除
            {selectedKeyIds.length > 0 ? `（${selectedKeyIds.length}）` : ''}
          </button>
        </div>

        <div className={tableWrapClass}>
          <div className="overflow-auto">
            <table className={`${tableClass} min-w-[940px]`}>
              <thead>
                <tr>
                  <th className={thClass}>
                    <input
                      type="checkbox"
                      checked={selectedAllVisibleKeys}
                      onChange={(e) => toggleAllVisibleKeys(e.target.checked)}
                      aria-label="选择当前列表所有 key"
                      className="h-4 w-4 rounded border-[#cfd9d2] text-[#238a43] focus:ring-[#238a43]"
                    />
                  </th>
                  <th className={thClass}>备注</th>
                  <th className={thClass}>完整凭据</th>
                  <th className={thClass}>模型限制</th>
                  <th className={thClass}>Token 限制</th>
                  <th className={thClass}>状态</th>
                  <th className={thClass}>操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#e7efe9] bg-white">
                {keys.length > 0 ? (
                  keys.map((item) => (
                    <tr key={String(item.id)} className="align-top">
                      <td className={tdClass}>
                        <input
                          type="checkbox"
                          checked={selectedKeyIdSet.has(item.id)}
                          onChange={(e) =>
                            toggleKeySelection(item.id, e.target.checked)
                          }
                          aria-label={`选择 ${item.name || item.key_prefix || item.id}`}
                          className="h-4 w-4 rounded border-[#cfd9d2] text-[#238a43] focus:ring-[#238a43]"
                        />
                      </td>
                      <td className={`${tdClass} font-medium`}>
                        {item.name || '无备注'}
                        <div className="mt-1 text-xs text-[#9aa39e]">
                          最近使用：{fmtTs(item.last_used_at)}
                        </div>
                      </td>
                      <td className={`${tdClass} font-mono text-xs`}>
                        <div className="flex max-w-[360px] items-start gap-2">
                          <span className="min-w-0 break-all">
                            {item.plain_key ||
                              `${item.key_prefix}…${item.key_last4}`}
                          </span>
                          {item.plain_key ? (
                            <CopyButton value={item.plain_key} />
                          ) : null}
                        </div>
                      </td>
                      <td className={tdClass}>
                        {Array.isArray(item.allowed_models) &&
                        item.allowed_models.length > 0
                          ? item.allowed_models.join(', ')
                          : '不限'}
                      </td>
                      <td className={tdClass}>
                        {asInt(item.quota_total_tokens, 0) > 0
                          ? fmtNumber(item.quota_total_tokens)
                          : '不限'}
                      </td>
                      <td className={tdClass}>
                        <StatusBadge
                          active={!item.disabled}
                          trueText="启用"
                          falseText="禁用"
                        />
                      </td>
                      <td className={tdClass}>
                        <div className="flex flex-wrap gap-2">
                          <button
                            type="button"
                            onClick={() => startEditKey(item)}
                            disabled={loading}
                            className={tableActionButtonClass}
                          >
                            编辑
                          </button>
                          <button
                            type="button"
                            onClick={() =>
                              setKeyDisabled(item.id, !item.disabled)
                            }
                            disabled={loading}
                            className={`rounded-md px-4 py-2 text-xs font-semibold transition disabled:cursor-not-allowed disabled:opacity-60 ${
                              item.disabled
                                ? 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100'
                                : 'bg-rose-50 text-rose-700 hover:bg-rose-100'
                            }`}
                          >
                            {item.disabled ? '启用' : '禁用'}
                          </button>
                          <button
                            type="button"
                            onClick={() => deleteKey(item)}
                            disabled={loading}
                            className="rounded-md bg-rose-50 px-4 py-2 text-xs font-semibold text-rose-700 transition hover:bg-rose-100 disabled:cursor-not-allowed disabled:opacity-60"
                          >
                            删除
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td
                      colSpan={6}
                      className="px-4 py-10 text-center text-sm text-[#9aa39e]"
                    >
                      {keySearch ? '没有匹配的 API 凭据' : '暂无 API 凭据'}
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
    <SurfacePanel variant="admin" className="p-5 sm:p-6">
      <div className="space-y-5">
        <div>
          <h2 className="text-lg font-semibold text-[#1f2d25]">模型管理</h2>
          <div className="mt-1 text-sm text-[#7b8780]">
            共 {modelTotal} 个；列表会进入 `/v1/models`
            返回项，并参与请求启停校验。
          </div>
        </div>

        <form
          onSubmit={saveModel}
          className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_220px_auto_auto]"
        >
          <label className={fieldClass}>
            模型 ID
            <input
              value={modelForm.modelId}
              onChange={(e) =>
                setModelForm((current) => ({
                  ...current,
                  modelId: e.target.value,
                }))
              }
              className={inputClass}
              placeholder="例如 gpt-5.5"
            />
          </label>
          <label className={fieldClass}>
            归属方
            <input
              value={modelForm.ownedBy}
              onChange={(e) =>
                setModelForm((current) => ({
                  ...current,
                  ownedBy: e.target.value,
                }))
              }
              className={inputClass}
              placeholder="openai"
            />
          </label>
          <label className="flex items-center gap-2 rounded-md border border-[#d6ded8] bg-white px-3 py-2.5 text-sm font-medium text-[#1f2d25] lg:self-end">
            <input
              type="checkbox"
              checked={modelForm.enabled}
              onChange={(e) =>
                setModelForm((current) => ({
                  ...current,
                  enabled: e.target.checked,
                }))
              }
              className="h-4 w-4 rounded border-[#c8d4cc] text-[#238a43] focus:ring-[#238a43]"
            />
            默认启用
          </label>
          <button
            type="submit"
            disabled={loading || !modelForm.modelId.trim()}
            className={`${primaryButtonClass} lg:self-end`}
          >
            保存模型
          </button>
        </form>

        <div className={tableWrapClass}>
          <div className="overflow-auto">
            <table className={`${tableClass} min-w-[680px]`}>
              <thead>
                <tr>
                  <th className={thClass}>模型</th>
                  <th className={thClass}>来源</th>
                  <th className={thClass}>状态</th>
                  <th className={thClass}>操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[#e7efe9] bg-white">
                {models.length > 0 ? (
                  models.map((item) => (
                    <tr key={String(item.id)} className="align-top">
                      <td className={`${tdClass} font-mono`}>
                        {item.model_id}
                        <div className="mt-1 text-xs text-[#9aa39e]">
                          {item.owned_by || '-'}
                        </div>
                      </td>
                      <td className={tdClass}>{item.source || '-'}</td>
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
                            onClick={() => editModel(item)}
                            disabled={loading}
                            className={tableActionButtonClass}
                          >
                            编辑
                          </button>
                          <button
                            type="button"
                            onClick={() =>
                              setModelEnabled(item.id, !item.enabled)
                            }
                            disabled={loading}
                            className={`rounded-md px-4 py-2 text-xs font-semibold transition disabled:cursor-not-allowed disabled:opacity-60 ${
                              item.enabled
                                ? 'bg-rose-50 text-rose-700 hover:bg-rose-100'
                                : 'bg-emerald-50 text-emerald-700 hover:bg-emerald-100'
                            }`}
                          >
                            {item.enabled ? '禁用' : '启用'}
                          </button>
                          <button
                            type="button"
                            onClick={() => deleteModel(item)}
                            disabled={loading}
                            className="rounded-md bg-rose-50 px-4 py-2 text-xs font-semibold text-rose-700 transition hover:bg-rose-100 disabled:cursor-not-allowed disabled:opacity-60"
                          >
                            删除
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td
                      colSpan={4}
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
      </div>
    </SurfacePanel>
  )

  const renderUsage = () => (
    <SurfacePanel variant="admin" className="p-5 sm:p-6">
      <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-[#1f2d25]">最近调用</h2>
          <div className="mt-1 text-sm text-[#7b8780]">
            24 小时内最近 {usageItems.length} 条 / 共 {usageTotal} 条。
          </div>
        </div>
      </div>
      <form
        onSubmit={(e) => {
          e.preventDefault()
          loadAll({ usageFilterOverride: usageFilters })
        }}
        className="mb-5 grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_180px_auto_auto]"
      >
        <label className={fieldClass}>
          调用凭据
          <select
            value={usageFilters.keyId}
            onChange={(e) =>
              setUsageFilters((current) => ({
                ...current,
                keyId: e.target.value,
              }))
            }
            className={inputClass}
          >
            <option value="">全部凭据</option>
            {keys.map((item) => (
              <option key={item.id} value={item.id}>
                {item.name || item.key_prefix || `凭据 ${item.id}`}
              </option>
            ))}
          </select>
        </label>
        <label className={fieldClass}>
          请求模型
          <select
            value={usageFilters.model}
            onChange={(e) =>
              setUsageFilters((current) => ({
                ...current,
                model: e.target.value,
              }))
            }
            className={inputClass}
          >
            <option value="">全部模型</option>
            {modelOptions.map((modelId) => (
              <option key={modelId} value={modelId}>
                {modelId}
              </option>
            ))}
          </select>
        </label>
        <label className={fieldClass}>
          请求状态
          <select
            value={usageFilters.success}
            onChange={(e) =>
              setUsageFilters((current) => ({
                ...current,
                success: e.target.value,
              }))
            }
            className={inputClass}
          >
            <option value="">全部状态</option>
            <option value="true">成功</option>
            <option value="false">失败</option>
          </select>
        </label>
        <button
          type="submit"
          disabled={loading}
          className={`${primaryButtonClass} lg:self-end`}
        >
          应用筛选
        </button>
        <button
          type="button"
          onClick={() => {
            setUsageFilters(INITIAL_USAGE_FILTERS)
            loadAll({ usageFilterOverride: INITIAL_USAGE_FILTERS })
          }}
          disabled={loading}
          className={`${secondaryButtonClass} lg:self-end`}
        >
          重置
        </button>
      </form>
      {renderUsageTable(false)}
    </SurfacePanel>
  )

  return (
    <AdminFrame
      breadcrumb={`配置管理 / ${currentConfig.title}`}
      title={currentConfig.title}
      description={currentConfig.description}
      actions={
        <button
          type="button"
          onClick={loadAll}
          disabled={loading}
          className="rounded-md bg-[#238a43] px-3 py-1.5 text-sm font-semibold text-white transition hover:bg-[#1d7538] disabled:cursor-not-allowed disabled:bg-[#cbd8d0] disabled:text-[#7b8780]"
        >
          {loading ? '刷新中...' : '刷新'}
        </button>
      }
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
            <div>完整 key 已保存，后续可在列表继续查看。</div>
            <div className="mt-2 flex flex-col gap-2 sm:flex-row sm:items-start">
              <div className="min-w-0 flex-1 break-all font-mono text-xs text-[#1f2d25] sm:text-sm">
                {newKey.plain_key}
              </div>
              <CopyButton value={newKey.plain_key} label="复制完整凭据" />
            </div>
          </div>
        ) : null}

        {currentView === 'dashboard' ? renderDashboard() : null}
        {currentView === 'keys' ? renderKeys() : null}
        {currentView === 'models' ? renderModels() : null}
        {currentView === 'usage' ? renderUsage() : null}
      </div>
    </AdminFrame>
  )
}
