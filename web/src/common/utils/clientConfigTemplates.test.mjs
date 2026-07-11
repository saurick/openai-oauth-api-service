import assert from 'node:assert/strict'
import test from 'node:test'
import fs from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'

function loadClientConfigTemplateModule() {
  const filePath = path.resolve(
    import.meta.dirname,
    './clientConfigTemplates.js'
  )
  const source = fs.readFileSync(filePath, 'utf8')
  const transformed = source
    .replace(/export const /g, 'const ')
    .replace(/export function /g, 'function ')
    .concat(
      '\nmodule.exports = { CLIENT_CONFIG_DEFAULTS, CLIENT_CONFIG_MODEL_OPTIONS, normalizeBaseUrl, normalizeApiKey, normalizeProfile, normalizeModel, getClientConfigTemplate, renderClientConfigTemplate, getClientConfigFilename, getClientConfigInstallPath };\n'
    )

  const sandbox = {
    module: { exports: {} },
    exports: {},
  }
  vm.runInNewContext(transformed, sandbox, { filename: filePath })
  return sandbox.module.exports
}

const {
  CLIENT_CONFIG_MODEL_OPTIONS,
  getClientConfigFilename,
  getClientConfigInstallPath,
  normalizeModel,
  normalizeProfile,
  renderClientConfigTemplate,
} = loadClientConfigTemplateModule()

const expectedModels = [
  'gpt-5.6-sol',
  'gpt-5.6-terra',
  'gpt-5.6-luna',
  'gpt-5.5',
]

test('clientConfigTemplates: Codex Windows 生成 profile v2 独立配置并选择目标模型', () => {
  const content = renderClientConfigTemplate({
    tool: 'codex',
    os: 'win',
    baseUrl: 'https://example.com/v1/',
    apiKey: 'ogw_test_key',
    model: 'gpt-5.6-terra',
  })

  assert.match(content, /base_url = "https:\/\/example\.com\/v1"/u)
  assert.match(content, /experimental_bearer_token = "ogw_test_key"/u)
  assert.match(content, /model = "gpt-5\.6-terra"/u)
  assert.doesNotMatch(content, /(^|\n)profile =/u)
  assert.doesNotMatch(content, /\[profiles\./u)
  assert.match(content, /model_reasoning_effort = "medium"/u)
  assert.match(content, /model_reasoning_summary = "detailed"/u)
  assert.match(content, /model_supports_reasoning_summaries = true/u)
  assert.match(content, /hide_agent_reasoning = false/u)
  assert.match(content, /\[windows\]\nsandbox = "elevated"/u)
  assert.doesNotMatch(content, /projects\./u)
  assert.doesNotMatch(content, /notify =/u)
})

test('clientConfigTemplates: opencode 只包含四模型并按选择更新 agent 默认模型', () => {
  const content = renderClientConfigTemplate({
    tool: 'opencode',
    os: 'mac',
    baseUrl: 'http://localhost:8400/v1/',
    apiKey: 'ogw_local',
    model: 'gpt-5.6-luna',
  })
  const parsed = JSON.parse(content)

  assert.equal(
    parsed.provider['oauth-api-service'].options.baseURL,
    'http://localhost:8400/v1'
  )
  assert.equal(parsed.provider['oauth-api-service'].options.apiKey, 'ogw_local')
  assert.equal(parsed.agent.build.model, 'oauth-api-service/gpt-5.6-luna')
  assert.equal(parsed.agent.plan.model, 'oauth-api-service/gpt-5.6-luna')
  assert.equal(parsed.agent.build.variant, 'medium')
  assert.equal(parsed.agent.plan.variant, 'medium')
  assert.equal(
    parsed.provider['oauth-api-service'].models['gpt-5.6-sol'].reasoningEffort,
    'medium'
  )
  assert.equal(
    parsed.provider['oauth-api-service'].models['gpt-5.6-sol'].variants.xhigh
      .reasoningEffort,
    'xhigh'
  )
  assert.deepEqual(
    Object.keys(parsed.provider['oauth-api-service'].models),
    expectedModels
  )
  for (const model of expectedModels) {
    assert.equal(
      parsed.provider['oauth-api-service'].models[model].variants.xhigh
        .reasoningEffort,
      'xhigh'
    )
  }
})

test('clientConfigTemplates: Codex profile v2 文件名和路径区分 mac 与 Windows', () => {
  assert.equal(
    getClientConfigFilename('codex', 'mac', 'team main'),
    'team-main.config.toml'
  )
  assert.equal(
    getClientConfigFilename('codex', 'win', 'team main'),
    'team-main.config.toml'
  )
  assert.equal(getClientConfigFilename('opencode', 'mac'), 'opencode.json')
  assert.equal(getClientConfigFilename('opencode', 'win'), 'opencode.json')
  assert.equal(
    getClientConfigInstallPath('codex', 'mac', 'team main'),
    '~/.codex/team-main.config.toml'
  )
  assert.equal(
    getClientConfigInstallPath('codex', 'win', 'team main'),
    '%USERPROFILE%\\.codex\\team-main.config.toml'
  )
  assert.equal(normalizeProfile('  team/main 中文  '), 'team-main')
})

test('clientConfigTemplates: 模型目录固定且未知模型回退 Sol', () => {
  assert.deepEqual(
    Array.from(CLIENT_CONFIG_MODEL_OPTIONS, (option) => option.value),
    expectedModels
  )
  assert.equal(normalizeModel('gpt-5.5'), 'gpt-5.5')
  assert.equal(normalizeModel('gpt-4.1'), 'gpt-5.6-sol')
})
