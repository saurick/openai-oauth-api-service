import React from 'react'
import { Route, Navigate } from 'react-router'

// Protect 保护路由
export const Protect = ({
  authorized = false,
  redirectTo = '/',
  children,
  ...other
}) => {
  return (
    <Route
      {...other}
      render={() => {
        return authorized ? children : <Navigate to={redirectTo} />
      }}
    />
  )
}
