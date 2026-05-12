export const CLIENT_CONFIG_DEFAULTS = {
  baseUrl: 'https://oauth-api.saurick.me/v1',
  apiKey: '<API_KEY>',
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

const CODEX_COMMON_BEFORE_PROVIDER = `#:schema https://developers.openai.com/codex/config-schema.json

profile = "{{PROFILE}}"
approval_policy = "never"
sandbox_mode = "danger-full-access"
check_for_update_on_startup = true
personality = "pragmatic"

model = "gpt-5.5"
model_provider = "openai"
model_reasoning_effort = "medium"

[profiles."openai"]
model_provider = "openai"
model = "gpt-5.5"
model_reasoning_effort = "medium"
model_verbosity = "medium"
personality = "pragmatic"

[model_providers."oauth-api-service"]
name = "OpenAI OAuth API Service"
base_url = "{{BASE_URL}}"
wire_api = "responses"
experimental_bearer_token = "{{API_KEY}}"

[profiles."{{PROFILE}}"]
model_provider = "oauth-api-service"
model = "gpt-5.5"
model_reasoning_effort = "medium"
model_verbosity = "medium"
personality = "pragmatic"

[profiles.fast]
model_provider = "oauth-api-service"
model = "gpt-5.5"
model_reasoning_effort = "low"
model_verbosity = "low"
personality = "pragmatic"

[profiles.medium]
model_provider = "oauth-api-service"
model = "gpt-5.5"
model_reasoning_effort = "medium"
model_verbosity = "medium"
personality = "pragmatic"

[profiles.high]
model_provider = "oauth-api-service"
model = "gpt-5.5"
model_reasoning_effort = "high"
model_verbosity = "medium"
personality = "pragmatic"

[profiles.deep]
model_provider = "oauth-api-service"
model = "gpt-5.5"
model_reasoning_effort = "xhigh"
model_verbosity = "medium"
personality = "pragmatic"

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

const OPENCODE_TEMPLATE = `{
  "$schema": "https://opencode.ai/config.json",
  "default_agent": "build",
  "agent": {
    "build": {
      "model": "oauth-api-service/gpt-5.5",
      "variant": "medium",
      "options": {
        "store": false
      },
      "permission": {}
    },
    "plan": {
      "model": "oauth-api-service/gpt-5.5",
      "variant": "medium",
      "options": {
        "store": false
      },
      "permission": {}
    }
  },
  "permission": {
    "*": "allow"
  },
  "provider": {
    "oauth-api-service": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "OpenAI OAuth API Service",
      "options": {
        "baseURL": "{{BASE_URL}}",
        "apiKey": "{{API_KEY}}",
        "timeout": 600000
      },
      "models": {
        "gpt-5.5": {
          "name": "gpt-5.5",
          "reasoning": true,
          "reasoningEffort": "medium",
          "variants": {
            "low": {
              "reasoningEffort": "low"
            },
            "medium": {
              "reasoningEffort": "medium"
            },
            "high": {
              "reasoningEffort": "high"
            },
            "xhigh": {
              "reasoningEffort": "xhigh"
            }
          },
          "modalities": {
            "input": ["text", "image", "pdf"],
            "output": ["text"]
          }
        }
      }
    }
  }
}
`

const INSTALL_PATHS = {
  codex: {
    mac: '~/.codex/config.toml',
    win: '%USERPROFILE%\\.codex\\config.toml',
  },
  opencode: {
    mac: '~/.config/opencode/opencode.json',
    win: '%USERPROFILE%\\.config\\opencode\\opencode.json',
  },
}

export function normalizeBaseUrl(value) {
  return String(value || '').trim().replace(/\/+$/u, '')
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

export function getClientConfigTemplate(tool, os) {
  if (tool === 'codex') {
    return os === 'win'
      ? `${CODEX_COMMON_BEFORE_PROVIDER}${CODEX_WINDOWS_EXTRA}`
      : CODEX_COMMON_BEFORE_PROVIDER
  }
  return OPENCODE_TEMPLATE
}

export function renderClientConfigTemplate({ tool, os, baseUrl, apiKey, profile }) {
  const replacements = {
    '{{BASE_URL}}': normalizeBaseUrl(baseUrl) || CLIENT_CONFIG_DEFAULTS.baseUrl,
    '{{API_KEY}}': normalizeApiKey(apiKey) || CLIENT_CONFIG_DEFAULTS.apiKey,
    '{{PROFILE}}': normalizeProfile(profile),
  }
  return Object.entries(replacements).reduce(
    (content, [placeholder, value]) => content.split(placeholder).join(value),
    getClientConfigTemplate(tool, os)
  )
}

export function getClientConfigFilename(tool, os) {
  return tool === 'codex' ? 'config.toml' : 'opencode.json'
}

export function getClientConfigInstallPath(tool, os) {
  return INSTALL_PATHS[tool]?.[os] || ''
}
