// 统一维护前端消费侧错误码分组与默认文案；原始码表由生成文件提供。
import { RpcErrorCode } from './errorCodes.generated.js'

export { RpcErrorCode }

export const AUTH_FAILURE_ERROR_CODES = Object.freeze([
  RpcErrorCode.AUTH_EXPIRED,
  RpcErrorCode.AUTH_INVALID,
  RpcErrorCode.AUTH_REQUIRED,
])

// 仅登录态失效错误会触发登出，权限不足必须保留当前会话。
export function isAuthFailureCode(code) {
  return AUTH_FAILURE_ERROR_CODES.includes(Number(code))
}

export const DEFAULT_RPC_ERROR_MESSAGES = Object.freeze({
  [RpcErrorCode.ADMIN_REQUIRED]: '只有管理员才能操作',
  [RpcErrorCode.AUTH_REQUIRED]: '请先登录',
  [RpcErrorCode.ADMIN_DISABLED]: '管理员已禁用',
  [RpcErrorCode.PERMISSION_DENIED]: '权限不足',
  [RpcErrorCode.AUTH_USER_NOT_FOUND]: '用户不存在',
  [RpcErrorCode.AUTH_INVALID_PASSWORD]: '密码错误',
  [RpcErrorCode.AUTH_USER_DISABLED]: '用户已被禁用',
  [RpcErrorCode.AUTH_USER_EXISTS]: '用户名已存在',
  [RpcErrorCode.AUTH_EXPIRED]: '登录已过期，请重新登录',
  [RpcErrorCode.AUTH_INVALID]: '登录无效，请重新登录',
})
