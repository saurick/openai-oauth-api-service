const GATEWAY_ERROR_TYPES = {
  gateway_api_key_invalid: {
    label: 'API key 无效',
    description: '请求携带的下游 API key 不存在或格式无效。',
  },
  gateway_api_key_disabled: {
    label: 'API key 已禁用',
    description: '请求携带的下游 API key 已在后台禁用。',
  },
  gateway_model_disabled: {
    label: '模型已禁用',
    description: '请求模型在后台模型管理中已禁用。',
  },
  gateway_model_not_allowed: {
    label: '模型未授权',
    description: '请求模型不在该 API 凭据允许模型范围内。',
  },
  gateway_rate_limited: {
    label: '网关限流',
    description: '请求命中服务端网关限流。',
  },
  gateway_quota_exceeded: {
    label: '凭据额度超限',
    description: '请求开始前该 API 凭据的 Token 额度已超限。',
  },
  gateway_reasoning_effort_invalid: {
    label: 'Effort 非法',
    description:
      '请求传入的 reasoning_effort 不在 low / medium / high / xhigh 范围内。',
  },
  gateway_error: {
    label: '网关错误',
    description: '网关侧未分类错误，需要结合 request_id 和服务日志查看。',
  },
  codex_backend_auth_failed: {
    label: 'Backend 鉴权失败',
    description:
      '服务器 Codex 登录态无效、auth.json / refresh token 失效，或上游返回 401 / 403。',
  },
  codex_backend_rate_limited: {
    label: 'Backend 限流',
    description: '上游返回 429，可能是账号、模型或组织维度被限流。',
  },
  codex_backend_http_5xx: {
    label: 'Backend 5xx',
    description: 'Codex backend 或其上游服务返回 5xx。',
  },
  codex_backend_timeout: {
    label: 'Backend 超时',
    description:
      'Codex backend 调用超过超时时间；常见于上游慢、网络慢或 CODEX_BACKEND_TIMEOUT_SECONDS 到期。',
  },
  codex_backend_response_failed: {
    label: 'Backend response failed',
    description: '上游 SSE 返回 response.failed，表示本次 response 执行失败。',
  },
  context_length_exceeded: {
    label: '上下文超限',
    description:
      '请求历史超过模型上下文窗口；网关会先尝试压缩可压缩历史，仍超限时直接拦截，避免客户端反复重试。',
  },
  codex_backend_response_incomplete: {
    label: 'Backend response incomplete',
    description:
      '上游 SSE 返回 response.incomplete，可能因长度、上下文、策略、工具或内部中断。',
  },
  codex_backend_stream_error: {
    label: 'Backend 流中断',
    description:
      'SSE 流在首个有效上游事件前连接 reset、unexpected EOF、代理或网络断流。',
  },
  codex_backend_stream_interrupted: {
    label: 'Backend 流中途断开',
    description:
      '上游 SSE 已返回部分事件，但尚未返回 response.completed / [DONE] 就断开；通常是上游、代理或网络中途断流，不能在网关侧安全自动重试。',
  },
  codex_backend_http_error: {
    label: 'Backend HTTP 错误',
    description: 'backend 返回其他非 2xx HTTP 状态，且不属于鉴权、限流或 5xx。',
  },
  codex_backend_upstream_failed: {
    label: 'Backend 未分类失败',
    description: 'backend 兜底错误，需要结合服务日志里的 err 查看。',
  },
  client_canceled: {
    label: '客户端取消',
    description:
      '下游客户端或入口代理主动断开请求，通常需要排查客户端超时、网络中断或流式保活识别。',
  },
  codex_cli_timeout: {
    label: 'CLI 超时',
    description: 'Codex CLI 执行超过 CODEX_CLI_TIMEOUT_SECONDS。',
  },
  codex_cli_not_found: {
    label: 'CLI 不存在',
    description: '容器内找不到 codex 二进制，或 CODEX_CLI_BIN / PATH 配错。',
  },
  codex_cli_empty_prompt: {
    label: 'CLI 空输入',
    description: '请求体没有有效 user input，或请求转换后 prompt 为空。',
  },
  codex_cli_empty_answer: {
    label: 'CLI 空回复',
    description:
      'CLI 正常退出但未解析到最终回答，可能输出格式变化或模型无最终回答。',
  },
  codex_cli_upstream_failed: {
    label: 'CLI 未分类失败',
    description: 'CLI 兜底错误，需要结合服务日志里的命令错误和输出摘要查看。',
  },
}

export const GATEWAY_ERROR_TYPE_HELP =
  '错误字段来自 usage.error_type；失败或中断时记录细分类型，例如 backend 超时、鉴权失败、上游 5xx、限流、客户端取消或 CLI 超时等；客户端取消会单独统计，不计入服务错误率。诊断字段只保存请求大小、fallback 状态和脱敏上游摘要，不保存 prompt / output 正文。'

export function gatewayErrorTypeInfo(code) {
  const normalized = String(code || '').trim()
  if (!normalized) return null
  return GATEWAY_ERROR_TYPES[normalized] || null
}

export function gatewayErrorTypeLabel(code) {
  return gatewayErrorTypeInfo(code)?.label || ''
}

export function gatewayErrorTypeDescription(code) {
  return gatewayErrorTypeInfo(code)?.description || ''
}

export function gatewayErrorTypeTitle(code) {
  const normalized = String(code || '').trim()
  if (!normalized) return ''
  const info = gatewayErrorTypeInfo(normalized)
  if (!info) return normalized
  return `${normalized}：${info.label}。${info.description}`
}
