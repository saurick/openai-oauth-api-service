// web/src/common/utils/rpcError.js
// 示例
// try {
//     const result = await authRpc.call('login', { username, password })
//     // result.login.userId / result.login.nickname ...
//   } catch (e) {
//     if (e instanceof RpcError) {
//       console.error('RPC 错误：', e.message, e.code, e.httpStatus)
//     } else {
//       console.error('未知错误：', e)
//     }
//   }
export class RpcError extends Error {
  constructor(message, extra = {}) {
    super(message || 'RPC Error')
    Object.setPrototypeOf(this, RpcError.prototype)
    this.name = 'RpcError'

    // 常用字段
    this.code = extra.code ?? null // 业务错误码 / 框架错误码
    this.httpStatus = extra.httpStatus ?? null
    this.isNetworkError = !!extra.isNetworkError
    this.json = extra.json ?? null // 原始 JSON 响应
    this.cause = extra.cause // 原始异常（可选）
  }

  // HTTP 非 2xx
  static fromHttp(status, json) {
    const msg = (json && json.message) || `HTTP error ${status}`
    return new RpcError(msg, {
      code: json?.code ?? status,
      httpStatus: status,
      json,
    })
  }

  // Kratos 错误格式 { code, reason, message, metadata }
  static fromKratos(json) {
    return new RpcError(json.message || 'Server error', {
      code: json.code,
      httpStatus: 500,
      json,
    })
  }

  // 标准 JSON-RPC error 对象（如果你以后真用上）
  static fromJsonRpc(json) {
    const errObj =
      typeof json.error === 'object'
        ? json.error
        : { message: String(json.error) }
    return new RpcError(errObj.message || 'JSON-RPC error', {
      code: errObj.code,
      json,
    })
  }

  // 我们约定的业务级错误：result.code != 0
  static fromBiz(json) {
    const result = json.result || {}
    return new RpcError(result.message || 'Business error', {
      code: result.code,
      json,
    })
  }
}
