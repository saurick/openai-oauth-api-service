// web/src/App.jsx
import React, { Suspense, useEffect } from 'react'
import { Navigate, Route, Routes, useNavigate } from 'react-router-dom'
import { Helmet } from 'react-helmet-async'
import { Loading } from '@/common/components/loading'
import AuthGuard from '@/common/auth/AuthGuard'
import { authBus } from '@/common/auth/authBus'
import { appAlert } from '@/common/components/modal/alertBridge'
import AdminLoginPage from '@/pages/AdminLogin'
import AdminApiPage from '@/pages/AdminApi'
import AdminDashboardPage from '@/pages/AdminDashboard'
import OAuthCallbackPage from '@/pages/OAuthCallback'

import 'normalize.css/normalize.css'

// const Index = lazy(() => import('@/pages'))

const App = () => {
  const navigate = useNavigate()
  const appTitle = import.meta.env.VITE_APP_TITLE || 'Saurick API Console'

  useEffect(() => {
    return authBus.onUnauthorized(({ from, message, loginPath }) => {
      // 如果 payload 没带，就 fallback 为当前 location
      const safeFrom = from || {
        pathname: window.location.pathname,
        search: window.location.search,
        hash: window.location.hash,
      }
      const targetLoginPath = loginPath || '/admin-login'

      appAlert({
        title: '登录状态已失效',
        message: message || '登录已过期，请重新登录',
        confirmText: '重新登录',
        onConfirm: () => {
          navigate(targetLoginPath, {
            replace: true,
            state: { from: safeFrom },
          })
        },
      })
    })
  }, [navigate])

  return (
    <>
      <Helmet>
        <title>{appTitle}</title>
      </Helmet>
      <Suspense fallback={<Loading />}>
        <Routes>
          {/* <Route path="*" element={<Index />} />  // 匹配所有路径，显示Index组件 */}
          {/* <Route path="/about" element={<About />} />  // 匹配/about路径，显示About组件 */}
          <Route path="/login" element={<Navigate to="/admin-login" replace />} />
          <Route path="/oauth-login" element={<Navigate to="/admin-login" replace />} />
          <Route path="/oauth/callback" element={<OAuthCallbackPage />} />
          <Route path="/admin-login" element={<AdminLoginPage />} />
          <Route path="/register" element={<Navigate to="/admin-login" replace />} />
          <Route
            path="/admin-menu"
            element={
              <AuthGuard requireAdmin>
                <Navigate to="/admin-dashboard" replace />
              </AuthGuard>
            }
          />
          <Route
            path="/admin-accounts"
            element={
              <AuthGuard requireAdmin>
                <Navigate to="/admin-dashboard" replace />
              </AuthGuard>
            }
          />
          <Route
            path="/admin-api"
            element={
              <AuthGuard requireAdmin>
                <Navigate to="/admin-dashboard" replace />
              </AuthGuard>
            }
          />
          <Route
            path="/admin-keys"
            element={
              <AuthGuard requireAdmin>
                <AdminApiPage view="keys" />
              </AuthGuard>
            }
          />
          <Route
            path="/admin-models"
            element={
              <AuthGuard requireAdmin>
                <AdminApiPage view="models" />
              </AuthGuard>
            }
          />
          <Route
            path="/admin-analytics"
            element={
              <AuthGuard requireAdmin>
                <Navigate to="/admin-usage" replace />
              </AuthGuard>
            }
          />
          <Route
            path="/admin-usage"
            element={
              <AuthGuard requireAdmin>
                <AdminApiPage view="usage" />
              </AuthGuard>
            }
          />
          <Route
            path="/admin-dashboard"
            element={
              <AuthGuard requireAdmin>
                <AdminDashboardPage />
              </AuthGuard>
            }
          />
          <Route
            path="/admin-guide"
            element={<Navigate to="/admin-dashboard" replace />}
          />
          <Route
            path="/admin-oauth"
            element={<Navigate to="/admin-dashboard" replace />}
          />
          <Route path="/portal" element={<Navigate to="/admin-login" replace />} />
          <Route
            path="/admin-users"
            element={<Navigate to="/admin-dashboard" replace />}
          />
          <Route
            path="/admin-hierarchy"
            element={<Navigate to="/admin-dashboard" replace />}
          />
          <Route path="/" element={<Navigate to="/admin-login" replace />} />
        </Routes>
      </Suspense>
    </>
  )
}

export default App
