import React, { useEffect, useState } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import AppShell from '@/common/components/layout/AppShell'
import SurfacePanel from '@/common/components/layout/SurfacePanel'
import { AUTH_SCOPE, persistAuth } from '@/common/auth/auth'

export default function OAuthCallbackPage() {
  const location = useLocation()
  const navigate = useNavigate()
  const [error, setError] = useState('')

  useEffect(() => {
    try {
      const params = new URLSearchParams(location.hash.replace(/^#/, ''))
      const data = {
        access_token: params.get('access_token'),
        expires_at: params.get('expires_at'),
        token_type: params.get('token_type') || 'Bearer',
        user_id: params.get('user_id'),
        username: params.get('username'),
      }
      const scope =
        params.get('scope') === AUTH_SCOPE.ADMIN ? AUTH_SCOPE.ADMIN : AUTH_SCOPE.USER
      if (!data.access_token) {
        setError('授权登录返回缺少登录态，请重试')
        return
      }
      if (scope !== AUTH_SCOPE.ADMIN) {
        setError('当前前端只支持管理员授权登录')
        return
      }
      persistAuth(data, scope)
      navigate(params.get('redirect') || '/admin-dashboard', {
        replace: true,
      })
    } catch {
      setError('授权登录处理失败，请重试')
    }
  }, [location.hash, navigate])

  return (
    <AppShell className="flex items-center justify-center px-4 py-10">
      <div className="w-full max-w-[480px]">
        <SurfacePanel className="p-6 text-center">
          <div className="text-lg font-semibold text-slate-50">
            {error ? '授权登录失败' : '正在完成授权登录'}
          </div>
          <div className="mt-3 text-sm leading-6 text-slate-300">
            {error || '请稍候，正在写入本系统登录态。'}
          </div>
          {error ? (
            <Link
              to="/admin-login"
              className="mt-5 inline-flex rounded-2xl bg-cyan-300 px-4 py-2 text-sm font-semibold text-slate-950 transition hover:bg-cyan-200"
            >
              返回后台登录页
            </Link>
          ) : null}
        </SurfacePanel>
      </div>
    </AppShell>
  )
}
