// web/src/common/utils/errorMessage.js
import { logout } from '@/common/auth/auth'
import {
  DEFAULT_RPC_ERROR_MESSAGES,
  RpcErrorCode,
  isAuthFailureCode,
} from '@/common/consts/errorCodes'

const RAW_ERROR_MESSAGE_MAP = Object.freeze({
  'business error': '请求失败，请稍后重试',
  forbidden: DEFAULT_RPC_ERROR_MESSAGES[RpcErrorCode.PERMISSION_DENIED],
  'invalid json response from server': '服务器返回异常，请稍后重试',
  'json-rpc error': '请求失败，请稍后重试',
  'network error': '网络错误，请稍后重试',
  'request failed': '请求失败，请稍后重试',
  'request failed, network error': '网络错误，请稍后重试',
  'rpc error': '请求失败，请稍后重试',
  'server error': '服务器异常，请稍后重试',
  'session expired': DEFAULT_RPC_ERROR_MESSAGES[RpcErrorCode.AUTH_REQUIRED],
  unauthorized: DEFAULT_RPC_ERROR_MESSAGES[RpcErrorCode.AUTH_REQUIRED],
  'unknown error': '未知错误，请稍后重试',
})

const RAW_ERROR_PATTERNS = Object.freeze([
  [
    /^http error 401$/iu,
    DEFAULT_RPC_ERROR_MESSAGES[RpcErrorCode.AUTH_REQUIRED],
  ],
  [
    /^http error 403$/iu,
    DEFAULT_RPC_ERROR_MESSAGES[RpcErrorCode.PERMISSION_DENIED],
  ],
  [/^http error \d+$/iu, '服务请求失败，请稍后重试'],
])

function containsCjk(text) {
  return /[\u3400-\u9fff]/u.test(String(text || ''))
}

function normalizeErrorText(message) {
  const text = String(message || '').trim()
  if (!text) return ''
  return text.replace(/^(rpcerror|error):\s*/iu, '')
}

function translateKnownErrorMessage(message) {
  const normalized = normalizeErrorText(message)
  if (!normalized) return ''

  const mapped = RAW_ERROR_MESSAGE_MAP[normalized.toLowerCase()]
  if (mapped) return mapped

  for (const [pattern, translated] of RAW_ERROR_PATTERNS) {
    if (pattern.test(normalized)) return translated
  }

  if (containsCjk(normalized)) return normalized
  return ''
}

function resolveActionErrorFallback(
  action,
  { fallback, suffix = '请稍后重试', defaultAction = '请求' } = {}
) {
  if (fallback) return fallback

  const normalizedAction = String(action || '').trim()
  const normalizedSuffix = String(suffix || '').trim() || '请稍后重试'
  if (!normalizedAction) {
    return `${defaultAction}失败，${normalizedSuffix}`
  }

  // 允许调用点只传“登录/保存/加载”这类动作词，统一在这里补完整兜底。
  if (
    /[，,]/u.test(normalizedAction) ||
    /(失败|异常)$/u.test(normalizedAction)
  ) {
    return normalizedAction
  }

  return `${normalizedAction}失败，${normalizedSuffix}`
}

// 用户可见错误统一走这里，避免把 transport 层英文兜底原文直接显示到页面。
export function getUserFacingErrorMessage(
  err,
  fallback = '请求失败，请稍后重试'
) {
  const code = Number(err?.code)
  if (DEFAULT_RPC_ERROR_MESSAGES[code]) {
    return DEFAULT_RPC_ERROR_MESSAGES[code]
  }
  if (err?.isNetworkError) {
    return '网络错误，请稍后重试'
  }

  const translated = translateKnownErrorMessage(
    typeof err === 'string' ? err : (err?.message ?? err)
  )
  if (translated) return translated

  return fallback
}

export function getActionErrorMessage(err, action, options = {}) {
  return getUserFacingErrorMessage(
    err,
    resolveActionErrorFallback(action, options)
  )
}

export function handleRpcError(err, { onNeedLogin } = {}) {
  const code = err?.code
  const msg = getUserFacingErrorMessage(err)

  // 仅登录态失效才触发登出，权限不足要保留当前会话。
  if (isAuthFailureCode(code)) {
    logout()
    onNeedLogin?.()
  }

  return msg
}

export const ERROR_MESSAGES = DEFAULT_RPC_ERROR_MESSAGES
