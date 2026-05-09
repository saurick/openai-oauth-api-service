import React, { useEffect, useMemo, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import AppShell from '@/common/components/layout/AppShell'
import AdminThemeToggle from '@/common/components/layout/AdminThemeToggle'
import { AUTH_SCOPE, persistAuth } from '@/common/auth/auth'
import { ADMIN_BASE_PATH } from '@/common/utils/adminRpc'
import { getActionErrorMessage } from '@/common/utils/errorMessage'
import { JsonRpc } from '@/common/utils/jsonRpc'

export default function AdminLoginPage() {
  const navigate = useNavigate()
  const location = useLocation()

  const from =
    (location.state?.from?.pathname || '/admin-dashboard') +
    (location.state?.from?.search || '') +
    (location.state?.from?.hash || '')

  const authRpc = useMemo(
    () =>
      new JsonRpc({
        url: 'auth',
        basePath: ADMIN_BASE_PATH,
        authScope: AUTH_SCOPE.ADMIN,
      }),
    []
  )

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [errMsg, setErrMsg] = useState('')
  const [oauthConfig, setOauthConfig] = useState({ enabled: false, provider: '' })

  const canSubmit = useMemo(
    () => username.trim().length > 0 && password.length > 0 && !submitting,
    [username, password, submitting]
  )

  const onSubmit = async (e) => {
    e.preventDefault()
    if (!canSubmit) return

    setErrMsg('')
    setSubmitting(true)

    try {
      const result = await authRpc.call('admin_login', {
        username: username.trim(),
        password,
      })

      persistAuth(result?.data, AUTH_SCOPE.ADMIN)
      navigate(from, { replace: true })
    } catch (err) {
      setErrMsg(getActionErrorMessage(err, '登录'))
    } finally {
      setSubmitting(false)
    }
  }

  useEffect(() => {
    let cancelled = false
    fetch('/auth/oauth/config')
      .then((response) => (response.ok ? response.json() : null))
      .then((data) => {
        if (!cancelled && data?.enabled) {
          setOauthConfig({
            enabled: true,
            provider: data.provider || 'oauth',
          })
        }
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [])

  const startOAuthLogin = () => {
    const query = new URLSearchParams({
      frontend_origin: window.location.origin,
      next: from,
    })
    window.location.assign(`/auth/oauth/start?${query.toString()}`)
  }

  return (
    <AppShell variant="adminLogin">
      <div className="absolute right-4 top-4 z-10 sm:right-6 sm:top-6">
        <AdminThemeToggle />
      </div>
      <div className="flex min-h-screen items-center justify-center px-4 py-8 sm:px-6">
        <div className="w-full max-w-[660px] rounded-lg border border-[#d3e1dc] bg-white px-6 py-7 shadow-[0_6px_24px_rgba(34,70,54,0.12)] sm:px-8 md:px-10">
          <div className="mb-7">
            <div className="flex items-end gap-3">
              <div className="text-[56px] font-extrabold leading-[0.85] tracking-tight text-[#173b59] sm:text-[70px]">
                OA<span className="text-[#d6a23a]">S</span>
              </div>
              <div className="pb-1">
                <div className="text-[34px] font-extrabold leading-none tracking-tight text-[#173b59] sm:text-[44px]">
                  API SERVICE
                </div>
                <div className="mt-1 text-sm font-extrabold uppercase tracking-[0.03em] text-[#173b59] sm:text-lg">
                  OpenAI OAuth API Service
                </div>
              </div>
            </div>
            <h1 className="mt-6 text-2xl font-bold text-[#202422]">
              OAuth API 管理后台
            </h1>
          </div>

          <form onSubmit={onSubmit} className="space-y-6">
            <div>
              <label className="mb-2 block text-sm text-[#202422]">
                <span className="text-red-500">*</span> 管理员账号
              </label>
              <input
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                className="h-11 w-full rounded-lg border border-[#d6d9d8] bg-white px-4 text-base text-[#202422] outline-none transition placeholder:text-[#a5aaa8] focus:border-[#2f934d] focus:ring-2 focus:ring-[#2f934d]/15"
                placeholder="请输入账号"
              />
            </div>

            <div>
              <label className="mb-2 block text-sm text-[#202422]">
                <span className="text-red-500">*</span> 密码
              </label>
              <div className="relative">
                <input
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="current-password"
                  className="h-11 w-full rounded-lg border border-[#d6d9d8] bg-white px-4 pr-11 text-base text-[#202422] outline-none transition placeholder:text-[#a5aaa8] focus:border-[#2f934d] focus:ring-2 focus:ring-[#2f934d]/15"
                  placeholder="请输入密码"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((current) => !current)}
                  className="absolute right-3 top-1/2 flex h-7 w-7 -translate-y-1/2 items-center justify-center rounded-md text-[#8f9692] transition hover:bg-[#f0f4f1] hover:text-[#2f934d]"
                  aria-label={showPassword ? '隐藏密码' : '显示密码'}
                >
                  {showPassword ? <EyeIcon /> : <EyeOffIcon />}
                </button>
              </div>
            </div>

            {errMsg ? (
              <div className="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700">
                {errMsg}
              </div>
            ) : null}

            <button
              type="submit"
              aria-label="登录"
              disabled={!canSubmit}
              className={`h-11 w-full rounded-lg text-base font-semibold tracking-wide shadow-[inset_0_-2px_0_rgba(0,0,0,0.12)] transition ${
                canSubmit
                  ? 'bg-[#2d9047] text-white hover:bg-[#267f3d] active:bg-[#206e35]'
                  : 'cursor-not-allowed bg-[#2d9047] text-white opacity-75'
              }`}
            >
              {submitting ? '登录中…' : '登 录'}
            </button>

            {oauthConfig.enabled ? (
              <button
                type="button"
                onClick={startOAuthLogin}
                className="h-11 w-full rounded-lg border border-[#d6d9d8] bg-white text-base font-semibold text-[#173b59] transition hover:border-[#2f934d] hover:text-[#2f934d]"
              >
                使用 {oauthConfig.provider === 'google' ? 'Google' : 'OAuth'} 登录
              </button>
            ) : null}

          </form>
        </div>
      </div>
    </AppShell>
  )
}

function EyeIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path
        d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Z"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2"
      />
      <path
        d="M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
      />
    </svg>
  )
}

function EyeOffIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path
        d="m3 3 18 18M10.6 10.7A3 3 0 0 0 13.3 13.4M9.9 5.4A9.7 9.7 0 0 1 12 5c6 0 9.5 7 9.5 7a16.2 16.2 0 0 1-2.7 3.5M6.6 6.7C4 8.5 2.5 12 2.5 12s3.5 7 9.5 7c1.3 0 2.5-.3 3.6-.8"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2"
      />
    </svg>
  )
}
