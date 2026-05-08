import React, { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import AppShell from '@/common/components/layout/AppShell'
import SurfacePanel from '@/common/components/layout/SurfacePanel'
import { AUTH_SCOPE, logout } from '@/common/auth/auth'
import { ADMIN_BASE_PATH } from '@/common/utils/adminRpc'
import { JsonRpc } from '@/common/utils/jsonRpc'

export default function AdminMenuPage() {
  const navigate = useNavigate()
  const [showLogoutConfirm, setShowLogoutConfirm] = useState(false)
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
      setShowLogoutConfirm(false)
      logout(AUTH_SCOPE.ADMIN)
      navigate('/admin-login', { replace: true })
    }
  }

  return (
    <AppShell className="flex items-center justify-center px-4 py-10">
      <div className="w-full max-w-[520px]">
        <div className="mb-6 text-center">
          <div className="inline-flex rounded-full border border-amber-300/30 bg-amber-300/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.24em] text-amber-100">
            管理控制台
          </div>
          <div className="mt-4 text-2xl font-semibold tracking-tight text-slate-50">
            管理控制台
          </div>
          <div className="mt-2 text-sm leading-6 text-slate-300">
            看板只放 API 转发和用量概览；key、模型和 usage 明细进入独立入口。
          </div>
        </div>

        <SurfacePanel className="p-4 sm:p-6">
          <div className="space-y-3 p-4 sm:p-6">
            <button
              type="button"
              onClick={() => navigate('/admin-api')}
              className="w-full rounded-2xl bg-cyan-300 px-4 py-3 font-semibold text-slate-950 transition hover:bg-cyan-200 active:bg-cyan-400"
            >
              业务看板
            </button>

            <button
              type="button"
              onClick={() => navigate('/admin-keys')}
              className="w-full rounded-2xl bg-cyan-300 px-4 py-3 font-semibold text-slate-950 transition hover:bg-cyan-200 active:bg-cyan-400"
            >
              下游 key
            </button>

            <button
              type="button"
              onClick={() => navigate('/admin-models')}
              className="w-full rounded-2xl bg-cyan-300 px-4 py-3 font-semibold text-slate-950 transition hover:bg-cyan-200 active:bg-cyan-400"
            >
              模型列表
            </button>

            <button
              type="button"
              onClick={() => navigate('/admin-usage')}
              className="w-full rounded-2xl bg-cyan-300 px-4 py-3 font-semibold text-slate-950 transition hover:bg-cyan-200 active:bg-cyan-400"
            >
              usage 记录
            </button>

            <button
              type="button"
              onClick={() => setShowLogoutConfirm(true)}
              className="w-full rounded-2xl bg-amber-300 px-4 py-3 font-semibold text-slate-950 transition hover:bg-amber-200 active:bg-amber-400"
            >
              退出登录
            </button>
            {showLogoutConfirm ? (
              <div className="rounded-2xl border border-white/10 bg-black/20 p-3">
                <div className="mb-3 text-center text-sm text-slate-200">
                  确认退出管理员登录吗？
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={handleLogout}
                    className="rounded-xl bg-amber-300 px-3 py-2 text-sm font-semibold text-slate-950 transition hover:bg-amber-200 active:bg-amber-400"
                  >
                    确认退出
                  </button>
                  <button
                    type="button"
                    onClick={() => setShowLogoutConfirm(false)}
                    className="border-white/14 hover:bg-white/8 rounded-xl border bg-transparent px-3 py-2 text-sm font-semibold text-slate-100 transition"
                  >
                    取消
                  </button>
                </div>
              </div>
            ) : null}
          </div>
        </SurfacePanel>
      </div>
    </AppShell>
  )
}
