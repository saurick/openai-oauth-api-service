export const CLIENT_CONFIG_MODEL_OPTIONS = [
  { label: 'GPT-5.6 Sol', value: 'gpt-5.6-sol' },
  { label: 'GPT-5.6 Terra', value: 'gpt-5.6-terra' },
  { label: 'GPT-5.6 Luna', value: 'gpt-5.6-luna' },
  { label: 'GPT-5.5', value: 'gpt-5.5' },
]

export const CLIENT_CONFIG_DEFAULTS = {
  baseUrl: 'https://oauth-api.saurick.me/v1',
  apiKey: '<API_KEY>',
  model: CLIENT_CONFIG_MODEL_OPTIONS[0].value,
  profile: 'saurick',
}

export const CLIENT_CONFIG_OS_OPTIONS = [
  { label: 'macOS / Linux', value: 'mac' },
  { label: 'Windows', value: 'win' },
]

export const CLIENT_CONFIG_TOOL_OPTIONS = [
  { label: 'Codex', value: 'codex' },
  { label: 'opencode', value: 'opencode' },
]

const CODEX_TEMPLATE = `#:schema https://developers.openai.com/codex/config-schema.json

model = "{{MODEL}}"
model_provider = "oauth-api-service"
approval_policy = "never"
sandbox_mode = "danger-full-access"
check_for_update_on_startup = true
personality = "pragmatic"
model_reasoning_effort = "medium"
model_reasoning_summary = "detailed"
model_supports_reasoning_summaries = true
hide_agent_reasoning = false

[model_providers."oauth-api-service"]
name = "OpenAI OAuth API Service"
base_url = "{{BASE_URL}}"
wire_api = "responses"
experimental_bearer_token = "{{API_KEY}}"

[features]
multi_agent = true

[sandbox_workspace_write]
network_access = true

[mcp_servers.openaiDeveloperDocs]
url = "https://developers.openai.com/mcp"
enabled = true

[history]
persistence = "save-all"
max_bytes = 10485760
`

const CODEX_WINDOWS_EXTRA = `
[windows]
sandbox = "elevated"
`

const INSTALL_PATHS = {
  opencode: {
    mac: '~/.config/opencode/opencode.json',
    win: '%USERPROFILE%\\.config\\opencode\\opencode.json',
  },
}

export function normalizeBaseUrl(value) {
  return String(value || '')
    .trim()
    .replace(/\/+$/u, '')
}

export function normalizeApiKey(value) {
  return String(value || '').trim()
}

export function normalizeProfile(value) {
  const normalized = String(value || '')
    .trim()
    .replace(/[^A-Za-z0-9_-]/gu, '-')
    .replace(/-+/gu, '-')
    .replace(/^-|-$/gu, '')
  return normalized || CLIENT_CONFIG_DEFAULTS.profile
}

export function normalizeModel(value) {
  return CLIENT_CONFIG_MODEL_OPTIONS.some((option) => option.value === value)
    ? value
    : CLIENT_CONFIG_DEFAULTS.model
}

function buildOpenCodeTemplate({ baseUrl, apiKey, model }) {
  const variants = {
    low: { reasoningEffort: 'low' },
    medium: { reasoningEffort: 'medium' },
    high: { reasoningEffort: 'high' },
    xhigh: { reasoningEffort: 'xhigh' },
  }
  const models = Object.fromEntries(
    CLIENT_CONFIG_MODEL_OPTIONS.map((option) => [
      option.value,
      {
        name: option.value,
        reasoning: true,
        reasoningEffort: 'medium',
        variants,
        modalities: {
          input: ['text', 'image', 'pdf'],
          output: ['text'],
        },
      },
    ])
  )

  return `${JSON.stringify(
    {
      $schema: 'https://opencode.ai/config.json',
      default_agent: 'build',
      agent: {
        build: {
          model: `oauth-api-service/${model}`,
          variant: 'medium',
          options: { store: false },
          permission: {},
        },
        plan: {
          model: `oauth-api-service/${model}`,
          variant: 'medium',
          options: { store: false },
          permission: {},
        },
      },
      permission: { '*': 'allow' },
      provider: {
        'oauth-api-service': {
          npm: '@ai-sdk/openai-compatible',
          name: 'OpenAI OAuth API Service',
          options: {
            baseURL: baseUrl,
            apiKey,
            timeout: 600000,
          },
          models,
        },
      },
    },
    null,
    2
  )}\n`
}

export function getClientConfigTemplate(tool, os) {
  if (tool === 'codex') {
    return os === 'win'
      ? `${CODEX_TEMPLATE}${CODEX_WINDOWS_EXTRA}`
      : CODEX_TEMPLATE
  }
  return buildOpenCodeTemplate({
    baseUrl: '{{BASE_URL}}',
    apiKey: '{{API_KEY}}',
    model: '{{MODEL}}',
  })
}

export function renderClientConfigTemplate({
  tool,
  os,
  baseUrl,
  apiKey,
  model,
}) {
  const values = {
    baseUrl: normalizeBaseUrl(baseUrl) || CLIENT_CONFIG_DEFAULTS.baseUrl,
    apiKey: normalizeApiKey(apiKey) || CLIENT_CONFIG_DEFAULTS.apiKey,
    model: normalizeModel(model),
  }
  if (tool !== 'codex') return buildOpenCodeTemplate(values)

  const replacements = {
    '{{BASE_URL}}': values.baseUrl,
    '{{API_KEY}}': values.apiKey,
    '{{MODEL}}': values.model,
  }
  return Object.entries(replacements).reduce(
    (content, [placeholder, value]) => content.split(placeholder).join(value),
    getClientConfigTemplate(tool, os)
  )
}

export function getClientConfigFilename(tool, os, profile) {
  return tool === 'codex'
    ? `${normalizeProfile(profile)}.config.toml`
    : 'opencode.json'
}

export function getClientConfigInstallPath(tool, os, profile) {
  if (tool !== 'codex') return INSTALL_PATHS[tool]?.[os] || ''
  const filename = getClientConfigFilename(tool, os, profile)
  return os === 'win'
    ? `%USERPROFILE%\\.codex\\${filename}`
    : `~/.codex/${filename}`
}
