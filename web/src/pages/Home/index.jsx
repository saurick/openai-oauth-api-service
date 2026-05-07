import React from 'react'
import { Link, useNavigate } from 'react-router-dom'
import AppShell from '@/common/components/layout/AppShell'
import SurfacePanel from '@/common/components/layout/SurfacePanel'
import { AUTH_SCOPE, getCurrentUser, logout } from '@/common/auth/auth'

function SessionCard({
  badge,
  title,
  description,
  actions,
  accentClass,
  children = null,
}) {
  return (
    <SurfacePanel className="h-full p-5 sm:p-6">
      <div className="space-y-5">
        <div className="space-y-3">
          <div
            className={`inline-flex rounded-full border px-3 py-1 text-xs font-medium uppercase tracking-[0.2em] ${accentClass}`}
          >
            {badge}
          </div>
          <div className="space-y-2">
            <div className="text-xl font-semibold text-slate-50">{title}</div>
            <div className="text-sm leading-6 text-slate-300">
              {description}
            </div>
          </div>
          {children}
        </div>
        <div className="flex flex-wrap gap-3">{actions}</div>
      </div>
    </SurfacePanel>
  )
}

export default function HomePage() {
  const navigate = useNavigate()
  const admin = getCurrentUser(AUTH_SCOPE.ADMIN)

  const handleLogout = (scope, nextPath) => {
    logout(scope)
    navigate(nextPath, { replace: true })
  }

  return (
    <AppShell className="px-4 py-8 sm:px-6 sm:py-10">
      <div className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-6xl items-center">
        <div className="grid w-full gap-6 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
          <SurfacePanel className="p-6 sm:p-8 lg:p-10">
            <div className="space-y-8">
              <div className="space-y-4">
                <div className="inline-flex rounded-full border border-cyan-300/30 bg-cyan-300/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.24em] text-cyan-100">
                  OpenAI OAuth API Service
                </div>
                <div className="max-w-2xl space-y-3">
                  <h1 className="text-3xl font-semibold tracking-tight text-slate-50 sm:text-4xl lg:text-5xl">
                    OAuth 登录与 API 用量服务
                  </h1>
                  <p className="text-sm leading-7 text-slate-300 sm:text-base">
                    管理员接入 OAuth/OIDC 登录，统一管理下游 API key、OpenAI
                    上游出口、用量记录和运行状态。上游只接入官方 OpenAI API key，
                    不复用客户端登录态。
                  </p>
                </div>
              </div>

              <div className="grid gap-4 sm:grid-cols-3">
                <div className="rounded-3xl border border-white/10 bg-white/[0.03] p-5">
                  <div className="text-sm font-medium text-slate-100">
                    下游接入
                  </div>
                  <div className="mt-2 text-sm leading-6 text-slate-300">
                    下游系统使用本系统签发的 API key 调用 `/v1/*`，不直接接触真实 OpenAI API
                    key。
                  </div>
                </div>
                <div className="rounded-3xl border border-white/10 bg-white/[0.03] p-5">
                  <div className="text-sm font-medium text-slate-100">
                    管理后台
                  </div>
                  <div className="mt-2 text-sm leading-6 text-slate-300">
                    管理员维护 key、配额、代理配置、usage 统计和审计记录。
                  </div>
                </div>
                <div className="rounded-3xl border border-white/10 bg-white/[0.03] p-5">
                  <div className="text-sm font-medium text-slate-100">
                    合规边界
                  </div>
                  <div className="mt-2 text-sm leading-6 text-slate-300">
                    禁止接入 Codex / ChatGPT 登录态、Cookie、设备码或个人账号
                    token。
                  </div>
                </div>
              </div>
            </div>
          </SurfacePanel>

          <div className="grid gap-6">
            <SessionCard
              badge="管理入口"
              title={admin ? `管理员：${admin.username}` : '管理控制台'}
              description={
                admin
                  ? '管理员已登录，可以继续进入后台控制台。'
                  : '管理员通过独立入口登录，用于访问后台控制台、接入管理和 usage 监控。'
              }
              accentClass="border-amber-300/30 bg-amber-300/10 text-amber-100"
              actions={
                admin
                  ? [
                    <Link
                      key="admin-console"
                      to="/admin-menu"
                      className="rounded-full bg-amber-300 px-4 py-2 text-sm font-semibold text-slate-950 transition hover:bg-amber-200"
                    >
                      进入管理控制台
                    </Link>,
                    <button
                      key="admin-logout"
                      type="button"
                      onClick={() =>
                          handleLogout(AUTH_SCOPE.ADMIN, '/admin-login')
                        }
                      className="border-white/14 hover:bg-white/8 rounded-full border px-4 py-2 text-sm font-medium text-slate-100 transition"
                    >
                      退出管理员登录
                    </button>,
                    ]
                  : [
                    <Link
                      key="admin-login"
                      to="/admin-login"
                      className="rounded-full bg-amber-300 px-4 py-2 text-sm font-semibold text-slate-950 transition hover:bg-amber-200"
                    >
                      管理员登录
                    </Link>,
                    ]
              }
            >
              {!admin ? (
                <div className="rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-slate-300">
                  管理员账号由后端初始化或部署环境变量控制，生产环境必须替换默认密码。
                </div>
              ) : null}
            </SessionCard>
          </div>
        </div>
      </div>
    </AppShell>
  )
}
