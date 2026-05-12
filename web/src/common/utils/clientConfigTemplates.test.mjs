import assert from 'node:assert/strict'
import test from 'node:test'
import fs from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'

function loadClientConfigTemplateModule() {
  const filePath = path.resolve(import.meta.dirname, './clientConfigTemplates.js')
  const source = fs.readFileSync(filePath, 'utf8')
  const transformed = source
    .replace(/export const /g, 'const ')
    .replace(/export function /g, 'function ')
    .concat(
      '\nmodule.exports = { CLIENT_CONFIG_DEFAULTS, normalizeBaseUrl, normalizeApiKey, normalizeProfile, getClientConfigTemplate, renderClientConfigTemplate, getClientConfigFilename, getClientConfigInstallPath };\n'
    )

  const sandbox = {
    module: { exports: {} },
    exports: {},
  }
  vm.runInNewContext(transformed, sandbox, { filename: filePath })
  return sandbox.module.exports
}

const {
  getClientConfigFilename,
  getClientConfigInstallPath,
  normalizeProfile,
  renderClientConfigTemplate,
} = loadClientConfigTemplateModule()

test('clientConfigTemplates: Codex Windows 模板只渲染必要 provider 字段和 Windows 沙箱', () => {
  const content = renderClientConfigTemplate({
    tool: 'codex',
    os: 'win',
    baseUrl: 'https://example.com/v1/',
    apiKey: 'ogw_test_key',
    profile: 'team main',
  })

  assert.match(content, /base_url = "https:\/\/example\.com\/v1"/u)
  assert.match(content, /experimental_bearer_token = "ogw_test_key"/u)
  assert.match(content, /profile = "team-main"/u)
  assert.match(content, /model_reasoning_effort = "medium"/u)
  assert.match(content, /\[windows\]\nsandbox = "elevated"/u)
  assert.doesNotMatch(content, /projects\./u)
  assert.doesNotMatch(content, /notify =/u)
})

test('clientConfigTemplates: opencode 模板替换 baseURL、apiKey 并保留模型变体', () => {
  const content = renderClientConfigTemplate({
    tool: 'opencode',
    os: 'mac',
    baseUrl: 'http://localhost:8400/v1/',
    apiKey: 'ogw_local',
    profile: 'ignored-by-opencode',
  })
  const parsed = JSON.parse(content)

  assert.equal(parsed.provider['oauth-api-service'].options.baseURL, 'http://localhost:8400/v1')
  assert.equal(parsed.provider['oauth-api-service'].options.apiKey, 'ogw_local')
  assert.equal(parsed.agent.build.model, 'oauth-api-service/gpt-5.5')
  assert.equal(parsed.agent.build.variant, 'medium')
  assert.equal(parsed.agent.plan.variant, 'medium')
  assert.equal(parsed.provider['oauth-api-service'].models['gpt-5.5'].reasoningEffort, 'medium')
  assert.equal(
    parsed.provider['oauth-api-service'].models['gpt-5.5'].variants.xhigh.reasoningEffort,
    'xhigh'
  )
  assert.deepEqual(Object.keys(parsed.provider['oauth-api-service'].models), ['gpt-5.5'])
})

test('clientConfigTemplates: 下载文件名使用真实配置文件名，安装路径区分 mac 和 win', () => {
  assert.equal(getClientConfigFilename('codex', 'mac'), 'config.toml')
  assert.equal(getClientConfigFilename('codex', 'win'), 'config.toml')
  assert.equal(getClientConfigFilename('opencode', 'mac'), 'opencode.json')
  assert.equal(getClientConfigFilename('opencode', 'win'), 'opencode.json')
  assert.equal(getClientConfigInstallPath('codex', 'mac'), '~/.codex/config.toml')
  assert.equal(getClientConfigInstallPath('codex', 'win'), '%USERPROFILE%\\.codex\\config.toml')
  assert.equal(normalizeProfile('  team/main 中文  '), 'team-main')
})
