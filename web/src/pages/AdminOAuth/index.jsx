import React, { useEffect, useMemo, useState } from 'react'
import AdminFrame from '@/common/components/layout/AdminFrame'
import SurfacePanel from '@/common/components/layout/SurfacePanel'

const REQUIRED_ENV = [
  'OAUTH_API_OAUTH_ENABLED=true',
  'OAUTH_API_OAUTH_PROVIDER_NAME=OpenAI',
  'OAUTH_API_OAUTH_CLIENT_ID=...',
  'OAUTH_API_OAUTH_CLIENT_SECRET=...',
  'OAUTH_API_OAUTH_AUTH_URL=...',
  'OAUTH_API_OAUTH_TOKEN_URL=...',
  'OAUTH_API_OAUTH_USERINFO_URL=...',
  'OAUTH_API_OAUTH_REDIRECT_URL=http://localhost:8200/auth/oauth/callback',
  'OAUTH_API_OAUTH_SCOPES=openid,profile,email',
]

function ConfigRow({ label, value }) {
  return (
    <div className="rounded-lg border border-[#dde8df] bg-[#f7fbf8] px-4 py-3">
      <div className="text-xs uppercase tracking-[0.18em] text-[#7b8780]">
        {label}
      </div>
      <div className="mt-1 break-all text-sm font-medium text-[#1f2d25]">
        {value || '未配置'}
      </div>
    </div>
  )
}

export default function AdminOAuthPage() {
  const [config, setConfig] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    fetch('/auth/oauth/config', { headers: { Accept: 'application/json' } })
      .then(async (res) => {
        const contentType = res.headers.get('content-type') || ''
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        if (!contentType.includes('application/json')) {
          throw new Error('OAuth 配置接口未代理到后端')
        }
        return res.json()
      })
      .then((data) => {
        if (!cancelled) setConfig(data)
      })
      .catch((err) => {
        if (!cancelled) setError(err.message || '读取 OAuth 配置失败')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const loginUrl = config?.login_url || '/auth/oauth/start'
  const adminLoginUrl = `${loginUrl}?scope=admin&redirect=/admin-dashboard`
  const callbackUrl = useMemo(() => {
    const { protocol, host } = window.location
    return `${protocol}//${host.replace(/:\d+$/, ':8200')}/auth/oauth/callback`
  }, [])

  return (
    <AdminFrame
      title="OAuth/SSO 登录配置"
      description="这里用于检查管理员 SSO 登录状态。密钥仍通过后端环境变量注入，避免在前端暴露 client secret。"
      breadcrumb="基础资料 / OAuth 配置"
    >
      <SurfacePanel variant="admin" className="p-4 sm:p-6">
        <div className="space-y-5">
          {loading ? (
            <div className="text-sm text-[#7b8780]">正在读取 OAuth 配置...</div>
          ) : null}

          {error ? (
            <div className="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
              {error}。如果看到的是 Vite HTML 回退，请重启前端并确认 `/auth` 代理已生效。
            </div>
          ) : null}

          <div className="grid gap-3 sm:grid-cols-2">
            <ConfigRow
              label="启用状态"
              value={config?.enabled ? '已启用' : '未启用'}
            />
            <ConfigRow label="提供方" value={config?.provider_name || 'OAuth'} />
            <ConfigRow label="管理员登录入口" value={adminLoginUrl} />
            <ConfigRow label="回调地址" value={callbackUrl} />
          </div>

          <div className="rounded-lg border border-[#dde8df] bg-[#f7fbf8] p-4">
            <div className="text-sm font-semibold text-[#1f2d25]">
              启用所需环境变量
            </div>
            <pre className="mt-3 overflow-x-auto rounded-lg bg-[#1f2d25] p-4 text-xs leading-6 text-[#e7efe9]">
              {REQUIRED_ENV.join('\n')}
            </pre>
          </div>

          <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-800">
            OpenAI/ChatGPT 个人账号 OAuth token 不能作为上游 API 凭据。本系统只把 OAuth/OIDC 用作管理员登录身份源；OpenAI 上游仍必须配置官方 API key。
          </div>
        </div>
      </SurfacePanel>
    </AdminFrame>
  )
}
