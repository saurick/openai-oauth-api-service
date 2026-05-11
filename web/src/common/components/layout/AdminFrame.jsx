import React, { useMemo } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import AppShell from '@/common/components/layout/AppShell'
import AdminThemeToggle from '@/common/components/layout/AdminThemeToggle'
import { AUTH_SCOPE, getCurrentUser, logout } from '@/common/auth/auth'
import { ADMIN_BASE_PATH } from '@/common/utils/adminRpc'
import { JsonRpc } from '@/common/utils/jsonRpc'

const NAV_GROUPS = [
  {
    label: '运营总览',
    items: [{ to: '/admin-dashboard', label: '业务看板', icon: GaugeIcon }],
  },
  {
    label: '转发配置',
    items: [
      { to: '/admin-keys', label: 'API 凭据', icon: KeyIcon },
      { to: '/admin-models', label: '模型管理', icon: RouteIcon },
      { to: '/admin-upstream', label: '上游模式', icon: UpstreamIcon },
    ],
  },
  {
    label: '用量统计',
    items: [
      { to: '/admin-usage', label: '用量日志', icon: ChartIcon },
    ],
  },
]

function BrandMark() {
  return (
    <div className="leading-none">
      <div className="text-[30px] font-extrabold tracking-tight text-[#173b59]">
        <span>OA</span>
        <span className="text-[#d6a23a]">S</span>
      </div>
      <div className="mt-[-2px] text-[9px] font-bold uppercase tracking-[0.06em] text-[#173b59]">
        Saurick API Console
      </div>
    </div>
  )
}

function NavIcon({ icon: Icon }) {
  return (
    <span className="flex h-5 w-5 shrink-0 items-center justify-center">
      <Icon />
    </span>
  )
}

function Sidebar() {
  return (
    <aside className="border-b border-[#dce8df] bg-[#f5fbf7] lg:min-h-screen lg:border-b-0 lg:border-r">
      <div className="flex h-[86px] items-center border-b border-[#dce8df] px-4">
        <BrandMark />
      </div>

      <nav className="max-h-[calc(100vh-86px)] overflow-auto px-4 py-5">
        {NAV_GROUPS.map((group) => (
          <div key={group.label} className="mb-6">
            <div className="mb-3 px-3 text-sm text-[#7b8780]">
              {group.label}
            </div>
            <div className="space-y-1">
              {group.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) =>
                    [
                      'flex min-h-10 items-center gap-3 rounded-md px-3 text-sm font-semibold transition',
                      isActive
                        ? 'bg-[#c7d3c5] text-[#238a43]'
                        : 'text-[#1f2d25] hover:bg-[#e7efe9] hover:text-[#238a43]',
                    ].join(' ')
                  }
                >
                  <NavIcon icon={item.icon} />
                  <span>{item.label}</span>
                </NavLink>
              ))}
            </div>
          </div>
        ))}
      </nav>
    </aside>
  )
}

export default function AdminFrame({
  title,
  description,
  breadcrumb,
  actions = null,
  children,
}) {
  const navigate = useNavigate()
  const user = getCurrentUser(AUTH_SCOPE.ADMIN)
  const authRpc = useMemo(
    () =>
      new JsonRpc({
        url: 'auth',
        basePath: ADMIN_BASE_PATH,
        authScope: AUTH_SCOPE.ADMIN,
      }),
    []
  )

  const handleLogout = async () => {
    try {
      await authRpc.call('logout')
    } catch (e) {
      console.warn('服务器 logout 失败', e)
    } finally {
      logout(AUTH_SCOPE.ADMIN)
      navigate('/admin-login', { replace: true })
    }
  }

  return (
    <AppShell variant="admin" className="admin-frame">
      <div className="lg:grid lg:min-h-screen lg:grid-cols-[276px_minmax(0,1fr)]">
        <Sidebar />

        <div className="min-w-0">
          <header className="sticky top-0 z-20 border-b border-[#dce8df] bg-white/95 backdrop-blur">
            <div className="flex min-h-[86px] flex-col gap-3 px-4 py-3 sm:px-5 lg:flex-row lg:items-center lg:justify-between lg:px-6">
              <div className="text-sm font-semibold text-[#1f2d25]">
                API 管理后台
              </div>
              <div className="flex flex-wrap items-center gap-2">
                {actions}
                <AdminThemeToggle />
                <span className="rounded-md border border-[#f0c868] bg-[#fff8df] px-3 py-1.5 text-xs font-semibold text-[#c07a00]">
                  超级管理员
                </span>
                <span className="text-sm text-[#7b8780]">
                  {user?.username || 'admin'}
                </span>
                <button
                  type="button"
                  onClick={handleLogout}
                  className="rounded-md border border-[#d6ded8] bg-white px-3 py-1.5 text-sm text-[#1f2d25] transition hover:border-[#238a43] hover:text-[#238a43]"
                >
                  退出
                </button>
              </div>
            </div>
          </header>

          <main className="px-4 py-5 sm:px-5 lg:px-6">
            {breadcrumb ? (
              <div className="mb-4 text-sm text-[#7b8780]">{breadcrumb}</div>
            ) : null}

            {title || description ? (
              <section className="mb-5 rounded-lg border border-[#dde8df] bg-white px-5 py-6 shadow-[0_2px_10px_rgba(24,61,42,0.08)]">
                {title ? (
                  <h1 className="text-xl font-bold text-[#1f2d25]">{title}</h1>
                ) : null}
                {description ? (
                  <p className="mt-3 max-w-5xl text-sm leading-6 text-[#7b8780]">
                    {description}
                  </p>
                ) : null}
              </section>
            ) : null}

            <div className="space-y-5">{children}</div>
          </main>
        </div>
      </div>
    </AppShell>
  )
}

function GaugeIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path
        d="M5 14a7 7 0 0 1 14 0M12 14l3-4M7 17h10"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="2"
      />
    </svg>
  )
}

function RouteIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path
        d="M6 6h.01M18 18h.01M7 6h5a3 3 0 0 1 0 6H9a3 3 0 0 0 0 6h8"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="2"
      />
    </svg>
  )
}

function KeyIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path
        d="M14 10a4 4 0 1 1-2.4-3.67A4 4 0 0 1 14 10Zm0 0h6m-3 0v3"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2"
      />
    </svg>
  )
}

function ChartIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path
        d="M5 19V5m0 14h14M9 16v-5m4 5V8m4 8v-7"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="2"
      />
    </svg>
  )
}

function UpstreamIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path
        d="M4 7h11m0 0-3-3m3 3-3 3M20 17H9m0 0 3-3m-3 3 3 3"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2"
      />
    </svg>
  )
}
