import React, { useMemo, useState } from 'react'
import AdminFrame from '@/common/components/layout/AdminFrame'
import SurfacePanel from '@/common/components/layout/SurfacePanel'
import {
  CLIENT_CONFIG_DEFAULTS,
  CLIENT_CONFIG_OS_OPTIONS,
  CLIENT_CONFIG_TOOL_OPTIONS,
  getClientConfigFilename,
  getClientConfigInstallPath,
  normalizeApiKey,
  normalizeBaseUrl,
  normalizeProfile,
  renderClientConfigTemplate,
} from '@/common/utils/clientConfigTemplates'

const inputClass =
  'rounded-md border border-[#d6ded8] bg-white px-3 py-2.5 text-sm text-[#1f2d25] outline-none transition placeholder:text-[#9aa39e] focus:border-[#238a43] focus:ring-2 focus:ring-[#238a43]/15'
const fieldClass = 'grid gap-1.5 text-sm font-medium text-[#365141]'
const hintClass = 'text-xs font-normal leading-5 text-[#7b8780]'
const primaryButtonClass = 'admin-button admin-button-primary'
const secondaryButtonClass = 'admin-button admin-button-default'

export default function AdminClientConfigPage() {
  const [tool, setTool] = useState('codex')
  const [os, setOs] = useState('mac')
  const [baseUrl, setBaseUrl] = useState(CLIENT_CONFIG_DEFAULTS.baseUrl)
  const [apiKey, setApiKey] = useState(CLIENT_CONFIG_DEFAULTS.apiKey)
  const [profile, setProfile] = useState(CLIENT_CONFIG_DEFAULTS.profile)
  const [copyStatus, setCopyStatus] = useState('')

  const renderValues = useMemo(
    () => ({
      apiKey: normalizeApiKey(apiKey),
      baseUrl: normalizeBaseUrl(baseUrl),
      os,
      profile: normalizeProfile(profile),
      tool,
    }),
    [apiKey, baseUrl, os, profile, tool]
  )

  const templateContent = useMemo(
    () => renderClientConfigTemplate(renderValues),
    [renderValues]
  )
  const filename = getClientConfigFilename(tool, os)
  const installPath = getClientConfigInstallPath(tool, os)

  const handleDownload = () => {
    downloadTextFile(templateContent, filename)
  }

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(templateContent)
      setCopyStatus('已复制配置内容')
    } catch (error) {
      console.warn('复制配置失败', error)
      setCopyStatus('复制失败，请手动选中复制')
    }
  }

  return (
    <AdminFrame
      title="客户端配置模板"
      description="导出 Codex 和 opencode 的 macOS / Windows 最小配置模板，只保留 Base URL、API Key 和 Codex profile 等必要字段；历史记录、工作区信任、通知路径等个人字段不写入模板。"
      breadcrumb="转发配置 / 客户端配置模板"
    >
      <SurfacePanel variant="admin" className="p-5">
        <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.8fr)]">
          <div className="grid gap-4">
            <div className="grid gap-3 sm:grid-cols-2">
              <ToggleGroup label="客户端" value={tool} options={CLIENT_CONFIG_TOOL_OPTIONS} onChange={setTool} />
              <ToggleGroup label="系统" value={os} options={CLIENT_CONFIG_OS_OPTIONS} onChange={setOs} />
            </div>

            <div className="grid gap-3 lg:grid-cols-3">
              <label className={fieldClass}>
                Base URL
                <input
                  className={inputClass}
                  value={baseUrl}
                  onChange={(event) => setBaseUrl(event.target.value)}
                  placeholder="https://example.com/v1"
                />
                <span className={hintClass}>会写入 Codex base_url 或 opencode baseURL。</span>
              </label>
              <label className={fieldClass}>
                API Key
                <input
                  className={inputClass}
                  value={apiKey}
                  onChange={(event) => setApiKey(event.target.value)}
                  placeholder="ogw_xxx 或 sk-xxx"
                />
                <span className={hintClass}>下载前替换；不需要把真实 key 固化进仓库。</span>
              </label>
              <label className={fieldClass}>
                Codex profile
                <input
                  className={inputClass}
                  value={profile}
                  onChange={(event) => setProfile(event.target.value)}
                  disabled={tool !== 'codex'}
                  placeholder="saurick"
                />
                <span className={hintClass}>仅 Codex 使用；opencode 通过 agent model 选择 provider。</span>
              </label>
            </div>
          </div>

          <div className="rounded-lg border border-[#dde8df] bg-[#fbfdfb] p-4">
            <h2 className="text-base font-semibold text-[#1f2d25]">替换与安装教程</h2>
            <ol className="mt-3 list-decimal space-y-2 pl-5 text-sm leading-6 text-[#365141]">
              <li>在本页填入目标服务 <strong>Base URL</strong>、<strong>API Key</strong>{tool === 'codex' ? '，再确认 profile。' : '。'}</li>
              <li>点击下载，把模板保存到本机。</li>
              <li>在新电脑安装 {tool === 'codex' ? 'Codex' : 'opencode'}，先运行一次让它创建配置目录。</li>
              <li>备份旧文件，再把下载文件改名并放到：<code>{installPath}</code></li>
              <li>{tool === 'codex' ? `执行 codex --profile ${renderValues.profile} 验证。` : '执行 opencode 并选择 build / plan agent 验证。'}</li>
            </ol>
            <div className="mt-4 rounded-md border border-[#f0c868] bg-[#fff8df] px-3 py-2 text-xs leading-5 text-[#c07a00]">
              不会导出 Codex 的 auth.json、历史会话、projects 信任记录、opencode secrets 目录或本机绝对路径；这些属于个人状态，应在目标机器重新生成。
            </div>
          </div>
        </div>
      </SurfacePanel>

      <SurfacePanel variant="admin" className="overflow-hidden">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[#dde8df] px-5 py-4">
          <div>
            <h2 className="text-base font-semibold text-[#1f2d25]">配置预览</h2>
            <p className="mt-1 text-sm text-[#7b8780]">{filename} · 安装路径 {installPath}</p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {copyStatus ? <span className="text-xs text-[#7b8780]">{copyStatus}</span> : null}
            <button type="button" className={secondaryButtonClass} onClick={handleCopy}>
              复制内容
            </button>
            <button type="button" className={primaryButtonClass} onClick={handleDownload}>
              下载配置
            </button>
          </div>
        </div>
        <pre className="admin-code-preview max-h-[560px] overflow-auto p-5 text-xs leading-5"><code>{templateContent}</code></pre>
      </SurfacePanel>
    </AdminFrame>
  )
}

function ToggleGroup({ label, value, options, onChange }) {
  return (
    <div className="grid gap-1.5 text-sm font-medium text-[#365141]">
      <span>{label}</span>
      <div className="admin-view-tabs" role="tablist" aria-label={label}>
        {options.map((option) => (
          <button
            key={option.value}
            type="button"
            role="tab"
            aria-selected={value === option.value}
            className="admin-view-tab"
            onClick={() => onChange(option.value)}
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  )
}

function downloadTextFile(content, filename) {
  const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
