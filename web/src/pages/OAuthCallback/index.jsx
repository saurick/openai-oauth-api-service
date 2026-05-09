import React, { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AppShell from '@/common/components/layout/AppShell'
import { AUTH_SCOPE, persistAuth } from '@/common/auth/auth'

function parseCallbackParams() {
  const hash = window.location.hash?.startsWith('#')
    ? window.location.hash.slice(1)
    : ''
  const search = window.location.search?.startsWith('?')
    ? window.location.search.slice(1)
    : ''
  return new URLSearchParams(hash || search)
}

function safeNextPath(value) {
  if (!value || !value.startsWith('/') || value.startsWith('//') || value.includes('\\')) {
    return '/admin-dashboard'
  }
  return value
}

export default function OAuthCallbackPage() {
  const navigate = useNavigate()
  const [message, setMessage] = useState('正在完成登录…')
  const params = useMemo(parseCallbackParams, [])

  useEffect(() => {
    const error = params.get('error')
    if (error) {
      setMessage('授权登录失败，请返回管理员登录页重试')
      window.history.replaceState(null, '', '/oauth/callback')
      window.setTimeout(() => navigate('/admin-login', { replace: true }), 800)
      return
    }

    const accessToken = params.get('access_token')
    if (!accessToken) {
      setMessage('授权结果无效，请返回管理员登录页重试')
      window.history.replaceState(null, '', '/oauth/callback')
      window.setTimeout(() => navigate('/admin-login', { replace: true }), 800)
      return
    }

    persistAuth(
      {
        access_token: accessToken,
        expires_at: params.get('expires_at'),
        token_type: params.get('token_type') || 'Bearer',
        user_id: params.get('user_id'),
        username: params.get('username'),
      },
      AUTH_SCOPE.ADMIN
    )

    const next = safeNextPath(params.get('next'))
    window.history.replaceState(null, '', '/oauth/callback')
    navigate(next, { replace: true })
  }, [navigate, params])

  return (
    <AppShell variant="adminLogin">
      <div className="flex min-h-screen items-center justify-center px-4">
        <div className="rounded-lg border border-[#d3e1dc] bg-white px-6 py-5 text-sm text-[#202422] shadow-[0_6px_24px_rgba(34,70,54,0.12)]">
          {message}
        </div>
      </div>
    </AppShell>
  )
}
