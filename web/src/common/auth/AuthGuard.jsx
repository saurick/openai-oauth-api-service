// web/src/common/auth/AuthGuard.jsx
import React from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { AUTH_SCOPE, getCurrentUser, getLoginPath } from '@/common/auth/auth'

/**
 * AuthGuard 保护路由
 * @param {boolean} requireAdmin - 是否需要管理员权限
 * @param {React.ReactNode} children - 子组件
 * @returns {React.ReactNode} 保护路由组件
 */
export default function AuthGuard({ requireAdmin = false, children }) {
  const location = useLocation()
  const authScope = requireAdmin ? AUTH_SCOPE.ADMIN : AUTH_SCOPE.USER
  const user = getCurrentUser(authScope)

  // 未登录 → 去登录页
  if (!user) {
    return (
      <Navigate
        to={getLoginPath(authScope)}
        replace
        state={{ from: location }}
      />
    )
  }

  // 需要管理员但不是 admin
  if (requireAdmin && user.role !== 'admin') {
    return (
      <Navigate
        to={getLoginPath(AUTH_SCOPE.ADMIN)}
        replace
        state={{ from: location }}
      />
    )
  }

  return children
}
