// web/src/common/utils/jsonRpc.js
import { RpcError } from '@/common/utils/rpcError'
import { getToken, logout, getLoginPath } from '@/common/auth/auth'
import { authBus } from '@/common/auth/authBus'
import { isAuthFailureCode } from '@/common/consts/errorCodes'

let globalRpcId = 0

export class JsonRpc {
  constructor({ url, basePath = '/rpc', authScope = 'user' }) {
    if (!url) {
      throw new Error('JsonRpc: url is required, e.g. "system" or "auth"')
    }
    this.url = url
    this.basePath = basePath
    this.authScope = authScope
  }

  async call(method, params = {}, options = {}) {
    const { receiveError = false, signal } = options
    const id = String(++globalRpcId)

    let response
    let json

    // 自动附带 token。
    const token = getToken(this.authScope)
    const headers = {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    }
    if (token) {
      headers.Authorization = `Bearer ${token}`
    }

    try {
      response = await fetch(`${this.basePath}/${this.url}`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          jsonrpc: '2.0',
          id,
          method,
          params,
        }),
        signal,
      })
    } catch (e) {
      throw new RpcError('Network error', {
        isNetworkError: true,
        cause: e,
      })
    }

    try {
      json = await response.json()
    } catch (e) {
      throw new RpcError('Invalid JSON response from server', {
        httpStatus: response.status,
        cause: e,
      })
    }

    // 1) HTTP 非 2xx
    if (!response.ok) {
      throw RpcError.fromHttp(response.status, json)
    }

    // 2) Kratos 框架级错误
    if (typeof json.code === 'number' && json.message) {
      const err = RpcError.fromKratos(json)
      if (receiveError) return err
      throw err
    }

    // 3) JSON-RPC error 字段
    if (json.error) {
      const err = RpcError.fromJsonRpc(json)
      if (receiveError) return err
      throw err
    }

    // 4) 业务错误 result.code != 0
    const { result } = json
    if (result && typeof result.code === 'number' && result.code !== 0) {
      handleAuthError(result.code, result.message, this.authScope)
      const err = RpcError.fromBiz(json)
      if (receiveError) return err
      throw err
    }

    return result
  }
}

function handleAuthError(code, message, authScope) {
  // 仅登录态失效才清 token，避免把权限不足误处理成登出。
  if (!isAuthFailureCode(code)) return

  // 1) 清 token
  logout(authScope)

  // 2) 通知 UI：弹窗 + 跳转交给 React
  authBus.emitUnauthorized?.({
    from: {
      pathname: window.location.pathname,
      search: window.location.search,
      hash: window.location.hash,
    },
    message: message || '请重新登录',
    loginPath: getLoginPath(authScope),
  })
}
