import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import fs from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'
import { setTimeout as delay } from 'node:timers/promises'

import { chromium } from 'playwright'

const webDir = path.resolve(import.meta.dirname, '..')
const outputDir = path.resolve(webDir, 'output', 'playwright', 'style-l1')
const devServerPort = Number(process.env.STYLE_L1_PORT || 4173)
const externalBaseURL = String(process.env.STYLE_L1_BASE_URL || '').trim()
const baseURL = externalBaseURL || `http://127.0.0.1:${devServerPort}`
const headless = process.env.HEADED !== '1'
const scenarioFilter = String(process.env.STYLE_L1_SCENARIOS || '')
  .split(',')
  .map((name) => name.trim())
  .filter(Boolean)
const CODEX_MODEL_IDS = [
  'gpt-5.5',
  'gpt-5.4',
  'gpt-5.4-mini',
  'gpt-5.3-codex',
  'gpt-5.3-codex-spark',
  'gpt-5.2',
]

let devServerProcess = null
let devServerLogs = ''

const scenarios = [
  {
    name: 'home-desktop',
    path: '/',
    viewport: { width: 1440, height: 900 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectRole(page, 'button', '登录')
      await expectText(page, 'Saurick API Console')
    },
  },
  {
    name: 'home-mobile',
    path: '/',
    viewport: { width: 390, height: 844 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectRole(page, 'button', '登录')
      await expectText(page, 'Saurick API Console')
    },
  },
  {
    name: 'login-desktop',
    path: '/login',
    viewport: { width: 1280, height: 800 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectRole(page, 'button', '登录')
      await expectText(page, 'Saurick API Console')
      await assertThemeToggle(page, 'login-desktop', '.admin-login-shell')
    },
  },
  {
    name: 'register-mobile',
    path: '/register',
    viewport: { width: 390, height: 844 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectRole(page, 'button', '登录')
    },
  },
  {
    name: 'legacy-oauth-login-redirect',
    path: '/oauth-login',
    viewport: { width: 1280, height: 800 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectRole(page, 'button', '登录')
    },
  },
  {
    name: 'admin-login-mobile',
    path: '/admin-login',
    viewport: { width: 390, height: 844 },
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectRole(page, 'button', '登录')
      await expectText(page, 'Saurick API Console')
    },
  },
  {
    name: 'admin-login-desktop',
    path: '/admin-login',
    viewport: { width: 1440, height: 900 },
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectRole(page, 'button', '登录')
      await expectText(page, 'Saurick API Console')
    },
  },
  {
    name: 'admin-menu-redirect',
    path: '/admin-menu',
    viewport: { width: 1280, height: 800 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectRole(page, 'button', '登录')
    },
  },
  {
    name: 'portal-redirect',
    path: '/portal',
    viewport: { width: 390, height: 844 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectRole(page, 'button', '登录')
    },
  },
  {
    name: 'admin-analytics-redirect',
    path: '/admin-analytics',
    viewport: { width: 1280, height: 800 },
    expectPath: '/admin-usage',
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectText(page, '用量日志')
      await expectText(page, '每日模型')
      await assertAdminChrome(page, 'admin-analytics-redirect')
      await assertUsageTableVisuals(page, 'admin-analytics-redirect')
    },
  },
  {
    name: 'admin-usage-desktop',
    path: '/admin-usage',
    viewport: { width: 1440, height: 900 },
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectText(page, '用量日志')
      await expectText(page, '每日模型')
      await expectText(page, '时间范围')
      await expectText(page, '24h 范围内第')
      await expectNoText(page, '返回控制台')
      await assertAdminChrome(page, 'admin-usage-desktop')
      await assertUsageTableVisuals(page, 'admin-usage-desktop')
    },
  },
  {
    name: 'admin-usage-mobile',
    path: '/admin-usage',
    viewport: { width: 390, height: 844 },
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectText(page, '用量日志')
      await expectText(page, '每日模型')
      await expectText(page, '时间范围')
      await expectText(page, '24h 范围内第')
      await expectNoText(page, '返回控制台')
      await assertAdminChrome(page, 'admin-usage-mobile')
      await assertUsageTableVisuals(page, 'admin-usage-mobile')
    },
  },
  {
    name: 'admin-upstream-desktop',
    path: '/admin-upstream',
    viewport: { width: 1440, height: 900 },
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, '上游策略')
      await expectText(page, 'Codex 上游策略')
      await expectText(page, 'Backend 直连')
      await expectText(page, 'Backend + CLI 兜底')
      await expectText(page, '强制 CLI')
      await expectNoText(page, '最近上游请求')
      await expectNoText(page, '每日模型汇总')
      await expectNoText(page, '会话聚合')
      await page.getByRole('tab', { name: '强制 CLI', exact: true }).click()
      await page.waitForFunction(() => {
        const tab = [...document.querySelectorAll('[role="tab"]')].find(
          (node) => node.textContent.trim() === '强制 CLI'
        )
        return tab?.getAttribute('aria-selected') === 'true'
      })
      await page.getByRole('tab', { name: 'Backend 直连', exact: true }).click()
      await page.waitForFunction(() => {
        const tab = [...document.querySelectorAll('[role="tab"]')].find(
          (node) => node.textContent.trim() === 'Backend 直连'
        )
        return tab?.getAttribute('aria-selected') === 'true'
      })
      const calls = page.__styleL1ApiRpcCalls || []
      assert(
        calls.some((call) => call.method === 'gateway_upstream_get') &&
          calls.some((call) => call.method === 'gateway_upstream_set'),
        `admin-upstream-desktop 未调用上游策略读写接口: ${JSON.stringify(calls)}`
      )
    },
  },
  {
    name: 'admin-client-config-desktop',
    path: '/admin-client-config',
    viewport: { width: 1440, height: 900 },
    adminAuth: true,
    verify: async (page) => {
      await expectText(page, '客户端配置生成器')
      await expectText(page, 'Codex')
      await expectText(page, 'opencode')
      await expectRole(page, 'link', '打开公开页')
      await assertAdminChrome(page, 'admin-client-config-desktop')
      await assertClientConfigVisuals(page, 'admin-client-config-desktop')
    },
  },
  {
    name: 'admin-client-config-mobile',
    path: '/admin-client-config',
    viewport: { width: 390, height: 844 },
    adminAuth: true,
    verify: async (page) => {
      await expectText(page, '客户端配置生成器')
      await expectText(page, '配置预览')
      await expectRole(page, 'link', '打开公开页')
      await assertAdminChrome(page, 'admin-client-config-mobile')
      await assertClientConfigVisuals(page, 'admin-client-config-mobile')
    },
  },
  {
    name: 'public-client-config-desktop',
    path: '/client-config',
    viewport: { width: 1440, height: 900 },
    verify: async (page) => {
      await expectText(page, '客户端配置生成器')
      await expectText(page, '公开客户端配置生成器')
      await expectNoText(page, '超级管理员')
      await assertPublicClientConfigVisuals(
        page,
        'public-client-config-desktop'
      )
    },
  },
  {
    name: 'public-client-config-mobile',
    path: '/client-config',
    viewport: { width: 390, height: 844 },
    verify: async (page) => {
      await expectText(page, '客户端配置生成器')
      await expectText(page, '配置预览')
      await expectNoText(page, '超级管理员')
      await assertPublicClientConfigVisuals(page, 'public-client-config-mobile')
    },
  },
  {
    name: 'admin-codex-balance-desktop',
    path: '/admin-codex-balance',
    viewport: { width: 1440, height: 900 },
    adminAuth: true,
    mockCodexBalance: true,
    verify: async (page) => {
      await expectText(page, 'Codex 余额')
      await expectRole(page, 'link', '打开公开接口')
      await expectRole(page, 'button', '刷新')
      await expectText(page, '接口状态')
      await expectText(page, '正常')
      await expectText(page, 'Credits remaining')
      await assertAdminChrome(page, 'admin-codex-balance-desktop')
      await assertThemeToggle(
        page,
        'admin-codex-balance-desktop',
        '.admin-frame'
      )
      await assertCodexBalanceVisuals(page, 'admin-codex-balance-desktop')
    },
  },
  {
    name: 'admin-codex-balance-mobile',
    path: '/admin-codex-balance',
    viewport: { width: 390, height: 844 },
    adminAuth: true,
    mockCodexBalance: true,
    verify: async (page) => {
      await expectText(page, 'Codex 余额')
      await expectRole(page, 'link', '打开公开接口')
      await expectRole(page, 'button', '刷新')
      await expectText(page, '5 小时额度')
      await expectText(page, '每周额度')
      await assertAdminChrome(page, 'admin-codex-balance-mobile')
      await assertCodexBalanceVisuals(page, 'admin-codex-balance-mobile')
    },
  },
  {
    name: 'admin-keys-desktop',
    path: '/admin-keys',
    viewport: { width: 1440, height: 900 },
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'API 凭据')
      await expectRole(page, 'button', '新建 API 凭据')
      await assertAdminChrome(page, 'admin-keys-desktop')
      await assertKeyTableVisuals(page, 'admin-keys-desktop')
    },
  },
  {
    name: 'admin-keys-mobile',
    path: '/admin-keys',
    viewport: { width: 390, height: 844 },
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'API 凭据')
      await expectRole(page, 'button', '新建 API 凭据')
      await assertAdminChrome(page, 'admin-keys-mobile')
      await assertKeyTableVisuals(page, 'admin-keys-mobile')
    },
  },
  {
    name: 'admin-models-desktop',
    path: '/admin-models',
    viewport: { width: 1440, height: 900 },
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, '模型管理')
      await assertAdminChrome(page, 'admin-models-desktop')
      await assertModelTableVisuals(page, 'admin-models-desktop')
      await page.getByRole('button', { name: '浅色', exact: true }).click()
      await page.waitForFunction(
        () => document.documentElement.dataset.adminTheme === 'light'
      )
      await page
        .getByRole('button', { name: '上下文', exact: true })
        .first()
        .click()
      await expectText(page, '模型上下文策略')
      await expectText(page, '开始压缩 tokens')
      await expectText(page, '支持 K / M 单位')
      await expectText(page, '当前生效：260,000 tokens')
      await expectText(page, '不是无限制')
      await expectText(page, '填入当前值')
      await assertModelContextModalLayout(page, 'admin-models-desktop-light')
      await page.getByRole('button', { name: '关闭弹窗' }).click()
      await page.getByRole('button', { name: '暗色', exact: true }).click()
      await page.waitForFunction(
        () => document.documentElement.dataset.adminTheme === 'dark'
      )
      await page
        .getByRole('button', { name: '上下文', exact: true })
        .first()
        .click()
      await assertModelContextModalLayout(page, 'admin-models-desktop-dark')
      await page.getByRole('button', { name: '填入当前值' }).nth(1).click()
      assert.equal(
        await page.getByLabel('开始压缩 tokens').inputValue(),
        '260K'
      )
      await page.getByLabel('开始压缩 tokens').fill('260K')
      await page.getByLabel('硬拦截 tokens').fill('0.38M')
      await page.getByLabel('开始压缩 bytes').fill('1.04M')
      await page.getByLabel('硬拦截 bytes').fill('1.9M')
      await page.getByRole('button', { name: '保存策略' }).click()
      await waitForApiRpcCall(
        page,
        'model_context_update',
        (params) =>
          params.context_compact_tokens === 260000 &&
          params.context_hard_tokens === 380000 &&
          params.context_compact_bytes === 1040000 &&
          params.context_hard_bytes === 1900000
      )
      await page
        .getByRole('button', { name: '上下文', exact: true })
        .first()
        .click()
      await page.getByLabel('保留最近条数').fill('8K')
      await page.getByRole('button', { name: '保存策略' }).click()
      await expectText(page, '保留条数只能填整数')
      await page.getByRole('button', { name: '关闭弹窗' }).click()
    },
  },
  {
    name: 'admin-models-mobile',
    path: '/admin-models',
    viewport: { width: 390, height: 844 },
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, '模型管理')
      await assertAdminChrome(page, 'admin-models-mobile')
      await assertModelTableVisuals(page, 'admin-models-mobile')
    },
  },
  {
    name: 'admin-dashboard-desktop',
    path: '/admin-dashboard',
    viewport: { width: 1440, height: 900 },
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectText(page, '业务看板')
      await expectText(page, '30 天趋势')
      await expectText(page, '最近调用')
      await assertAdminChrome(page, 'admin-dashboard-desktop')
      await assertThemeToggle(page, 'admin-dashboard-desktop', '.admin-frame')
      await assertApiVisuals(page, 'admin-dashboard-desktop')
    },
  },
  {
    name: 'admin-dashboard-mobile',
    path: '/admin-dashboard',
    viewport: { width: 390, height: 844 },
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectText(page, '业务看板')
      await expectText(page, '30 天趋势')
      await expectText(page, '最近调用')
      await assertAdminChrome(page, 'admin-dashboard-mobile')
      await assertApiVisuals(page, 'admin-dashboard-mobile')
    },
  },
  {
    name: 'admin-session-expired-modal-desktop',
    path: '/admin-dashboard',
    viewport: { width: 1440, height: 900 },
    adminAuth: true,
    mockApiRpcAuthExpired: true,
    verify: async (page) => {
      await assertSessionExpiredAlertModal(
        page,
        'admin-session-expired-modal-desktop'
      )
      await page.getByRole('button', { name: '重新登录', exact: true }).click()
      await waitForPath(page, '/admin-login')
    },
  },
  {
    name: 'admin-session-expired-modal-mobile-dark',
    path: '/admin-dashboard',
    viewport: { width: 390, height: 844 },
    adminAuth: true,
    adminTheme: 'dark',
    mockApiRpcAuthExpired: true,
    verify: async (page) => {
      await assertSessionExpiredAlertModal(
        page,
        'admin-session-expired-modal-mobile-dark'
      )
      await page.getByRole('button', { name: '重新登录', exact: true }).click()
      await waitForPath(page, '/admin-login')
    },
  },
  {
    name: 'admin-guide-redirect',
    path: '/admin-guide',
    viewport: { width: 1280, height: 800 },
    expectPath: '/admin-dashboard',
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectText(page, '业务看板')
      await expectNoText(page, '功能路线')
      await assertAdminChrome(page, 'admin-guide-redirect')
      await assertApiVisuals(page, 'admin-guide-redirect')
    },
  },
  {
    name: 'admin-accounts-redirect',
    path: '/admin-accounts',
    viewport: { width: 1280, height: 800 },
    expectPath: '/admin-dashboard',
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectText(page, '业务看板')
      await expectNoText(page, '账号目录')
      await assertAdminChrome(page, 'admin-accounts-redirect')
      await assertApiVisuals(page, 'admin-accounts-redirect')
    },
  },
  {
    name: 'admin-oauth-redirect',
    path: '/admin-oauth',
    viewport: { width: 1280, height: 800 },
    expectPath: '/admin-dashboard',
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'API 管理后台')
      await expectText(page, '业务看板')
      await expectNoText(page, '授权登录')
      await assertAdminChrome(page, 'admin-oauth-redirect')
      await assertApiVisuals(page, 'admin-oauth-redirect')
    },
  },
]

async function main() {
  await fs.mkdir(outputDir, { recursive: true })

  try {
    if (!externalBaseURL) {
      devServerProcess = startDevServer()
      await waitForServer(baseURL)
    }

    const selectedScenarios = scenarioFilter.length
      ? scenarios.filter((scenario) => scenarioFilter.includes(scenario.name))
      : scenarios

    assert(
      selectedScenarios.length > 0,
      `[style:l1] 未找到指定场景: ${scenarioFilter.join(', ')}`
    )

    const browser = await chromium.launch({ headless })
    try {
      for (const scenario of selectedScenarios) {
        await runScenario(browser, scenario)
      }
    } finally {
      await browser.close()
    }

    console.log(`[style:l1] 通过，共验证 ${selectedScenarios.length} 个场景`)
  } finally {
    await stopDevServer()
  }
}

function startDevServer() {
  const child = spawn(
    'pnpm',
    [
      'exec',
      'vite',
      '--host',
      '127.0.0.1',
      '--port',
      String(devServerPort),
      '--strictPort',
    ],
    {
      cwd: webDir,
      env: {
        ...process.env,
        BROWSER: 'none',
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    }
  )

  child.stdout.on('data', (chunk) => {
    devServerLogs += chunk.toString()
  })
  child.stderr.on('data', (chunk) => {
    devServerLogs += chunk.toString()
  })

  child.on('exit', (code) => {
    if (code !== null && code !== 0) {
      devServerLogs += `\n[vite exited with code ${code}]`
    }
  })

  return child
}

async function stopDevServer() {
  if (!devServerProcess) {
    return
  }

  if (devServerProcess.exitCode === null) {
    devServerProcess.kill('SIGTERM')
    await Promise.race([
      new Promise((resolve) => devServerProcess.once('exit', resolve)),
      delay(3000),
    ])
  }

  if (devServerProcess.exitCode === null) {
    devServerProcess.kill('SIGKILL')
  }

  devServerProcess = null
}

async function waitForServer(url) {
  const deadline = Date.now() + 30_000
  let lastError = 'server did not become ready'

  while (Date.now() < deadline) {
    try {
      const response = await fetch(url, {
        redirect: 'manual',
      })
      if (response.ok || response.status === 302 || response.status === 304) {
        return
      }
      lastError = `unexpected status ${response.status}`
    } catch (error) {
      lastError = error.message
    }
    await delay(300)
  }

  throw new Error(
    `[style:l1] 无法启动前端预览：${lastError}\n最近 vite 输出：\n${tailLogs(devServerLogs)}`
  )
}

async function runScenario(browser, scenario) {
  const page = await browser.newPage({ viewport: scenario.viewport })
  const errors = []

  page.on('console', (message) => {
    if (message.type() === 'error') {
      errors.push(`console error: ${message.text()}`)
    }
  })
  page.on('pageerror', (error) => {
    errors.push(`page error: ${error.message}`)
  })

  try {
    if (scenario.adminAuth) {
      await page.addInitScript((token) => {
        window.localStorage.setItem('admin_access_token', token)
      }, createFakeAdminToken())
    }

    await page.addInitScript((theme) => {
      if (theme) {
        window.localStorage.setItem('admin_theme', theme)
      } else if (!window.localStorage.getItem('admin_theme')) {
        window.localStorage.setItem('admin_theme', 'system')
      }
    }, scenario.adminTheme || '')

    if (scenario.userAuth) {
      await page.addInitScript((token) => {
        window.localStorage.setItem('user_access_token', token)
      }, createFakeUserToken())
    }

    await installAuthConfigMock(page)

    if (scenario.mockApiRpcAuthExpired) {
      await installApiRpcAuthExpiredMock(page)
    } else if (scenario.mockApiRpc) {
      await installApiRpcMock(page)
    }

    if (scenario.mockCodexBalance) {
      await installCodexBalanceMock(page)
    }

    await page.goto(new URL(scenario.path, `${baseURL}/`).toString(), {
      waitUntil: 'domcontentloaded',
    })
    await delay(300)

    if (scenario.expectPath) {
      await waitForPath(page, scenario.expectPath)
    }

    await scenario.verify(page)
    await assertNoHorizontalOverflow(page, scenario.name)
    assert.deepEqual(errors, [], `${scenario.name} 出现控制台或运行时错误`)

    const screenshotPath = path.resolve(outputDir, `${scenario.name}.png`)
    await page.screenshot({ path: screenshotPath, fullPage: true })
  } catch (error) {
    throw new Error(
      `[style:l1] 场景失败: ${scenario.name}\n${error.message}\n最近 vite 输出：\n${tailLogs(devServerLogs)}`
    )
  } finally {
    await page.close()
  }
}

async function waitForPath(page, expectedPath) {
  const deadline = Date.now() + 10_000
  while (Date.now() < deadline) {
    if (new URL(page.url()).pathname === expectedPath) {
      return
    }
    await delay(100)
  }
  assert.equal(new URL(page.url()).pathname, expectedPath)
}

async function expectHeading(page, text) {
  const locator = page.getByRole('heading', { name: text })
  await locator.waitFor({ state: 'visible', timeout: 10_000 })
}

async function expectRole(page, role, name) {
  const locator = page.getByRole(role, { exact: true, name })
  await locator.waitFor({ state: 'visible', timeout: 10_000 })
}

async function expectText(page, text) {
  const locator = page.getByText(text, { exact: false })
  await locator.first().waitFor({ state: 'visible', timeout: 10_000 })
}

async function expectNoText(page, text) {
  const hasText = await page.evaluate(
    (value) => document.body.innerText.includes(value),
    text
  )
  assert(!hasText, `页面不应出现文案: ${text}`)
}

async function waitForApiRpcCall(page, method, predicate) {
  const deadline = Date.now() + 10_000
  while (Date.now() < deadline) {
    const matched = (page.__styleL1ApiRpcCalls || []).some(
      (call) => call.method === method && predicate(call.params || {})
    )
    if (matched) {
      return
    }
    await delay(100)
  }
  assert.fail(
    `未捕获符合条件的 ${method} 调用: ${JSON.stringify(
      page.__styleL1ApiRpcCalls || []
    )}`
  )
}

async function assertNoHorizontalOverflow(page, scenarioName) {
  const metrics = await page.evaluate(() => ({
    bodyScrollWidth: document.body.scrollWidth,
    docScrollWidth: document.documentElement.scrollWidth,
    viewportWidth: window.innerWidth,
  }))

  assert(
    metrics.bodyScrollWidth <= metrics.viewportWidth + 2,
    `${scenarioName} body 出现横向溢出: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.docScrollWidth <= metrics.viewportWidth + 2,
    `${scenarioName} document 出现横向溢出: ${JSON.stringify(metrics)}`
  )
}

async function assertThemeToggle(page, scenarioName, rootSelector) {
  const toggle = page.locator('[data-admin-theme-toggle]')
  assert.equal(await toggle.count(), 1, `${scenarioName} 主题切换控件数量异常`)
  await toggle.waitFor({ state: 'visible', timeout: 10_000 })

  const systemOption = page.locator('[data-admin-theme-option="system"]')
  const darkOption = page.locator('[data-admin-theme-option="dark"]')
  const lightOption = page.locator('[data-admin-theme-option="light"]')
  assert.equal(await systemOption.count(), 1, `${scenarioName} 缺少跟系统选项`)
  assert.equal(await darkOption.count(), 1, `${scenarioName} 缺少暗色选项`)
  assert.equal(await lightOption.count(), 1, `${scenarioName} 缺少浅色选项`)

  const systemMetrics = await getThemeMetrics(page, rootSelector)
  assert.equal(
    systemMetrics.mode,
    'system',
    `${scenarioName} 初始模式应为 system`
  )
  assert.equal(
    systemMetrics.systemPressed,
    'true',
    `${scenarioName} 跟系统选项未默认选中`
  )
  assert(
    ['light', 'dark'].includes(systemMetrics.theme),
    `${scenarioName} 系统主题解析异常: ${JSON.stringify(systemMetrics)}`
  )

  await darkOption.click()
  await page.waitForFunction(
    () => document.documentElement.dataset.adminTheme === 'dark'
  )
  await delay(120)

  const darkMetrics = await getThemeMetrics(page, rootSelector)
  assert.equal(darkMetrics.mode, 'dark', `${scenarioName} 未切到暗色模式`)
  assert.equal(
    darkMetrics.storedTheme,
    'dark',
    `${scenarioName} 未持久化暗色主题`
  )
  assert(
    darkMetrics.surfaceLuminance < 60,
    `${scenarioName} 暗色主题面板亮度异常: ${JSON.stringify(darkMetrics)}`
  )
  assert(
    darkMetrics.textLuminance > darkMetrics.surfaceLuminance + 80,
    `${scenarioName} 暗色主题文字对比异常: ${JSON.stringify(darkMetrics)}`
  )
  if (rootSelector === '.admin-frame') {
    assert(
      darkMetrics.headerLuminance < 60,
      `${scenarioName} 暗色主题顶部栏背景异常: ${JSON.stringify(darkMetrics)}`
    )
  }

  await page.reload({ waitUntil: 'domcontentloaded' })
  await page.waitForFunction(
    () =>
      document.documentElement.dataset.adminThemeMode === 'dark' &&
      document.documentElement.dataset.adminTheme === 'dark'
  )
  const persistedMetrics = await getThemeMetrics(page, rootSelector)
  assert.equal(
    persistedMetrics.storedTheme,
    'dark',
    `${scenarioName} 刷新后暗色主题丢失`
  )

  await page.locator('[data-admin-theme-option="system"]').click()
  await page.waitForFunction(
    () => document.documentElement.dataset.adminThemeMode === 'system'
  )
  await delay(120)
  const resetMetrics = await getThemeMetrics(page, rootSelector)
  assert.equal(
    resetMetrics.storedTheme,
    'system',
    `${scenarioName} 跟系统模式未持久化`
  )
}

async function getThemeMetrics(page, rootSelector) {
  return page.evaluate((selector) => {
    const root = document.querySelector(selector)
    const surface =
      document.querySelector('.admin-surface-panel') ||
      document.querySelector(`${selector} [class~='bg-white']`) ||
      root
    const text =
      root?.querySelector('h1, h2, label, .admin-theme-toggle') || root
    const rootStyle = window.getComputedStyle(root)
    const headerStyle = window.getComputedStyle(
      document.querySelector(`${selector} header`) || root
    )
    const surfaceStyle = window.getComputedStyle(surface)
    const textStyle = window.getComputedStyle(text)

    return {
      headerBackground: headerStyle.backgroundColor,
      headerLuminance: getLuminance(headerStyle.backgroundColor),
      rootBackground: rootStyle.backgroundColor,
      storedTheme: window.localStorage.getItem('admin_theme'),
      surfaceBackground: surfaceStyle.backgroundColor,
      surfaceLuminance: getLuminance(surfaceStyle.backgroundColor),
      mode: document.documentElement.dataset.adminThemeMode,
      systemPressed: document
        .querySelector('[data-admin-theme-option="system"]')
        ?.getAttribute('aria-pressed'),
      textColor: textStyle.color,
      textLuminance: getLuminance(textStyle.color),
      theme: document.documentElement.dataset.adminTheme,
    }

    function getLuminance(color) {
      const channels = color
        .match(/\d+(\.\d+)?/gu)
        ?.slice(0, 3)
        .map(Number)
      if (!channels || channels.length < 3) return 0
      return channels[0] * 0.299 + channels[1] * 0.587 + channels[2] * 0.114
    }
  }, rootSelector)
}

async function assertAdminChrome(page, scenarioName) {
  const metrics = await page.evaluate(() => {
    const rectOf = (selector) => {
      const node = document.querySelector(selector)
      if (!node) return null
      const rect = node.getBoundingClientRect()
      return {
        bottom: rect.bottom,
        height: rect.height,
        left: rect.left,
        right: rect.right,
        top: rect.top,
        width: rect.width,
      }
    }

    return {
      aside: rectOf('aside'),
      hasAccountNav: document.body.innerText.includes('账号目录'),
      hasGuideNav: document.body.innerText.includes('功能路线'),
      hasGlobalCustomerFilter: document.body.innerText.includes('全局客户'),
      hasGlobalSalesFilter: document.body.innerText.includes('全局业务员'),
      hasOAuthNav: document.body.innerText.includes('授权登录'),
      hasRefreshCurrentPage: document.body.innerText.includes('刷新当前页'),
      header: rectOf('header'),
      main: rectOf('main'),
      viewportWidth: window.innerWidth,
    }
  })

  assert(metrics.aside, `${scenarioName} 缺少后台侧边栏`)
  assert(metrics.header, `${scenarioName} 缺少后台顶部栏`)
  assert(metrics.main, `${scenarioName} 缺少后台内容区`)
  assert(!metrics.hasGlobalSalesFilter, `${scenarioName} 仍显示全局业务员筛选`)
  assert(!metrics.hasGlobalCustomerFilter, `${scenarioName} 仍显示全局客户筛选`)
  assert(!metrics.hasAccountNav, `${scenarioName} 仍显示账号目录入口`)
  assert(!metrics.hasGuideNav, `${scenarioName} 仍显示功能路线入口`)
  assert(!metrics.hasOAuthNav, `${scenarioName} 仍显示授权登录入口`)
  assert(!metrics.hasRefreshCurrentPage, `${scenarioName} 仍显示刷新当前页入口`)
  assert(metrics.main.width > 0, `${scenarioName} 内容区宽度异常`)
  assert(metrics.main.height > 0, `${scenarioName} 内容区高度异常`)
  assert(
    metrics.aside.left <= 1 && metrics.aside.right <= metrics.header.left + 1,
    `${scenarioName} 后台侧边栏未保持左侧布局: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.aside.right <= metrics.main.left + 1,
    `${scenarioName} 后台侧边栏和内容区发生重叠: ${JSON.stringify(metrics)}`
  )
}

async function assertApiVisuals(page, scenarioName) {
  const metrics = await page.evaluate(() => {
    const headings = Array.from(document.querySelectorAll('h2')).map((node) =>
      node.textContent.trim()
    )
    const main = document.querySelector('main')
    const barTitles = Array.from(main?.querySelectorAll('[title]') || [])
      .map((node) => node.getAttribute('title') || '')
      .filter((title) => /请求|错误|费用|延迟|Token/u.test(title))
    const table = main?.querySelector('table')
    const tableRect = table?.getBoundingClientRect()
    const tableText = table?.innerText || ''

    return {
      barCount: barTitles.length,
      hasCoreCards: [
        '今日消费',
        '今日请求',
        '服务错误率',
        '响应耗时',
        '当前 RPM / TPM',
        '上游分布',
        'API 凭据',
      ].every((text) => document.body.innerText.includes(text)),
      hasDetailsLink: Boolean(main?.querySelector('a[href="/admin-usage"]')),
      hasDistributionPanels:
        headings.includes('模型用量分布') && headings.includes('接口分布'),
      hasRecentCalls: headings.includes('最近调用'),
      hasRecentCallsDetailFields:
        tableText.includes('req_style_l1_prod_1') &&
        tableText.includes('Session：session-style-l1') &&
        tableText.includes('productionapikey') &&
        tableText.includes('ogw_production') &&
        tableText.includes('缓存输入 / 推理输出') &&
        tableText.includes('缓存输入') &&
        tableText.includes('推理输出') &&
        tableText.includes('字节') &&
        tableText.includes('请求 4,096') &&
        tableText.includes('响应 8,192'),
      hasRecentCallsHeaderTooltips:
        table?.querySelectorAll('.admin-th-help[data-tooltip]').length >= 4,
      hasTokenPanel: headings.includes('Token 构成'),
      hasTrendPanel: headings.includes('30 天趋势'),
      trendBarCount: main?.querySelectorAll('[data-trend-bar]').length || 0,
      trendChartTypeButtons: Array.from(
        main?.querySelectorAll('[aria-label="图表类型"] button') || []
      ).map((node) => ({
        pressed: node.getAttribute('aria-pressed') === 'true',
        text: node.textContent.trim(),
      })),
      trendLineCount: main?.querySelectorAll('[data-trend-line]').length || 0,
      trendPointCount: main?.querySelectorAll('[data-trend-point]').length || 0,
      removedSections: ['凭据看板', '30 天按天统计'].filter((text) =>
        document.body.innerText.includes(text)
      ),
      tableHeight: tableRect?.height || 0,
      tableWidth: tableRect?.width || 0,
      trendMetricButtons: Array.from(
        main?.querySelectorAll('[aria-label="趋势指标"] button') || []
      ).map((node) => ({
        pressed: node.getAttribute('aria-pressed') === 'true',
        text: node.textContent.trim(),
      })),
    }
  })

  assert(metrics.hasCoreCards, `${scenarioName} 缺少业务看板核心指标卡`)
  assert(metrics.hasTrendPanel, `${scenarioName} 缺少 30 天趋势面板`)
  assert(metrics.hasTokenPanel, `${scenarioName} 缺少 Token 构成面板`)
  assert(metrics.hasDistributionPanels, `${scenarioName} 缺少用量分布面板`)
  assert(metrics.hasRecentCalls, `${scenarioName} 缺少最近调用面板`)
  assert(
    metrics.hasRecentCallsDetailFields,
    `${scenarioName} 最近调用字段未对齐调用明细: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.hasRecentCallsHeaderTooltips,
    `${scenarioName} 最近调用缺少明细同款表头说明 tooltip`
  )
  assert(metrics.hasDetailsLink, `${scenarioName} 最近调用缺少明细页入口`)
  assert(
    ['请求', '服务错误', '费用', '延迟', 'Token'].every((text) =>
      metrics.trendMetricButtons.some((item) => item.text === text)
    ) &&
      metrics.trendMetricButtons.some(
        (item) => item.text === '请求' && item.pressed
      ),
    `${scenarioName} 业务看板趋势指标切换异常: ${JSON.stringify(metrics.trendMetricButtons)}`
  )
  assert(
    ['柱状', '折线'].every((text) =>
      metrics.trendChartTypeButtons.some((item) => item.text === text)
    ) &&
      metrics.trendChartTypeButtons.some(
        (item) => item.text === '柱状' && item.pressed
      ) &&
      metrics.trendLineCount === 0 &&
      metrics.trendPointCount === 0,
    `${scenarioName} 业务看板图表类型初始状态异常: ${JSON.stringify(metrics)}`
  )
  assert.equal(
    metrics.removedSections.length,
    0,
    `${scenarioName} 业务看板仍残留非必要区块: ${JSON.stringify(metrics.removedSections)}`
  )
  assert(
    metrics.barCount >= 20 && metrics.trendBarCount >= 20,
    `${scenarioName} usage 趋势柱状图数量异常: ${JSON.stringify(metrics)}`
  )
  assert(metrics.tableHeight > 0, `${scenarioName} 最近调用表格高度异常`)
  assert(metrics.tableWidth > 0, `${scenarioName} 最近调用表格宽度异常`)

  await assertDashboardTrendMetricInteraction(page, scenarioName)
  await assertDashboardTrendChartTypeInteraction(page, scenarioName)
  await assertDashboardTrendTooltip(page, scenarioName)
  assertDashboardCompactRequests(page, scenarioName)
}

async function assertDashboardTrendMetricInteraction(page, scenarioName) {
  await page.getByRole('button', { name: '服务错误', exact: true }).click()
  const metrics = await page.evaluate(() => {
    const pressed = Array.from(
      document.querySelectorAll('[aria-label="趋势指标"] button')
    )
      .filter((node) => node.getAttribute('aria-pressed') === 'true')
      .map((node) => node.textContent.trim())
    const barTitles = Array.from(document.querySelectorAll('main [title]')).map(
      (node) => node.getAttribute('title') || ''
    )
    return {
      hasErrorTitle: barTitles.some((title) => title.includes('错误')),
      pressed,
    }
  })
  assert(
    metrics.pressed.length === 1 &&
      metrics.pressed[0] === '服务错误' &&
      metrics.hasErrorTitle,
    `${scenarioName} 趋势指标切换错误后状态异常: ${JSON.stringify(metrics)}`
  )
}

async function assertDashboardTrendChartTypeInteraction(page, scenarioName) {
  await page.getByRole('button', { name: '折线', exact: true }).click()
  const metrics = await page.evaluate(() => {
    const rectOf = (selector) => {
      const node = document.querySelector(selector)
      const rect = node?.getBoundingClientRect()
      if (!rect) return null
      return {
        bottom: rect.bottom,
        height: rect.height,
        left: rect.left,
        right: rect.right,
        top: rect.top,
        width: rect.width,
      }
    }
    const pressed = Array.from(
      document.querySelectorAll('[aria-label="图表类型"] button')
    )
      .filter((node) => node.getAttribute('aria-pressed') === 'true')
      .map((node) => node.textContent.trim())
    return {
      chartRect: rectOf('main [data-trend-chart]'),
      lineBoxRect: rectOf('main [data-trend-line-box]'),
      lineRect: rectOf('main [data-trend-line]'),
      lineCount: document.querySelectorAll('main [data-trend-line]').length,
      pointCount: document.querySelectorAll('main [data-trend-point]').length,
      pressed,
    }
  })
  assert(
    metrics.pressed.length === 1 &&
      metrics.pressed[0] === '折线' &&
      metrics.lineCount === 1 &&
      metrics.pointCount >= 20,
    `${scenarioName} 趋势图切换折线后状态异常: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.chartRect &&
      metrics.lineBoxRect &&
      metrics.lineRect &&
      metrics.lineBoxRect.top >= metrics.chartRect.top &&
      metrics.lineBoxRect.bottom <= metrics.chartRect.bottom &&
      metrics.lineRect.top >= metrics.lineBoxRect.top &&
      metrics.lineRect.bottom <= metrics.lineBoxRect.bottom &&
      metrics.lineRect.height <= metrics.chartRect.height,
    `${scenarioName} 趋势折线绘图区溢出: ${JSON.stringify(metrics)}`
  )
}

async function assertDashboardTrendTooltip(page, scenarioName) {
  const bar = page.locator('main [data-trend-bar]').nth(20)
  await bar.hover()
  const hoverMetrics = await page.evaluate(() => {
    const tooltip = document.querySelector('main [data-trend-tooltip]')
    return {
      text: tooltip?.textContent || '',
      visible: Boolean(tooltip),
    }
  })
  assert(
    hoverMetrics.visible &&
      hoverMetrics.text.includes('服务错误') &&
      hoverMetrics.text.includes('服务错误率') &&
      hoverMetrics.text.includes('总请求'),
    `${scenarioName} 趋势图 hover 未展示错误指标明细: ${JSON.stringify(hoverMetrics)}`
  )

  await page.getByRole('button', { name: 'Token', exact: true }).click()
  await page.locator('main [data-trend-bar]').nth(20).hover()
  const tokenMetrics = await page.evaluate(() => {
    const tooltip = document.querySelector('main [data-trend-tooltip]')
    return {
      text: tooltip?.textContent || '',
      visible: Boolean(tooltip),
    }
  })
  assert(
    tokenMetrics.visible &&
      tokenMetrics.text.includes('总 Token') &&
      tokenMetrics.text.includes('输入') &&
      tokenMetrics.text.includes('输出'),
    `${scenarioName} 趋势图 hover 未展示 Token 明细: ${JSON.stringify(tokenMetrics)}`
  )
}

function assertDashboardCompactRequests(page, scenarioName) {
  const calls = page.__styleL1ApiRpcCalls || []
  assert(
    !calls.some((call) => call.method === 'usage_key_summaries'),
    `${scenarioName} 精简看板不应请求凭据窗口统计: ${JSON.stringify(calls)}`
  )
  assert(
    calls.some((call) => call.method === 'usage_buckets') &&
      calls.some((call) => call.method === 'usage_list'),
    `${scenarioName} 精简看板缺少趋势或最近调用请求: ${JSON.stringify(calls)}`
  )
}

async function assertAnalyticsVisuals(page, scenarioName) {
  const metrics = await page.evaluate(() => {
    const main = document.querySelector('main')
    const table = main?.querySelector('table')
    const tableRect = table?.getBoundingClientRect()
    const mainRect = main?.getBoundingClientRect()

    return {
      hasAnalyticsNav: document.body.innerText.includes('用量统计'),
      hasDimensionTitle: document.body.innerText.includes('凭据维度'),
      hasModelFilter: Boolean(
        main?.querySelector('[role="combobox"][aria-label="按模型筛选"]')
      ),
      hasPagination: Boolean(main?.querySelector('.admin-table-pagination')),
      hasSearchInput: Boolean(
        main?.querySelector('input[placeholder="搜索备注、前缀或后四位"]')
      ),
      hasStatusFilter: Boolean(
        main?.querySelector('[role="combobox"][aria-label="按状态筛选"]')
      ),
      hasTokenStatsWindows: [
        '24h Token',
        '7 天 Token',
        '30 天 Token',
        '180 天 Token',
        '360 天 Token',
        '1 年 Token',
        '3 年 Token',
        '5 年 Token',
      ].every((text) => document.body.innerText.includes(text)),
      mainHeight: mainRect?.height || 0,
      tableHeight: tableRect?.height || 0,
      tableWidth: tableRect?.width || 0,
    }
  })

  assert(metrics.hasAnalyticsNav, `${scenarioName} 缺少用量统计菜单入口`)
  assert(metrics.hasDimensionTitle, `${scenarioName} 缺少凭据维度统计区`)
  assert(metrics.hasSearchInput, `${scenarioName} 缺少凭据统计搜索输入框`)
  assert(metrics.hasModelFilter, `${scenarioName} 缺少凭据统计模型筛选`)
  assert(metrics.hasStatusFilter, `${scenarioName} 缺少凭据统计状态筛选`)
  assert(metrics.hasPagination, `${scenarioName} 缺少凭据统计分页器`)
  assert(
    metrics.hasTokenStatsWindows,
    `${scenarioName} 凭据 token 统计窗口不完整`
  )
  assert(metrics.mainHeight > 0, `${scenarioName} 后台内容区高度异常`)
  assert(metrics.tableHeight > 0, `${scenarioName} 统计表格高度异常`)
  assert(metrics.tableWidth > 0, `${scenarioName} 统计表格宽度异常`)

  assertKeyTokenStatsRequests(page, scenarioName)
}

async function assertUsageTableVisuals(page, scenarioName) {
  const metrics = await page.evaluate(() => {
    const main = document.querySelector('main')
    const table = main?.querySelector('table')
    const tableRect = table?.getBoundingClientRect()
    const mainRect = main?.getBoundingClientRect()
    const tabs = Array.from(main?.querySelectorAll('[role="tab"]') || [])
    const tabTexts = tabs.map((node) => node.textContent.trim())

    return {
      activeUsageTab: tabs
        .find((node) => node.getAttribute('aria-selected') === 'true')
        ?.textContent.trim(),
      hasDetailButton: document.body.innerText.includes('详情'),
      hasDetailsDefault:
        document.body.innerText.includes('调用明细') &&
        document.body.innerText.includes('按请求级 usage 真源直接展示状态'),
      hasPagination: document.body.innerText.includes('共 12 条'),
      hasSidebarUsageNav: document.body.innerText.includes('用量日志'),
      hasTableRefreshAction: Array.from(
        main?.querySelectorAll('button') || []
      ).some((node) => node.textContent.trim().includes('刷新当前页')),
      hasTimeRangeFilter: Boolean(
        main?.querySelector('[role="combobox"][aria-label="时间范围"]')
      ),
      hasStatusCodeFilter: Boolean(
        main?.querySelector('[role="combobox"][aria-label="HTTP 状态码"]')
      ),
      hasUpstreamFilter: Boolean(
        main?.querySelector('[role="combobox"][aria-label="实际执行上游"]')
      ),
      hasErrorTypeFilter: Boolean(
        main?.querySelector('[role="combobox"][aria-label="错误或中断类型"]')
      ),
      hasUpstreamStats: document.body.innerText.includes('上游分布'),
      hasUsageTabs:
        JSON.stringify(tabTexts) ===
        JSON.stringify([
          '调用明细',
          '异常请求',
          '会话聚合',
          '凭据统计',
          '每日模型',
        ]),
      hasUsageWindowSummary: document.body.innerText.includes('24h 范围内第'),
      mainHeight: mainRect?.height || 0,
      tableHeight: tableRect?.height || 0,
      tableWidth: tableRect?.width || 0,
    }
  })

  assert(metrics.hasSidebarUsageNav, `${scenarioName} 缺少后台侧栏 usage 入口`)
  assert(metrics.hasTimeRangeFilter, `${scenarioName} 缺少 usage 时间范围筛选`)
  assert(metrics.hasStatusCodeFilter, `${scenarioName} 缺少 HTTP 状态码筛选`)
  assert(metrics.hasUpstreamFilter, `${scenarioName} 缺少实际上游筛选`)
  assert(metrics.hasErrorTypeFilter, `${scenarioName} 缺少错误 / 中断类型筛选`)
  assert(metrics.hasUpstreamStats, `${scenarioName} 缺少上游分布统计`)
  assert(metrics.hasUsageTabs, `${scenarioName} usage 分段视图顺序异常`)
  assert(
    metrics.activeUsageTab === '调用明细',
    `${scenarioName} usage 默认视图不是调用明细: ${metrics.activeUsageTab}`
  )
  assert(metrics.hasDetailsDefault, `${scenarioName} 缺少调用明细默认视图`)
  assert(
    metrics.hasUsageWindowSummary,
    `${scenarioName} usage 摘要未显示当前时间窗口`
  )
  assert(
    !metrics.hasTableRefreshAction,
    `${scenarioName} 主内容区不应再显示表格级刷新按钮`
  )
  assert(metrics.mainHeight > 0, `${scenarioName} 后台内容区高度异常`)
  assert(metrics.tableHeight > 0, `${scenarioName} usage 表格高度异常`)
  assert(metrics.tableWidth > 0, `${scenarioName} usage 表格宽度异常`)
  assertUsageAggregationRequests(page, scenarioName)
  await assertUsageDetailsTab(page, scenarioName)
  await assertUsageKeyMultiFilterRequest(page, scenarioName)
  await assertUsageTimeRangeRequest(page, scenarioName)
  await assertUsageStatusCodeFilterRequest(page, scenarioName)
  await assertUsagePaginationRequest(page, scenarioName)
  await assertUsageErrorsTab(page, scenarioName)
  await assertUsageSessionTab(page, scenarioName)
  await assertUsageKeyStatsTab(page, scenarioName)
  await assertUsageDailyModelDetail(page, scenarioName)
}

function assertUsageAggregationRequests(page, scenarioName) {
  const calls = page.__styleL1ApiRpcCalls || []
  assert(
    calls.some(
      (call) =>
        call.method === 'usage_buckets' && call.params?.group_by === 'day_model'
    ),
    `${scenarioName} 未请求每日 usage 聚合: ${JSON.stringify(calls)}`
  )
  assert(
    calls.some((call) => call.method === 'usage_session_summaries'),
    `${scenarioName} 未请求会话 usage 聚合: ${JSON.stringify(calls)}`
  )
  assert(
    calls.filter((call) => call.method === 'usage_key_summaries').length >= 8,
    `${scenarioName} 未请求凭据 token 窗口: ${JSON.stringify(calls)}`
  )
}

async function assertUsageDailyModelDetail(page, scenarioName) {
  await page.getByRole('tab', { name: '每日模型', exact: true }).click()
  await expectText(page, '每日模型汇总')
  await expectText(page, 'gpt-5.4')
  await page.getByRole('button', { name: '详情', exact: true }).first().click()
  await expectText(page, '输入 Tokens')
  await expectText(page, '凭据备注')
  await expectText(page, 'productionapikey')
  await expectText(page, 'Reasoning Tokens')
  await expectText(page, '下一页')
  const metrics = await page.evaluate(() => {
    const modal = document.querySelector('.admin-usage-day-model-modal')
    const rect = modal?.getBoundingClientRect()
    const table = modal?.querySelector('table')
    const tableRect = table?.getBoundingClientRect()
    const scroller = modal?.querySelector('.overflow-auto')
    const scrollerRect = scroller?.getBoundingClientRect()
    return {
      hasModelTitle: document.body.innerText.includes('gpt-5.4'),
      hasSuccessColumn: document.body.innerText.includes('成功'),
      height: rect?.height || 0,
      scrollerInside:
        Boolean(rect && scrollerRect) &&
        scrollerRect.left >= rect.left &&
        scrollerRect.right <= rect.right + 1,
      tableHasScrollableWidth:
        Boolean(scroller && tableRect) &&
        scroller.scrollWidth >= scroller.clientWidth,
      width: rect?.width || 0,
    }
  })
  assert(metrics.hasModelTitle, `${scenarioName} 每日模型详情缺少模型标题`)
  assert(metrics.hasSuccessColumn, `${scenarioName} 每日模型详情缺少成功列`)
  assert(
    metrics.width > 300 &&
      metrics.height > 320 &&
      metrics.scrollerInside &&
      metrics.tableHasScrollableWidth,
    `${scenarioName} 每日模型详情弹窗盒模型异常: ${JSON.stringify(metrics)}`
  )
  await page.getByRole('button', { name: '关闭弹窗' }).click()
}

async function assertUsageKeyStatsTab(page, scenarioName) {
  await page.getByRole('tab', { name: '凭据统计', exact: true }).click()
  await expectText(page, '凭据统计')
  await expectText(page, '24h Token')
  await expectText(page, '5 年 Token')
  const metrics = await page.evaluate(() => ({
    hasSearchInput: Boolean(
      document.querySelector('main input[placeholder="搜索备注、前缀或后四位"]')
    ),
    hasStatsRows:
      document.body.innerText.includes('productionapikey') &&
      document.body.innerText.includes('stagingkeylongname'),
  }))
  assert(metrics.hasSearchInput, `${scenarioName} 凭据统计缺少搜索框`)
  assert(metrics.hasStatsRows, `${scenarioName} 凭据统计缺少统计行`)
}

async function assertUsageSessionTab(page, scenarioName) {
  await page.getByRole('tab', { name: '会话聚合', exact: true }).click()
  await expectText(page, '会话聚合')
  await expectText(page, 'session-style-l1')
  await expectText(page, 'productionapikey')
  await expectText(page, '上下文压缩')
  await expectText(page, '2 次压缩')
  await expectText(page, '自动压缩摘要')
  await page.getByRole('button', { name: '详情', exact: true }).first().click()
  await expectText(page, '会话详情')
  await expectText(page, '凭据备注')
  await expectText(page, '请求明细')
  await expectText(page, 'req_style_l1_prod_1')
  const metrics = await page.evaluate(() => {
    const modal = document.querySelector('.admin-usage-session-modal')
    const rect = modal?.getBoundingClientRect()
    return {
      hasSessionID: document.body.innerText.includes('session-style-l1'),
      hasCalls: document.body.innerText.includes('req_style_l1_prod_1'),
      height: rect?.height || 0,
      width: rect?.width || 0,
    }
  })
  assert(metrics.hasSessionID, `${scenarioName} 会话详情缺少 session_id`)
  assert(metrics.hasCalls, `${scenarioName} 会话详情缺少请求明细`)
  assert(
    metrics.width > 300 && metrics.height > 260,
    `${scenarioName} 会话详情弹窗尺寸异常: ${JSON.stringify(metrics)}`
  )
  await page.getByRole('button', { name: '关闭弹窗' }).click()
}

async function assertUsageDetailsTab(page, scenarioName) {
  await page.getByRole('tab', { name: '调用明细', exact: true }).click()
  await expectText(page, '调用明细')
  await expectText(page, '费用估算')
  await expectText(page, '请求')
  await expectText(page, 'Session：session-style-l1')
  await expectText(page, 'productionapikey')
  await expectText(page, '缓存输入 / 推理输出')
  await expectText(page, '缓存输入')
  await expectText(page, 'context compacted')
  await expectText(page, '自动压缩摘要')
  await expectText(page, '推理输出')
  await expectText(page, '字节')
  const metrics = await page.evaluate(() => ({
    hasDetailButton: Boolean(
      document.querySelector('main table button')?.textContent?.includes('详情')
    ),
    hasHeaderTooltips:
      document.querySelectorAll('main table .admin-th-help[data-tooltip]')
        .length >= 4,
    hasPagination: document.body.innerText.includes('共 12 条'),
    hasRequestID: document.body.innerText.includes('req_style_l1_prod_1'),
    hasSessionID: document.body.innerText.includes('session-style-l1'),
    hasTable: Boolean(document.querySelector('main table')),
  }))
  assert(!metrics.hasDetailButton, `${scenarioName} 调用明细不应再有详情按钮`)
  assert(
    metrics.hasHeaderTooltips,
    `${scenarioName} 调用明细缺少表头说明 tooltip`
  )
  assert(metrics.hasPagination, `${scenarioName} 调用明细缺少分页器`)
  assert(metrics.hasRequestID, `${scenarioName} 调用明细缺少 request_id`)
  assert(metrics.hasSessionID, `${scenarioName} 调用明细缺少 session_id`)
  assert(metrics.hasTable, `${scenarioName} 调用明细缺少表格`)
}

async function assertUsageKeyMultiFilterRequest(page, scenarioName) {
  const calls = page.__styleL1ApiRpcCalls || []
  const startIndex = calls.length
  await page.getByRole('combobox', { name: '调用凭据' }).click()
  await page.getByRole('option', { name: 'productionapikey' }).click()
  await page
    .getByRole('option', { name: 'stagingkeylongnameforoverflowcheck' })
    .click()
  await page.getByRole('button', { name: '应用筛选', exact: true }).click()

  const deadline = Date.now() + 5_000
  while (Date.now() < deadline) {
    const matched = calls.slice(startIndex).some((call) => {
      const keyIDs = call.params?.key_ids || []
      return (
        call.method === 'usage_list' &&
        keyIDs.length === 2 &&
        keyIDs.includes(1) &&
        keyIDs.includes(2)
      )
    })
    const hasSummary = await page.evaluate(() => {
      const input = document.querySelector(
        'main [role="combobox"][aria-label="调用凭据"]'
      )
      return input?.value === '已选 2 个凭据'
    })
    if (matched && hasSummary) {
      return
    }
    await delay(100)
  }

  assert.fail(
    `${scenarioName} 多选调用凭据后未按 key_ids 查询: ${JSON.stringify(
      calls.slice(startIndex)
    )}`
  )
}

async function assertUsageErrorsTab(page, scenarioName) {
  const calls = page.__styleL1ApiRpcCalls || []
  const startIndex = calls.length
  await page.getByRole('tab', { name: '异常请求', exact: true }).click()
  await expectText(page, '异常请求')

  const deadline = Date.now() + 5_000
  while (Date.now() < deadline) {
    if (
      calls
        .slice(startIndex)
        .some(
          (call) =>
            call.method === 'usage_list' &&
            call.params?.success === false &&
            call.params?.exclude_error_type === 'client_canceled'
        )
    ) {
      return
    }
    await delay(100)
  }

  assert.fail(
    `${scenarioName} 切换异常请求后未按服务错误口径查询: ${JSON.stringify(
      calls.slice(startIndex)
    )}`
  )
}

async function assertUsageStatusCodeFilterRequest(page, scenarioName) {
  const calls = page.__styleL1ApiRpcCalls || []
  const startIndex = calls.length
  await page.getByRole('combobox', { name: 'HTTP 状态码' }).click()
  await page
    .getByRole('option', { name: '499 Client Closed Request', exact: true })
    .click()
  await page.getByRole('button', { name: '应用筛选', exact: true }).click()

  const deadline = Date.now() + 5_000
  while (Date.now() < deadline) {
    const matched = calls.slice(startIndex).some((call) => {
      return (
        call.method === 'usage_list' &&
        call.params?.offset === 0 &&
        call.params?.status_code === 499
      )
    })
    const hasSelected = await page.evaluate(() =>
      Array.from(document.querySelectorAll('[role="combobox"]')).some(
        (node) =>
          node.getAttribute('aria-label') === 'HTTP 状态码' &&
          node.value === '499 Client Closed Request'
      )
    )
    if (matched && hasSelected) {
      await page.getByRole('button', { name: '重置', exact: true }).click()
      return
    }
    await delay(100)
  }

  assert.fail(
    `${scenarioName} 状态码筛选未按 499 查询: ${JSON.stringify(
      calls.slice(startIndex)
    )}`
  )
}

async function assertUsageTimeRangeRequest(page, scenarioName) {
  const calls = page.__styleL1ApiRpcCalls || []
  const startIndex = calls.length
  await page.getByRole('combobox', { name: '时间范围' }).click()
  const optionTexts = await page.getByRole('option').allTextContents()
  for (const text of [
    '24h',
    '30 天',
    '180 天',
    '1 年',
    '2 年',
    '3 年',
    '5 年',
  ]) {
    assert(
      optionTexts.some((optionText) => optionText.trim() === text),
      `${scenarioName} usage 时间范围缺少 ${text}: ${JSON.stringify(optionTexts)}`
    )
  }
  await page.getByRole('option', { name: '180 天', exact: true }).click()
  await page.getByRole('button', { name: '应用筛选', exact: true }).click()

  const expectedWindowSeconds = 180 * 24 * 60 * 60
  const deadline = Date.now() + 5_000
  while (Date.now() < deadline) {
    const matched = calls.slice(startIndex).some((call) => {
      const startTime = Number(call.params?.start_time)
      const endTime = Number(call.params?.end_time)
      return (
        call.method === 'usage_list' &&
        call.params?.offset === 0 &&
        Number.isFinite(startTime) &&
        Number.isFinite(endTime) &&
        Math.abs(endTime - startTime - expectedWindowSeconds) <= 2
      )
    })
    const hasSummary = await page.evaluate(() =>
      document.body.innerText.includes('180 天 范围内第')
    )
    if (matched && hasSummary) {
      return
    }
    await delay(100)
  }

  assert.fail(
    `${scenarioName} 切换 usage 时间范围后未请求 180 天窗口: ${JSON.stringify(
      calls.slice(startIndex)
    )}`
  )
}

async function assertUsagePaginationRequest(page, scenarioName) {
  const calls = page.__styleL1ApiRpcCalls || []
  const startIndex = calls.length
  const firstPagination = page.locator('.admin-table-pagination').first()
  await firstPagination.getByRole('button', { name: '下一页' }).click()

  const deadline = Date.now() + 5_000
  while (Date.now() < deadline) {
    if (
      calls
        .slice(startIndex)
        .some(
          (call) => call.method === 'usage_list' && call.params?.offset === 8
        )
    ) {
      await firstPagination.getByRole('button', { name: '上一页' }).click()
      return
    }
    await delay(100)
  }

  assert.fail(
    `${scenarioName} 点击 usage 下一页后未请求 offset=8: ${JSON.stringify(
      calls.slice(startIndex)
    )}`
  )
}

async function assertClientConfigVisuals(page, scenarioName, options = {}) {
  const { requireAdminNav = true, requirePublicNotice = false } = options
  const metrics = await page.evaluate(() => {
    const main = document.querySelector('main')
    const pre = main?.querySelector('pre')
    const preRect = pre?.getBoundingClientRect()
    const mainRect = main?.getBoundingClientRect()
    const preStyle = window.getComputedStyle(pre)
    const text = document.body.innerText
    return {
      apiKeyInputPlaceholder:
        document.querySelector('input[placeholder*="API Key"]')?.placeholder ||
        '',
      apiKeyInputValue:
        document.querySelector('input[placeholder*="API Key"]')?.value ?? null,
      hasClientConfigNav: text.includes('客户端配置'),
      hasPublicClientConfigLink:
        document
          .querySelector('a[href="/client-config"]')
          ?.textContent.trim() === '打开公开页',
      hasNoPersonalStateWarning:
        text.includes('不会导出 Codex 的 auth.json') &&
        text.includes('projects 信任记录') &&
        text.includes('本机绝对路径'),
      hasPublicLocalNotice:
        text.includes('API Key 只在当前浏览器里用于生成配置') &&
        text.includes('不会上传到服务器') &&
        text.includes('不会保存到本系统'),
      hasRequiredKeyHint: text.includes('复制或下载前请填写 API Key'),
      hasNoRepositoryHint: !text.includes('固化进仓库'),
      hasInstallPath: text.includes('~/.codex/config.toml'),
      hasPlaceholders:
        pre?.innerText.includes('https://oauth-api.saurick.me/v1') &&
        pre?.innerText.includes('<API_KEY>') &&
        pre?.innerText.includes('[profiles."saurick"]'),
      hasNoUploadArea:
        !text.includes('上传已有模板') && !text.includes('选择配置文件'),
      mainHeight: mainRect?.height || 0,
      previewBackground: preStyle.backgroundColor,
      previewContrast: getContrastRatio(
        preStyle.color,
        preStyle.backgroundColor
      ),
      previewHeight: preRect?.height || 0,
      previewTextColor: preStyle.color,
      previewWidth: preRect?.width || 0,
    }

    function getContrastRatio(foreground, background) {
      const fg = getRelativeLuminance(foreground)
      const bg = getRelativeLuminance(background)
      const lighter = Math.max(fg, bg)
      const darker = Math.min(fg, bg)
      return (lighter + 0.05) / (darker + 0.05)
    }

    function getRelativeLuminance(color) {
      const channels = color
        .match(/\d+(\.\d+)?/gu)
        ?.slice(0, 3)
        .map(Number)
      if (!channels || channels.length < 3) return 0
      const [r, g, b] = channels.map((channel) => {
        const value = channel / 255
        return value <= 0.03928
          ? value / 12.92
          : ((value + 0.055) / 1.055) ** 2.4
      })
      return 0.2126 * r + 0.7152 * g + 0.0722 * b
    }
  })

  if (requireAdminNav) {
    assert(metrics.hasClientConfigNav, `${scenarioName} 缺少客户端配置菜单入口`)
    assert(
      metrics.hasPublicClientConfigLink,
      `${scenarioName} 缺少跳转公开配置页的入口`
    )
  }
  if (requirePublicNotice) {
    assert(
      metrics.hasPublicLocalNotice,
      `${scenarioName} 缺少公开页本地生成提示`
    )
  }
  assert(metrics.hasNoUploadArea, `${scenarioName} 不应再显示上传配置文件入口`)
  assert.equal(
    metrics.apiKeyInputValue,
    '',
    `${scenarioName} API Key 输入框不应默认填入占位 key`
  )
  assert(
    metrics.apiKeyInputPlaceholder.includes('ogw_xxx') &&
      metrics.apiKeyInputPlaceholder.includes('sk-xxx'),
    `${scenarioName} API Key 输入框缺少示例 placeholder: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.hasRequiredKeyHint,
    `${scenarioName} 缺少复制下载前填写 key 的提示`
  )
  assert(metrics.hasNoRepositoryHint, `${scenarioName} 不应出现仓库固化文案`)
  assert(metrics.hasPlaceholders, `${scenarioName} 配置预览未渲染默认占位模板`)
  assert(metrics.hasInstallPath, `${scenarioName} 缺少目标安装路径说明`)
  assert(
    metrics.hasNoPersonalStateWarning,
    `${scenarioName} 缺少不导出个人状态的安全提示`
  )
  assert(metrics.mainHeight > 0, `${scenarioName} 后台内容区高度异常`)
  assert(metrics.previewHeight > 0, `${scenarioName} 配置预览高度异常`)
  assert(metrics.previewWidth > 0, `${scenarioName} 配置预览宽度异常`)
  assert(
    metrics.previewContrast >= 7,
    `${scenarioName} 浅色模式代码预览对比度不足: ${JSON.stringify(metrics)}`
  )

  await assertClientConfigRequiresApiKey(page, scenarioName)

  await page.locator('[data-admin-theme-option="dark"]').click()
  await page.waitForFunction(
    () => document.documentElement.dataset.adminTheme === 'dark'
  )
  const darkMetrics = await page.evaluate(() => {
    const panel = document.querySelector('.admin-surface-panel')
    const label = document.querySelector('main label')
    const pre = document.querySelector('main pre')
    const luminance = (color) => {
      const channels = color
        .match(/\d+(\.\d+)?/gu)
        ?.slice(0, 3)
        .map(Number)
      if (!channels || channels.length < 3) return 0
      return channels[0] * 0.299 + channels[1] * 0.587 + channels[2] * 0.114
    }
    const panelStyle = window.getComputedStyle(panel)
    const labelStyle = window.getComputedStyle(label)
    const preStyle = window.getComputedStyle(pre)
    return {
      labelLuminance: luminance(labelStyle.color),
      panelLuminance: luminance(panelStyle.backgroundColor),
      preContrast: getContrastRatio(preStyle.color, preStyle.backgroundColor),
      preHeight: pre.getBoundingClientRect().height,
      preLuminance: luminance(preStyle.backgroundColor),
    }

    function getContrastRatio(foreground, background) {
      const fg = getRelativeLuminance(foreground)
      const bg = getRelativeLuminance(background)
      const lighter = Math.max(fg, bg)
      const darker = Math.min(fg, bg)
      return (lighter + 0.05) / (darker + 0.05)
    }

    function getRelativeLuminance(color) {
      const channels = color
        .match(/\d+(\.\d+)?/gu)
        ?.slice(0, 3)
        .map(Number)
      if (!channels || channels.length < 3) return 0
      const [r, g, b] = channels.map((channel) => {
        const value = channel / 255
        return value <= 0.03928
          ? value / 12.92
          : ((value + 0.055) / 1.055) ** 2.4
      })
      return 0.2126 * r + 0.7152 * g + 0.0722 * b
    }
  })
  assert(
    darkMetrics.panelLuminance < 60 &&
      darkMetrics.labelLuminance > darkMetrics.panelLuminance + 80,
    `${scenarioName} 暗色模式配置页文字对比异常: ${JSON.stringify(darkMetrics)}`
  )
  assert(darkMetrics.preHeight > 0, `${scenarioName} 暗色模式预览区高度异常`)
  assert(
    darkMetrics.preContrast >= 7,
    `${scenarioName} 暗色模式代码预览对比度不足: ${JSON.stringify(darkMetrics)}`
  )

  await page.locator('[data-admin-theme-option="system"]').click()
  await page.waitForFunction(
    () => document.documentElement.dataset.adminThemeMode === 'system'
  )
}

async function assertClientConfigRequiresApiKey(page, scenarioName) {
  await page.getByRole('button', { name: '复制内容', exact: true }).click()
  await page
    .getByRole('dialog', { name: '请先填写 API Key', exact: true })
    .waitFor({ state: 'visible' })

  const dialogMetrics = await page.evaluate(() => {
    const dialog = document.querySelector('[role="dialog"]')
    const text = dialog?.innerText || ''
    const activeTag = document.activeElement?.tagName || ''
    return {
      activeTag,
      hasApiKeyMessage: text.includes('复制或下载配置前需要填写 API Key'),
      hasLocalOnlyMessage:
        text.includes('API Key 只会在当前浏览器里用于生成配置') &&
        text.includes('不会上传到服务器'),
      hasNativeAlert: false,
      title:
        dialog?.querySelector('.admin-modal-title')?.textContent.trim() || '',
    }
  })
  assert.equal(
    dialogMetrics.title,
    '请先填写 API Key',
    `${scenarioName} API Key 缺失弹窗标题异常: ${JSON.stringify(dialogMetrics)}`
  )
  assert(
    dialogMetrics.hasApiKeyMessage && dialogMetrics.hasLocalOnlyMessage,
    `${scenarioName} API Key 缺失弹窗文案异常: ${JSON.stringify(dialogMetrics)}`
  )

  await page.getByRole('button', { name: '去填写', exact: true }).click()
  await page
    .getByRole('dialog', { name: '请先填写 API Key', exact: true })
    .waitFor({ state: 'hidden' })
  await page.waitForFunction(() =>
    document.activeElement?.labels?.[0]?.textContent.includes('API Key')
  )
}

async function assertPublicClientConfigVisuals(page, scenarioName) {
  await assertClientConfigVisuals(page, scenarioName, {
    requireAdminNav: false,
    requirePublicNotice: true,
  })

  await page.getByLabel('Base URL').fill('https://proxy.example.test/v1/')
  await page.getByLabel('API Key').fill('ogw_demo_public_key')
  await page.getByRole('tab', { name: 'opencode', exact: true }).click()
  await page.waitForFunction(() => {
    const pre = document.querySelector('main pre')
    return (
      pre?.innerText.includes('"baseURL": "https://proxy.example.test/v1"') &&
      pre?.innerText.includes('"apiKey": "ogw_demo_public_key"') &&
      pre?.innerText.includes('"model": "oauth-api-service/gpt-5.5"')
    )
  })

  const publicMetrics = await page.evaluate(() => {
    const text = document.body.innerText
    const adminLinks = [...document.querySelectorAll('a[href^="/admin-"]')]
    const rpcResources = performance
      .getEntriesByType('resource')
      .filter((entry) => entry.name.includes('/rpc/api'))
    return {
      adminLinkCount: adminLinks.length,
      hasAdminHeader: text.includes('API 管理后台'),
      hasLogout: text.includes('退出'),
      hasSuperAdminBadge: text.includes('超级管理员'),
      rpcResourceCount: rpcResources.length,
    }
  })

  assert.equal(
    publicMetrics.adminLinkCount,
    0,
    `${scenarioName} 公开页不应暴露后台导航链接: ${JSON.stringify(publicMetrics)}`
  )
  assert.equal(
    publicMetrics.rpcResourceCount,
    0,
    `${scenarioName} 公开页不应调用后台 RPC: ${JSON.stringify(publicMetrics)}`
  )
  assert(
    !publicMetrics.hasAdminHeader &&
      !publicMetrics.hasLogout &&
      !publicMetrics.hasSuperAdminBadge,
    `${scenarioName} 公开页不应显示后台壳内容: ${JSON.stringify(publicMetrics)}`
  )
}

async function assertCodexBalanceVisuals(page, scenarioName) {
  const metrics = await page.evaluate(() => {
    const main = document.querySelector('main')
    const link = document.querySelector('a[href="/public/codex/balance"]')
    const refreshButton = Array.from(document.querySelectorAll('button')).find(
      (node) => node.textContent.trim() === '刷新'
    )
    const panels = Array.from(
      main?.querySelectorAll('.admin-surface-panel') || []
    )
    const progressBars = Array.from(
      main?.querySelectorAll('[style*="width"]') || []
    )
    const linkRect = link?.getBoundingClientRect()
    const buttonRect = refreshButton?.getBoundingClientRect()
    return {
      buttonHeight: buttonRect?.height || 0,
      buttonWidth: buttonRect?.width || 0,
      hasCodexCard: document.body.innerText.includes('codex · prolite'),
      hasCreditsZero:
        document.body.innerText.includes('Credits remaining') &&
        document.body.innerText.includes('0'),
      hasNoError: !document.body.innerText.includes('Codex 余额查询失败'),
      hasSparkCard: document.body.innerText.includes('GPT-5.3-Codex-Spark'),
      linkHeight: linkRect?.height || 0,
      linkRel: link?.getAttribute('rel') || '',
      linkTarget: link?.getAttribute('target') || '',
      linkWidth: linkRect?.width || 0,
      panelCount: panels.length,
      progressBarWidths: progressBars.map(
        (node) => node.getBoundingClientRect().width
      ),
    }
  })

  assert.equal(
    metrics.linkTarget,
    '_blank',
    `${scenarioName} 公开接口链接未新窗口打开`
  )
  assert(
    metrics.linkRel.includes('noreferrer') &&
      metrics.linkRel.includes('noopener'),
    `${scenarioName} 公开接口链接 rel 不完整: ${metrics.linkRel}`
  )
  assert(
    metrics.linkWidth > 0 && metrics.linkHeight > 0,
    `${scenarioName} 公开接口按钮尺寸异常`
  )
  assert(
    metrics.buttonWidth > 0 && metrics.buttonHeight > 0,
    `${scenarioName} 刷新按钮尺寸异常`
  )
  assert(metrics.hasNoError, `${scenarioName} mock 余额接口不应显示失败提示`)
  assert(metrics.hasCreditsZero, `${scenarioName} 余额概览未显示 credits`)
  assert(metrics.hasCodexCard, `${scenarioName} 缺少 Codex 限额卡`)
  assert(metrics.hasSparkCard, `${scenarioName} 缺少 Spark 限额卡`)
  assert(
    metrics.panelCount >= 3,
    `${scenarioName} 余额页卡片数量异常: ${metrics.panelCount}`
  )
  assert(
    metrics.progressBarWidths.length >= 4 &&
      metrics.progressBarWidths.every((width) => width >= 0),
    `${scenarioName} 限额进度条渲染异常: ${JSON.stringify(metrics.progressBarWidths)}`
  )

  await page.locator('[data-admin-theme-option="dark"]').click()
  await page.waitForFunction(
    () => document.documentElement.dataset.adminTheme === 'dark'
  )
  const darkMetrics = await page.evaluate(() => {
    const panel = document.querySelector('.admin-surface-panel')
    const link = document.querySelector('a[href="/public/codex/balance"]')
    const text = document.querySelector('main h1, main h2') || panel
    const panelStyle = window.getComputedStyle(panel)
    const linkStyle = window.getComputedStyle(link)
    const textStyle = window.getComputedStyle(text)
    return {
      linkContrast: getContrastRatio(
        linkStyle.color,
        linkStyle.backgroundColor
      ),
      panelLuminance: luminance(panelStyle.backgroundColor),
      textLuminance: luminance(textStyle.color),
    }

    function luminance(color) {
      const channels = color
        .match(/\d+(\.\d+)?/gu)
        ?.slice(0, 3)
        .map(Number)
      if (!channels || channels.length < 3) return 0
      return channels[0] * 0.299 + channels[1] * 0.587 + channels[2] * 0.114
    }

    function getContrastRatio(foreground, background) {
      const fg = getRelativeLuminance(foreground)
      const bg = getRelativeLuminance(background)
      const lighter = Math.max(fg, bg)
      const darker = Math.min(fg, bg)
      return (lighter + 0.05) / (darker + 0.05)
    }

    function getRelativeLuminance(color) {
      const channels = color
        .match(/\d+(\.\d+)?/gu)
        ?.slice(0, 3)
        .map(Number)
      if (!channels || channels.length < 3) return 0
      const [r, g, b] = channels.map((channel) => {
        const value = channel / 255
        return value <= 0.03928
          ? value / 12.92
          : ((value + 0.055) / 1.055) ** 2.4
      })
      return 0.2126 * r + 0.7152 * g + 0.0722 * b
    }
  })
  assert(
    darkMetrics.panelLuminance < 60 &&
      darkMetrics.textLuminance > darkMetrics.panelLuminance + 80,
    `${scenarioName} 暗色模式余额页文字对比异常: ${JSON.stringify(darkMetrics)}`
  )
  assert(
    darkMetrics.linkContrast >= 4.5,
    `${scenarioName} 暗色模式公开接口按钮对比度不足: ${JSON.stringify(darkMetrics)}`
  )
  await page.locator('[data-admin-theme-option="system"]').click()
  await page.waitForFunction(
    () => document.documentElement.dataset.adminThemeMode === 'system'
  )
}

async function assertKeyTableVisuals(page, scenarioName) {
  const metrics = await page.evaluate(() => {
    const main = document.querySelector('main')
    const table = main?.querySelector('table')
    const tableRect = table?.getBoundingClientRect()
    const mainRect = main?.getBoundingClientRect()
    const createButton = Array.from(
      main?.querySelectorAll('button') || []
    ).find((node) => node.textContent.trim() === '新建 API 凭据')

    return {
      createButtonDisabled: Boolean(createButton?.disabled),
      hasFullPlainKey: document.body.innerText.includes(
        'ogw_productionapikey_8a2c'
      ),
      hasMaskedKey: document.body.innerText.includes('ogw_production…8a2c'),
      hasCopyFullKeyAction: Array.from(
        main?.querySelectorAll('button') || []
      ).some((node) => node.textContent.trim() === '复制完整凭据'),
      hasCurrentOperationRow: document.body.innerText.includes('当前操作'),
      hasBatchResetAction: Array.from(
        main?.querySelectorAll('button') || []
      ).some((node) => node.textContent.trim() === '重置 API key'),
      hasSelectPageCheckbox: Boolean(
        main?.querySelector('thead input[aria-label="选择当前页 API 凭据"]')
      ),
      hasPagination: Boolean(main?.querySelector('.admin-table-pagination')),
      hasRemarkHeader: document.body.innerText.includes('备注'),
      hasKeyIdentityHeader: document.body.innerText.includes('完整凭据'),
      hasCreatedAtHeader: document.body.innerText.includes('创建时间'),
      hasUpdatedAtHeader: document.body.innerText.includes('更新时间'),
      hasUpstreamStrategyHeader: document.body.innerText.includes('上游策略'),
      hasUpstreamStrategyValue:
        document.body.innerText.includes('Backend + CLI 兜底') &&
        document.body.innerText.includes('继承全局默认'),
      hasOperationHeader: Array.from(
        table?.querySelectorAll('thead th') || []
      ).some((node) => node.textContent.trim() === '操作'),
      createdAtCells: Array.from(main?.querySelectorAll('table tbody tr') || [])
        .map((row) => row.children[2]?.textContent.trim() || '')
        .filter(Boolean),
      updatedAtCells: Array.from(main?.querySelectorAll('table tbody tr') || [])
        .map((row) => row.children[3]?.textContent.trim() || '')
        .filter(Boolean),
      hasSearchInput: Boolean(
        main?.querySelector('input[placeholder="搜索备注、前缀或后四位"]')
      ),
      hasSearchAction: Array.from(main?.querySelectorAll('button') || []).some(
        (node) => node.textContent.trim() === '搜索'
      ),
      hasSidebarKeyNav: document.body.innerText.includes('API 凭据'),
      hasStatusFilter: Boolean(
        main?.querySelector('[role="combobox"][aria-label="按状态筛选"]')
      ),
      hasTableRefreshAction: Array.from(
        main?.querySelectorAll('button') || []
      ).some((node) => node.textContent.trim().includes('刷新当前页')),
      hasTableToolbar: Boolean(main?.querySelector('.admin-module-toolbar')),
      hasModelFilter: Boolean(
        main?.querySelector('[role="combobox"][aria-label="按模型筛选"]')
      ),
      hasTokenLimitHeader:
        document.body.innerText.includes('Token 日 / 周限制（百万）'),
      hasTokenLimitValue:
        document.body.innerText.includes('总量：日 1 百万 / 周 5 百万') &&
        document.body.innerText.includes('输入：日 0.8 百万 / 周 4 百万') &&
        document.body.innerText.includes('输出：日 0.2 百万 / 周 1 百万') &&
        document.body.innerText.includes(
          '非缓存输入：日 0.45 百万 / 周 2.25 百万'
        ),
      mainHeight: mainRect?.height || 0,
      tableHeight: tableRect?.height || 0,
      tableLayout: table ? window.getComputedStyle(table).tableLayout : '',
      tableWidth: tableRect?.width || 0,
      keyCells: Array.from(table?.querySelectorAll('tbody tr') || []).map(
        (row) => {
          const cell = row.children[4]
          const value = cell?.querySelector('.admin-key-value-text')
          const button = cell?.querySelector('button')
          const cellRect = cell?.getBoundingClientRect()
          const valueRect = value?.getBoundingClientRect()
          const buttonRect = button?.getBoundingClientRect()
          const valueStyle = value ? window.getComputedStyle(value) : null
          return {
            buttonHeight: buttonRect?.height || 0,
            buttonWidth: buttonRect?.width || 0,
            cellHeight: cellRect?.height || 0,
            cellWidth: cellRect?.width || 0,
            overflowWrap: valueStyle?.overflowWrap || '',
            text: value?.textContent.trim() || '',
            valueHeight: valueRect?.height || 0,
            valueWidth: valueRect?.width || 0,
            wordBreak: valueStyle?.wordBreak || '',
          }
        }
      ),
      selectionCheckboxes: Array.from(
        main?.querySelectorAll('table tbody input[type="checkbox"]') || []
      ).map((node) => {
        const rect = node.getBoundingClientRect()
        const style = window.getComputedStyle(node)
        return {
          height: rect.height,
          minHeight: style.minHeight,
          width: rect.width,
        }
      }),
      statusCells: Array.from(table?.querySelectorAll('tbody tr') || []).map(
        (row) => {
          const cell = row.children[8]
          const badge = cell?.querySelector('span')
          const cellRect = cell?.getBoundingClientRect()
          const badgeRect = badge?.getBoundingClientRect()
          const badgeStyle = badge ? window.getComputedStyle(badge) : null
          return {
            badgeHeight: badgeRect?.height || 0,
            badgeText: badge?.textContent.trim() || '',
            cellHeight: cellRect?.height || 0,
            hasButton: Boolean(cell?.querySelector('button')),
            whiteSpace: badgeStyle?.whiteSpace || '',
          }
        }
      ),
    }
  })

  assert(metrics.hasSidebarKeyNav, `${scenarioName} 缺少后台侧栏 API 凭据入口`)
  assert(metrics.hasFullPlainKey, `${scenarioName} 列表应展示完整 key`)
  assert(!metrics.hasMaskedKey, `${scenarioName} 完整 key 不应被前后截断展示`)
  assert(metrics.hasCopyFullKeyAction, `${scenarioName} 缺少完整 key 复制操作`)
  assert(metrics.hasRemarkHeader, `${scenarioName} 缺少备注列表列`)
  assert(metrics.hasKeyIdentityHeader, `${scenarioName} 缺少完整凭据列`)
  assert(metrics.hasCreatedAtHeader, `${scenarioName} 缺少创建时间列`)
  assert(metrics.hasUpdatedAtHeader, `${scenarioName} 缺少更新时间列`)
  assert(metrics.hasUpstreamStrategyHeader, `${scenarioName} 缺少上游策略列`)
  assert(
    metrics.hasUpstreamStrategyValue,
    `${scenarioName} 缺少 key 级上游策略展示`
  )
  assert(
    !metrics.hasOperationHeader,
    `${scenarioName} API 凭据表格不应再展示行内操作列`
  )
  assert(
    metrics.createdAtCells.length === 8 &&
      metrics.createdAtCells.every(
        (value) => value !== '-' && /\d/.test(value)
      ),
    `${scenarioName} 创建时间列展示异常: ${JSON.stringify(metrics.createdAtCells)}`
  )
  assert(
    metrics.updatedAtCells.length === 8 &&
      metrics.updatedAtCells.every(
        (value) => value !== '-' && /\d/.test(value)
      ),
    `${scenarioName} 更新时间列展示异常: ${JSON.stringify(metrics.updatedAtCells)}`
  )
  assert(metrics.hasTableToolbar, `${scenarioName} 缺少表格筛选工具条`)
  assert(metrics.hasPagination, `${scenarioName} 缺少 key 表格分页器`)
  assert(metrics.hasSearchInput, `${scenarioName} 缺少搜索输入框`)
  assert(
    !metrics.hasSearchAction,
    `${scenarioName} 搜索输入已自动触发，不应再显示搜索按钮`
  )
  assert(metrics.hasModelFilter, `${scenarioName} 缺少模型筛选`)
  assert(metrics.hasStatusFilter, `${scenarioName} 缺少状态筛选`)
  assert(
    !metrics.hasTableRefreshAction,
    `${scenarioName} 主内容区不应再显示表格级刷新按钮`
  )
  assert(metrics.hasCurrentOperationRow, `${scenarioName} 缺少当前操作行`)
  assert(
    metrics.hasBatchResetAction,
    `${scenarioName} 缺少批量重置 API key 操作`
  )
  assert(metrics.hasSelectPageCheckbox, `${scenarioName} 缺少表头全选框`)
  assert(metrics.hasTokenLimitHeader, `${scenarioName} 缺少百万 token 列头`)
  assert(metrics.hasTokenLimitValue, `${scenarioName} 缺少百万 token 换算展示`)
  assert(
    !metrics.createButtonDisabled,
    `${scenarioName} 新建 API 凭据按钮不应默认禁用`
  )
  assert(metrics.mainHeight > 0, `${scenarioName} 后台内容区高度异常`)
  assert(metrics.tableHeight > 0, `${scenarioName} key 表格高度异常`)
  assert(metrics.tableWidth > 0, `${scenarioName} key 表格宽度异常`)
  assert.equal(
    metrics.tableLayout,
    'fixed',
    `${scenarioName} key 表格应使用固定列宽避免 Windows 下凭据列被压窄: ${JSON.stringify(metrics)}`
  )
  assert.equal(metrics.keyCells.length, 8, `${scenarioName} key 凭据列数量异常`)
  for (const [index, keyCell] of metrics.keyCells.entries()) {
    assert(
      keyCell.cellWidth >= 220 && keyCell.valueWidth >= 140,
      `${scenarioName} 第 ${index + 1} 个完整凭据列宽异常，可能导致逐字符竖排: ${JSON.stringify(keyCell)}`
    )
    assert(
      keyCell.valueHeight <= 80,
      `${scenarioName} 第 ${index + 1} 个完整凭据被异常竖排或撑高: ${JSON.stringify(keyCell)}`
    )
    assert(
      keyCell.buttonWidth > 0 && keyCell.buttonHeight > 0,
      `${scenarioName} 第 ${index + 1} 个复制按钮尺寸异常: ${JSON.stringify(keyCell)}`
    )
    assert.equal(
      keyCell.wordBreak,
      'normal',
      `${scenarioName} 第 ${index + 1} 个完整凭据不应使用 break-all: ${JSON.stringify(keyCell)}`
    )
    assert.equal(
      keyCell.overflowWrap,
      'anywhere',
      `${scenarioName} 第 ${index + 1} 个完整凭据应允许超长连续字符串安全换行: ${JSON.stringify(keyCell)}`
    )
  }
  assert.equal(
    metrics.selectionCheckboxes.length,
    8,
    `${scenarioName} key 表格选择框数量异常`
  )
  for (const [index, checkbox] of metrics.selectionCheckboxes.entries()) {
    assert(
      checkbox.width >= 16 && checkbox.width <= 22,
      `${scenarioName} 第 ${index + 1} 个选择框宽度异常: ${JSON.stringify(checkbox)}`
    )
    assert(
      checkbox.height >= 16 && checkbox.height <= 22,
      `${scenarioName} 第 ${index + 1} 个选择框高度异常: ${JSON.stringify(checkbox)}`
    )
    assert.equal(
      checkbox.minHeight,
      '0px',
      `${scenarioName} 第 ${index + 1} 个选择框仍被 input min-height 撑开`
    )
  }
  assert.equal(
    metrics.statusCells.length,
    8,
    `${scenarioName} key 表格状态列数量异常`
  )
  for (const [index, statusCell] of metrics.statusCells.entries()) {
    assert(
      ['启用', '禁用'].includes(statusCell.badgeText),
      `${scenarioName} 第 ${index + 1} 个状态文案异常: ${JSON.stringify(statusCell)}`
    )
    assert.equal(
      statusCell.whiteSpace,
      'nowrap',
      `${scenarioName} 第 ${index + 1} 个状态文案仍可能换行: ${JSON.stringify(statusCell)}`
    )
    assert(
      !statusCell.hasButton,
      `${scenarioName} 第 ${index + 1} 个状态列不应保留行内操作按钮: ${JSON.stringify(statusCell)}`
    )
  }

  await assertKeyCreateModal(page, scenarioName)
  await assertKeyDarkTokenLimitModal(page, scenarioName)
  await assertKeyDoubleClickEdit(page, scenarioName)
  await assertTablePagination(page, scenarioName, {
    nextText: 'extraapikey9',
    previousText: 'productionapikey',
  })
  await assertKeySearchAutoQuery(page, scenarioName)
  await assertKeyTableSelectionInteraction(page, scenarioName)
}

function assertKeyTokenStatsRequests(page, scenarioName) {
  const calls = page.__styleL1ApiRpcCalls || []
  const windowCalls = calls.filter(
    (call) =>
      call.method === 'usage_key_summaries' &&
      Number.isFinite(Number(call.params?.start_time)) &&
      Number.isFinite(Number(call.params?.end_time))
  )
  assert(
    windowCalls.length >= 8,
    `${scenarioName} 凭据 token 统计未请求完整时间窗口: ${JSON.stringify(calls)}`
  )
}

async function assertKeySearchAutoQuery(page, scenarioName) {
  const calls = page.__styleL1ApiRpcCalls || []
  const startIndex = calls.length
  await page.locator('#key-search').fill('prod')

  const deadline = Date.now() + 5_000
  while (Date.now() < deadline) {
    if (
      calls
        .slice(startIndex)
        .some(
          (call) => call.method === 'key_list' && call.params?.search === 'prod'
        )
    ) {
      return
    }
    await delay(100)
  }

  assert.fail(
    `${scenarioName} 输入搜索词后未自动请求 key_list: ${JSON.stringify(
      calls.slice(startIndex)
    )}`
  )
}

async function assertKeyCreateModal(page, scenarioName) {
  await page.getByRole('button', { name: '新建 API 凭据', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '新建 API 凭据' })
  await dialog.waitFor({ state: 'visible' })

  const metrics = await dialog.evaluate((node) => {
    const rect = node.getBoundingClientRect()
    return {
      hasRemarkInput: Boolean(
        node.querySelector('input[placeholder="例如 team1"]')
      ),
      remarkPattern:
        node
          .querySelector('input[placeholder="例如 team1"]')
          ?.getAttribute('pattern') || '',
      hasRemarkHint:
        node.innerText.includes('留空时使用默认备注') &&
        node.innerText.includes('保存备注、额度、模型或上游策略不会重新生成'),
      formNoValidate: Boolean(node.querySelector('form')?.noValidate),
      hasNoResetButton: !Array.from(node.querySelectorAll('button')).some(
        (button) => button.textContent.trim() === '重置 API key'
      ),
      hasTokenLimitInput: Boolean(
        node.querySelector('input[placeholder="0 表示不限，1 = 100 万 token"]')
      ),
      hasDetailedTokenLabels:
        node.innerText.includes('细分 Token 限制') &&
        node.innerText.includes('每日输入 Token（百万）') &&
        node.innerText.includes('每日非缓存输入（百万）') &&
        node.innerText.includes('每日输出 Token（百万）'),
      hasUpstreamStrategySelect: Boolean(
        node.querySelector('[role="combobox"][aria-label="上游策略"]')
      ),
      hasUpstreamStrategyHint:
        node.innerText.includes('默认继承全局') &&
        node.innerText.includes('仅对该 API 凭据后续请求生效'),
      tokenLimitInputCount: node.querySelectorAll(
        'input[placeholder="0 表示不限，1 = 100 万 token"]'
      ).length,
      hasSubmitButton: Array.from(node.querySelectorAll('button')).some(
        (button) => button.textContent.trim() === '生成 API 凭据'
      ),
      height: rect.height,
      width: rect.width,
      viewportHeight: window.innerHeight,
      viewportWidth: window.innerWidth,
    }
  })

  assert(metrics.hasRemarkInput, `${scenarioName} 新建弹窗缺少备注输入框`)
  assert.equal(
    metrics.remarkPattern,
    '[A-Za-z0-9]*',
    `${scenarioName} 备注输入框未限制为字母数字`
  )
  assert(metrics.hasRemarkHint, `${scenarioName} 新建弹窗缺少备注生成口径提示`)
  assert(
    metrics.hasNoResetButton,
    `${scenarioName} 新建弹窗不应显示重置 API key 按钮`
  )
  assert(
    metrics.formNoValidate,
    `${scenarioName} 凭据弹窗应关闭浏览器原生校验，统一走页面中文错误提示`
  )
  assert(
    metrics.hasTokenLimitInput,
    `${scenarioName} 新建弹窗缺少百万 token 输入框`
  )
  assert(
    metrics.hasDetailedTokenLabels,
    `${scenarioName} 新建弹窗缺少细分 token 标签`
  )
  assert(
    metrics.hasUpstreamStrategySelect,
    `${scenarioName} 新建弹窗缺少 key 级上游策略选择`
  )
  assert(
    metrics.hasUpstreamStrategyHint,
    `${scenarioName} 新建弹窗缺少 key 级上游策略提示`
  )
  assert(
    metrics.tokenLimitInputCount >= 8,
    `${scenarioName} 新建弹窗细分 token 输入框数量不足: ${metrics.tokenLimitInputCount}`
  )
  assert(metrics.hasSubmitButton, `${scenarioName} 新建弹窗缺少生成按钮`)
  assert(metrics.width > 0, `${scenarioName} 新建弹窗宽度异常`)
  assert(metrics.height > 0, `${scenarioName} 新建弹窗高度异常`)
  assert(
    metrics.width <= metrics.viewportWidth,
    `${scenarioName} 新建弹窗宽度溢出视口`
  )
  assert(
    metrics.height <= metrics.viewportHeight,
    `${scenarioName} 新建弹窗高度溢出视口`
  )

  const remarkInput = dialog.locator('input[placeholder="例如 team1"]')
  await remarkInput.fill('team-1 中文')
  const sanitizedRemark = await remarkInput.inputValue()
  assert.equal(
    sanitizedRemark,
    'team1',
    `${scenarioName} 备注输入框未过滤非字母数字字符`
  )

  await dialog.getByRole('button', { name: '取消' }).click()
  await dialog.waitFor({ state: 'hidden' })
}

async function assertKeyDarkTokenLimitModal(page, scenarioName) {
  await page.locator('[data-admin-theme-option="dark"]').click()
  await page.waitForFunction(
    () => document.documentElement.dataset.adminTheme === 'dark'
  )
  await assertKeyTableDarkSelectedHover(page, scenarioName)
  await page.getByRole('button', { name: '新建 API 凭据', exact: true }).click()
  const dialog = page.getByRole('dialog', { name: '新建 API 凭据' })
  await dialog.waitFor({ state: 'visible' })

  const metrics = await dialog.evaluate((node) => {
    const panels = Array.from(node.querySelectorAll('[class~="bg-[#f7fbf8]"]'))
    const inputs = Array.from(
      node.querySelectorAll('input[placeholder="0 表示不限，1 = 100 万 token"]')
    )
    const requiredTexts = [
      '总 Token 限制',
      '每日总 Token（百万）',
      '每周总 Token（百万）',
      '细分 Token 限制',
      '每日输入 Token（百万）',
      '每周输入 Token（百万）',
      '每日非缓存输入（百万）',
      '每周非缓存输入（百万）',
      '每日输出 Token（百万）',
      '每周输出 Token（百万）',
    ]
    const missingTexts = requiredTexts.filter(
      (text) => !node.innerText.includes(text)
    )
    const luminance = (color) => {
      const channels = color
        .match(/\d+(\.\d+)?/gu)
        ?.slice(0, 3)
        .map(Number)
      if (!channels || channels.length < 3) return 0
      return channels[0] * 0.299 + channels[1] * 0.587 + channels[2] * 0.114
    }

    const panelMetrics = panels.map((panel) => {
      const label =
        panel.querySelector('.text-sm.font-semibold') ||
        panel.querySelector('label') ||
        panel
      const hint =
        panel.querySelector('[class~="text-xs"]') ||
        panel.querySelector('span') ||
        label
      const panelStyle = window.getComputedStyle(panel)
      const labelStyle = window.getComputedStyle(label)
      const hintStyle = window.getComputedStyle(hint)
      const rect = panel.getBoundingClientRect()
      return {
        background: panelStyle.backgroundColor,
        backgroundLuminance: luminance(panelStyle.backgroundColor),
        borderColor: panelStyle.borderColor,
        height: rect.height,
        hintColor: hintStyle.color,
        hintLuminance: luminance(hintStyle.color),
        labelColor: labelStyle.color,
        labelLuminance: luminance(labelStyle.color),
        width: rect.width,
      }
    })

    const inputMetrics = inputs.map((input) => {
      const style = window.getComputedStyle(input)
      const rect = input.getBoundingClientRect()
      return {
        background: style.backgroundColor,
        backgroundLuminance: luminance(style.backgroundColor),
        color: style.color,
        colorLuminance: luminance(style.color),
        height: rect.height,
        width: rect.width,
      }
    })

    return {
      inputCount: inputs.length,
      inputMetrics,
      missingTexts,
      mode: document.documentElement.dataset.adminTheme,
      panelCount: panels.length,
      panelMetrics,
    }
  })

  assert.equal(metrics.mode, 'dark', `${scenarioName} 未保持暗色主题`)
  assert.equal(
    metrics.panelCount,
    2,
    `${scenarioName} 暗色弹窗限额区块数量异常: ${JSON.stringify(metrics)}`
  )
  assert.equal(
    metrics.inputCount,
    8,
    `${scenarioName} 暗色弹窗 token 限制输入框数量异常: ${JSON.stringify(metrics)}`
  )
  assert.deepEqual(
    metrics.missingTexts,
    [],
    `${scenarioName} 暗色弹窗缺少限制字段: ${JSON.stringify(metrics.missingTexts)}`
  )
  for (const [index, panel] of metrics.panelMetrics.entries()) {
    assert(
      panel.width > 0 && panel.height > 0,
      `${scenarioName} 暗色弹窗第 ${index + 1} 个限额区块盒模型异常: ${JSON.stringify(panel)}`
    )
    assert(
      panel.backgroundLuminance < 70,
      `${scenarioName} 暗色弹窗第 ${index + 1} 个限额区块背景仍偏亮: ${JSON.stringify(panel)}`
    )
    assert(
      panel.labelLuminance > panel.backgroundLuminance + 80,
      `${scenarioName} 暗色弹窗第 ${index + 1} 个标题对比不足: ${JSON.stringify(panel)}`
    )
    assert(
      panel.hintLuminance > panel.backgroundLuminance + 45,
      `${scenarioName} 暗色弹窗第 ${index + 1} 个说明对比不足: ${JSON.stringify(panel)}`
    )
  }
  for (const [index, input] of metrics.inputMetrics.entries()) {
    assert(
      input.width > 0 && input.height > 0,
      `${scenarioName} 暗色弹窗第 ${index + 1} 个 token 输入框盒模型异常: ${JSON.stringify(input)}`
    )
    assert(
      input.colorLuminance > input.backgroundLuminance + 55,
      `${scenarioName} 暗色弹窗第 ${index + 1} 个 token 输入框对比不足: ${JSON.stringify(input)}`
    )
  }

  await dialog.getByRole('button', { name: '取消' }).click()
  await dialog.waitFor({ state: 'hidden' })
}

async function assertKeyTableDarkSelectedHover(page, scenarioName) {
  const rows = page.locator('main table tbody tr')
  await rows.nth(0).click()
  await rows.nth(0).hover()

  const metrics = await page.evaluate(() => {
    const luminance = (color) => {
      const channels = color
        .match(/\d+(\.\d+)?/gu)
        ?.slice(0, 3)
        .map(Number)
      if (!channels || channels.length < 3) return 0
      return channels[0] * 0.299 + channels[1] * 0.587 + channels[2] * 0.114
    }
    const row = document.querySelector(
      'main table tbody tr.admin-table-row-selected'
    )
    const cell = row?.querySelector('td:nth-child(2)')
    const rowStyle = row ? window.getComputedStyle(row) : null
    const cellStyle = cell ? window.getComputedStyle(cell) : null
    const rowRect = row?.getBoundingClientRect()
    const cellRect = cell?.getBoundingClientRect()

    return {
      background: rowStyle?.backgroundColor || '',
      backgroundLuminance: luminance(rowStyle?.backgroundColor || ''),
      cellColor: cellStyle?.color || '',
      cellLuminance: luminance(cellStyle?.color || ''),
      cellRect: cellRect
        ? {
            height: cellRect.height,
            width: cellRect.width,
          }
        : null,
      rowRect: rowRect
        ? {
            height: rowRect.height,
            width: rowRect.width,
          }
        : null,
      selectedRows: document.querySelectorAll(
        'main table tbody tr.admin-table-row-selected'
      ).length,
      theme: document.documentElement.dataset.adminTheme,
    }
  })

  assert.equal(metrics.theme, 'dark', `${scenarioName} 未处于暗色主题`)
  assert.equal(
    metrics.selectedRows,
    1,
    `${scenarioName} 暗色 hover 验证前应只有一条选中行`
  )
  assert(
    metrics.backgroundLuminance > 0 && metrics.backgroundLuminance < 70,
    `${scenarioName} 暗色选中行 hover 背景不应回退到浅色: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.cellLuminance > metrics.backgroundLuminance + 70,
    `${scenarioName} 暗色选中行 hover 文字对比不足: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.rowRect?.height > 0 &&
      metrics.rowRect?.width > 0 &&
      metrics.cellRect?.height > 0 &&
      metrics.cellRect?.width > 0,
    `${scenarioName} 暗色选中行 hover 盒模型异常: ${JSON.stringify(metrics)}`
  )
}

async function assertKeyDoubleClickEdit(page, scenarioName) {
  await page.locator('main table tbody tr').first().dblclick()
  const dialog = page.getByRole('dialog', { name: '编辑 API 凭据' })
  await dialog.waitFor({ state: 'visible' })
  const value = await dialog
    .locator('input[placeholder="例如 team1"]')
    .inputValue()
  assert.equal(
    value,
    'productionapikey',
    `${scenarioName} 双击 key 行未打开对应编辑弹窗`
  )
  const strategyValue = await dialog
    .getByRole('combobox', { name: '上游策略' })
    .inputValue()
  assert(
    strategyValue.includes('Backend + CLI 兜底'),
    `${scenarioName} 编辑弹窗未回显 key 级上游策略: ${strategyValue}`
  )
  const resetMetrics = await dialog.evaluate((node) => {
    const resetPanel = Array.from(
      node.querySelectorAll('[class~="bg-[#f7fbf8]"]')
    ).find((panel) => panel.innerText.includes('重置 API key'))
    const resetButton = Array.from(node.querySelectorAll('button')).find(
      (button) => button.textContent.trim() === '重置 API key'
    )
    const rect = resetPanel?.getBoundingClientRect()
    return {
      hasResetPanel: Boolean(resetPanel),
      hasResetButton: Boolean(resetButton),
      hasLeakCopy:
        node.innerText.includes('如果该 key 已泄密') &&
        node.innerText.includes('旧 key 会立即失效') &&
        node.innerText.includes('新生成的完整 key'),
      panelHeight: rect?.height || 0,
      panelWidth: rect?.width || 0,
    }
  })
  assert(
    resetMetrics.hasResetPanel,
    `${scenarioName} 编辑弹窗缺少重置 API key 区块`
  )
  assert(
    resetMetrics.hasResetButton,
    `${scenarioName} 编辑弹窗缺少重置 API key 按钮`
  )
  assert(resetMetrics.hasLeakCopy, `${scenarioName} 编辑弹窗缺少泄密重置说明`)
  assert(
    resetMetrics.panelWidth > 0 && resetMetrics.panelHeight > 0,
    `${scenarioName} 编辑弹窗重置区块盒模型异常: ${JSON.stringify(resetMetrics)}`
  )
  await dialog.getByRole('button', { name: '重置 API key' }).click()
  const resetDialog = page.getByRole('dialog', { name: '重置 API key' })
  await resetDialog.waitFor({ state: 'visible' })
  const confirmMetrics = await resetDialog.evaluate((node) => ({
    hasDescription:
      node.innerText.includes('确认重置 API 凭据') &&
      node.innerText.includes('旧 key 会立即失效') &&
      node.innerText.includes('新生成的完整 key'),
    hasCancel: Array.from(node.querySelectorAll('button')).some(
      (button) => button.textContent.trim() === '取消'
    ),
    hasConfirm: Array.from(node.querySelectorAll('button')).some(
      (button) => button.textContent.trim() === '重置 API key'
    ),
    rect: {
      width: node.getBoundingClientRect().width,
      height: node.getBoundingClientRect().height,
    },
  }))
  assert(
    confirmMetrics.hasDescription &&
      confirmMetrics.hasCancel &&
      confirmMetrics.hasConfirm,
    `${scenarioName} 单个 key 重置确认弹窗内容异常: ${JSON.stringify(confirmMetrics)}`
  )
  assert(
    confirmMetrics.rect.width > 0 && confirmMetrics.rect.height > 0,
    `${scenarioName} 单个 key 重置确认弹窗盒模型异常: ${JSON.stringify(confirmMetrics)}`
  )
  await resetDialog.getByRole('button', { name: '取消' }).click()
  await resetDialog.waitFor({ state: 'hidden' })
  await dialog.getByRole('button', { name: '取消' }).click()
  await dialog.waitFor({ state: 'hidden' })
}

async function assertTablePagination(
  page,
  scenarioName,
  { nextText, previousText }
) {
  const firstPagination = page.locator('.admin-table-pagination').first()
  const pageSizeInput = firstPagination.getByLabel('每页条数')
  await pageSizeInput.focus()
  await assertOpenSelectMenuNotClipped(page, `${scenarioName} 分页每页条数`, {
    placement: 'top',
  })
  await page.keyboard.press('Escape')
  await pageSizeInput.fill('8')
  await pageSizeInput.press('Enter')
  const paginationMetrics = await firstPagination.evaluate((node) => {
    const current = node.querySelector('.admin-page-button-current')
    const pageSize = node.querySelector('.admin-table-page-size-input')
    const prev = node.querySelector('[aria-label="上一页"]')
    const next = node.querySelector('[aria-label="下一页"]')
    const nextIcon = next?.querySelector('span')
    const currentStyle = current ? window.getComputedStyle(current) : null
    const prevStyle = prev ? window.getComputedStyle(prev) : null
    const nextStyle = next ? window.getComputedStyle(next) : null
    const nextIconStyle = nextIcon ? window.getComputedStyle(nextIcon) : null
    const nextChevronStyle = nextIcon
      ? window.getComputedStyle(nextIcon, '::before')
      : null
    const pageSizeStyle = pageSize ? window.getComputedStyle(pageSize) : null
    const currentRect = current?.getBoundingClientRect()
    const nextRect = next?.getBoundingClientRect()
    const nextIconRect = nextIcon?.getBoundingClientRect()
    const pageSizeRect = pageSize?.getBoundingClientRect()
    const nodeRect = node.getBoundingClientRect()
    return {
      text: node.innerText,
      hasCurrent: Boolean(current),
      hasPrev: Boolean(prev),
      hasNext: Boolean(next),
      currentLabel: current?.textContent?.trim() || '',
      currentBorderRadius: currentStyle?.borderRadius || '',
      currentBorderColor: currentStyle?.borderColor || '',
      currentBackground: currentStyle?.backgroundColor || '',
      pageSizeValue: pageSize?.value || '',
      pageSizeBorderRadius: pageSizeStyle?.borderRadius || '',
      pageSizeContentWidth:
        pageSizeRect && pageSizeStyle
          ? Math.round(
              pageSizeRect.width -
                Number.parseFloat(pageSizeStyle.paddingLeft || '0') -
                Number.parseFloat(pageSizeStyle.paddingRight || '0') -
                Number.parseFloat(pageSizeStyle.borderLeftWidth || '0') -
                Number.parseFloat(pageSizeStyle.borderRightWidth || '0')
            )
          : 0,
      prevArrowFontSize: prevStyle?.fontSize || '',
      nextArrowFontSize: nextStyle?.fontSize || '',
      nextIconDisplay: nextIconStyle?.display || '',
      nextIconLineHeight: nextIconStyle?.lineHeight || '',
      nextIconWidth: Math.round(nextIconRect?.width || 0),
      nextIconHeight: Math.round(nextIconRect?.height || 0),
      nextIconCenterDeltaX:
        nextRect && nextIconRect
          ? Math.round(
              ((nextIconRect.x +
                nextIconRect.width / 2 -
                (nextRect.x + nextRect.width / 2)) *
                100) /
                100
            )
          : null,
      nextIconCenterDeltaY:
        nextRect && nextIconRect
          ? Math.round(
              ((nextIconRect.y +
                nextIconRect.height / 2 -
                (nextRect.y + nextRect.height / 2)) *
                100) /
                100
            )
          : null,
      nextChevronWidth: Math.round(
        Number.parseFloat(nextChevronStyle?.width || '0')
      ),
      nextChevronHeight: Math.round(
        Number.parseFloat(nextChevronStyle?.height || '0')
      ),
      nextChevronBorderTopWidth: nextChevronStyle?.borderTopWidth || '',
      nextChevronBorderRightWidth: nextChevronStyle?.borderRightWidth || '',
      currentWidth: Math.round(currentRect?.width || 0),
      currentHeight: Math.round(currentRect?.height || 0),
      pageSizeWidth: Math.round(pageSizeRect?.width || 0),
      pageSizeHeight: Math.round(pageSizeRect?.height || 0),
      overflowsX: node.scrollWidth > Math.ceil(node.clientWidth),
      overflowsY: node.scrollHeight > Math.ceil(node.clientHeight),
      isVisible: nodeRect.width > 0 && nodeRect.height > 0,
    }
  })
  assert(
    paginationMetrics.isVisible &&
      paginationMetrics.hasPrev &&
      paginationMetrics.hasNext &&
      paginationMetrics.hasCurrent,
    `${scenarioName} 分页缺少 trade-erp 风格箭头或当前页: ${JSON.stringify(
      paginationMetrics
    )}`
  )
  assert.equal(
    paginationMetrics.pageSizeValue,
    '8 条/页',
    `${scenarioName} 分页每页条数未使用 trade-erp 文案: ${JSON.stringify(
      paginationMetrics
    )}`
  )
  assert(
    paginationMetrics.currentLabel === '1' &&
      paginationMetrics.currentWidth >= 34 &&
      paginationMetrics.currentWidth <= 38 &&
      paginationMetrics.currentHeight >= 34 &&
      paginationMetrics.currentHeight <= 38 &&
      paginationMetrics.currentBorderRadius.includes('50%') &&
      paginationMetrics.pageSizeWidth >= 118 &&
      paginationMetrics.pageSizeWidth <= 122 &&
      paginationMetrics.pageSizeHeight <= 38 &&
      paginationMetrics.pageSizeContentWidth >= 78 &&
      !paginationMetrics.overflowsX &&
      !paginationMetrics.overflowsY,
    `${scenarioName} 分页数字页码盒模型异常: ${JSON.stringify(
      paginationMetrics
    )}`
  )
  assert(
    paginationMetrics.prevArrowFontSize === '0px' &&
      paginationMetrics.nextArrowFontSize === '0px' &&
      paginationMetrics.nextIconDisplay === 'flex' &&
      paginationMetrics.nextIconLineHeight === '0px' &&
      paginationMetrics.nextIconWidth === 16 &&
      paginationMetrics.nextIconHeight === 16 &&
      Math.abs(paginationMetrics.nextIconCenterDeltaX) <= 1 &&
      Math.abs(paginationMetrics.nextIconCenterDeltaY) <= 1 &&
      paginationMetrics.nextChevronWidth === 8 &&
      paginationMetrics.nextChevronHeight === 8 &&
      paginationMetrics.nextChevronBorderTopWidth === '2px' &&
      paginationMetrics.nextChevronBorderRightWidth === '2px',
    `${scenarioName} 分页箭头大小或居中异常: ${JSON.stringify(
      paginationMetrics
    )}`
  )
  assert(
    !paginationMetrics.text.includes('第 1 /') &&
      !paginationMetrics.text.includes('1-8'),
    `${scenarioName} 分页仍残留旧范围/页数摘要: ${JSON.stringify(
      paginationMetrics
    )}`
  )
  await pageSizeInput.fill('100')
  await pageSizeInput.press('Enter')
  const pageSize100Metrics = await firstPagination.evaluate((node) => {
    const pageSize = node.querySelector('.admin-table-page-size-input')
    const pageSizeStyle = pageSize ? window.getComputedStyle(pageSize) : null
    const pageSizeRect = pageSize?.getBoundingClientRect()
    return {
      pageSizeValue: pageSize?.value || '',
      pageSizeWidth: Math.round(pageSizeRect?.width || 0),
      pageSizeClientWidth: pageSize?.clientWidth || 0,
      pageSizeScrollWidth: pageSize?.scrollWidth || 0,
      pageSizeContentWidth:
        pageSizeRect && pageSizeStyle
          ? Math.round(
              pageSizeRect.width -
                Number.parseFloat(pageSizeStyle.paddingLeft || '0') -
                Number.parseFloat(pageSizeStyle.paddingRight || '0') -
                Number.parseFloat(pageSizeStyle.borderLeftWidth || '0') -
                Number.parseFloat(pageSizeStyle.borderRightWidth || '0')
            )
          : 0,
    }
  })
  assert(
    pageSize100Metrics.pageSizeValue === '100 条/页' &&
      pageSize100Metrics.pageSizeContentWidth >= 78 &&
      pageSize100Metrics.pageSizeScrollWidth <=
        pageSize100Metrics.pageSizeClientWidth,
    `${scenarioName} 分页每页条数最长选项显示不完整: ${JSON.stringify(
      pageSize100Metrics
    )}`
  )
  await pageSizeInput.fill('8')
  await pageSizeInput.press('Enter')
  assert(
    await page.getByText(previousText).first().isVisible(),
    `${scenarioName} 分页第一页缺少 ${previousText}`
  )
  await firstPagination.getByRole('button', { name: '下一页' }).click()
  assert(
    await page.getByText(nextText).first().isVisible(),
    `${scenarioName} 点击下一页后缺少 ${nextText}`
  )
  await firstPagination.getByRole('button', { name: '上一页' }).click()
  assert(
    await page.getByText(previousText).first().isVisible(),
    `${scenarioName} 点击上一页后未回到 ${previousText}`
  )
}

async function assertOpenSelectMenuNotClipped(
  page,
  scenarioName,
  { placement = null } = {}
) {
  const metrics = await page.evaluate((expectedPlacement) => {
    const root = document.querySelector(
      '.admin-searchable-select[data-open="true"]'
    )
    const menu = root?.querySelector('.admin-searchable-select-menu')
    if (!menu) {
      return { hasMenu: false }
    }

    const input = root.querySelector('.admin-searchable-select-input')
    const inputRect = input?.getBoundingClientRect()
    const menuRect = menu.getBoundingClientRect()
    const clippingAncestors = []
    let node = menu.parentElement
    while (node && node !== document.body) {
      const style = window.getComputedStyle(node)
      const clipsX = ['hidden', 'clip', 'auto', 'scroll'].includes(
        style.overflowX
      )
      const clipsY = ['hidden', 'clip'].includes(style.overflowY)
      const clipsOverflow = clipsX || clipsY
      if (clipsOverflow) {
        const rect = node.getBoundingClientRect()
        const clipsMenuX =
          clipsX &&
          (menuRect.right > rect.right + 1 || menuRect.left < rect.left - 1)
        const clipsMenuY =
          clipsY &&
          (menuRect.top < rect.top - 1 || menuRect.bottom > rect.bottom + 1)
        clippingAncestors.push({
          className:
            typeof node.className === 'string'
              ? node.className
              : String(node.className),
          clipsMenu: clipsMenuX || clipsMenuY,
          overflow: `${style.overflow}/${style.overflowX}/${style.overflowY}`,
          rect: {
            bottom: rect.bottom,
            left: rect.left,
            right: rect.right,
            top: rect.top,
          },
        })
      }
      node = node.parentElement
    }

    return {
      clippingAncestors,
      expectedPlacement,
      hasMenu: true,
      inputRect: inputRect
        ? {
            bottom: inputRect.bottom,
            left: inputRect.left,
            right: inputRect.right,
            top: inputRect.top,
          }
        : null,
      menuRect: {
        bottom: menuRect.bottom,
        height: menuRect.height,
        left: menuRect.left,
        right: menuRect.right,
        top: menuRect.top,
        width: menuRect.width,
      },
      placement: root.getAttribute('data-menu-placement') || 'bottom',
    }
  }, placement)

  assert(metrics.hasMenu, `${scenarioName} 下拉菜单未打开`)
  assert(
    metrics.menuRect.height > 0 && metrics.menuRect.width > 0,
    `${scenarioName} 下拉菜单尺寸异常: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.clippingAncestors.every((item) => !item.clipsMenu),
    `${scenarioName} 下拉菜单被祖先容器裁剪: ${JSON.stringify(metrics)}`
  )
  if (placement) {
    assert.equal(
      metrics.placement,
      placement,
      `${scenarioName} 下拉方向异常: ${JSON.stringify(metrics)}`
    )
  }
  if (placement === 'top') {
    assert(
      metrics.inputRect && metrics.menuRect.bottom <= metrics.inputRect.top - 3,
      `${scenarioName} 下拉菜单未向上展开: ${JSON.stringify(metrics)}`
    )
  }
}

async function assertKeyTableSelectionInteraction(page, scenarioName) {
  const rows = page.locator('main table tbody tr')
  const checkboxes = page.locator('main table tbody input[type="checkbox"]')

  await rows.nth(0).click()
  assert.deepEqual(
    await readKeyTableSelectionState(page),
    {
      checked: [true, false, false, false, false, false, false, false],
      headerChecked: false,
      headerIndeterminate: true,
      selectedRows: 1,
    },
    `${scenarioName} 单击第一行后应只选中第一条`
  )

  await rows.nth(1).click()
  assert.deepEqual(
    await readKeyTableSelectionState(page),
    {
      checked: [false, true, false, false, false, false, false, false],
      headerChecked: false,
      headerIndeterminate: true,
      selectedRows: 1,
    },
    `${scenarioName} 单击第二行后应互斥切换选择`
  )

  await checkboxes.nth(1).click()
  assert.deepEqual(
    await readKeyTableSelectionState(page),
    {
      checked: [false, false, false, false, false, false, false, false],
      headerChecked: false,
      headerIndeterminate: false,
      selectedRows: 0,
    },
    `${scenarioName} 再次点击已选选择框应清空选择`
  )

  const headerCheckbox = page.locator(
    'main table thead input[aria-label="选择当前页 API 凭据"]'
  )
  await headerCheckbox.click()
  assert.deepEqual(
    await readKeyTableSelectionState(page),
    {
      checked: [true, true, true, true, true, true, true, true],
      headerChecked: true,
      headerIndeterminate: false,
      selectedRows: 8,
    },
    `${scenarioName} 表头全选应选中当前页所有凭据`
  )

  await headerCheckbox.click()
  assert.deepEqual(
    await readKeyTableSelectionState(page),
    {
      checked: [false, false, false, false, false, false, false, false],
      headerChecked: false,
      headerIndeterminate: false,
      selectedRows: 0,
    },
    `${scenarioName} 表头全选再次点击应清空当前页选择`
  )

  await rows.nth(1).click()
  assert.deepEqual(
    await readKeyTableSelectionState(page),
    {
      checked: [false, true, false, false, false, false, false, false],
      headerChecked: false,
      headerIndeterminate: true,
      selectedRows: 1,
    },
    `${scenarioName} 选择清空后应可重新单击行选中`
  )

  await checkboxes.nth(2).click()
  assert.deepEqual(
    await readKeyTableSelectionState(page),
    {
      checked: [false, true, true, false, false, false, false, false],
      headerChecked: false,
      headerIndeterminate: true,
      selectedRows: 2,
    },
    `${scenarioName} 选择框应支持多选后批量重置`
  )

  const nativeDialogMessages = []
  const nativeDialogHandler = async (dialog) => {
    nativeDialogMessages.push(dialog.message())
    await dialog.dismiss().catch(() => {})
  }
  page.on('dialog', nativeDialogHandler)
  try {
    await page.getByRole('button', { name: '重置 API key' }).click()
    const resetDialog = page.getByRole('dialog', { name: '批量重置 API key' })
    await resetDialog.waitFor({ state: 'visible' })
    const confirmMetrics = await resetDialog.evaluate((node) => ({
      hasDescription:
        node.innerText.includes('确认重置选中的 2 个 API 凭据') &&
        node.innerText.includes('旧 key 会立即失效') &&
        node.innerText.includes('同步到对应客户端'),
      hasCancel: Array.from(node.querySelectorAll('button')).some(
        (button) => button.textContent.trim() === '取消'
      ),
      hasConfirm: Array.from(node.querySelectorAll('button')).some(
        (button) => button.textContent.trim() === '重置 2 个 key'
      ),
      rect: {
        width: node.getBoundingClientRect().width,
        height: node.getBoundingClientRect().height,
      },
    }))
    assert.equal(
      nativeDialogMessages.length,
      0,
      `${scenarioName} 不应再触发浏览器原生确认框: ${nativeDialogMessages.join(' | ')}`
    )
    assert(
      confirmMetrics.hasDescription &&
        confirmMetrics.hasCancel &&
        confirmMetrics.hasConfirm,
      `${scenarioName} 批量重置确认弹窗内容异常: ${JSON.stringify(confirmMetrics)}`
    )
    assert(
      confirmMetrics.rect.width > 0 && confirmMetrics.rect.height > 0,
      `${scenarioName} 批量重置确认弹窗盒模型异常: ${JSON.stringify(confirmMetrics)}`
    )
    await resetDialog.getByRole('button', { name: '重置 2 个 key' }).click()
  } finally {
    page.off('dialog', nativeDialogHandler)
  }
  await page.getByText('批量重置已完成，共生成 2 个新 key').waitFor()
  const batchMetrics = await page.evaluate(() => ({
    hasCopyAll: Array.from(document.querySelectorAll('button')).some(
      (button) => button.textContent.trim() === '复制全部完整凭据'
    ),
    hasFirstResetKey: document.body.innerText.includes(
      'ogw_stagingkeylongnameforoverflowcheck_reset_2_r2st'
    ),
    hasSecondResetKey: document.body.innerText.includes(
      'ogw_extraapikey3_reset_3_r3st'
    ),
    documentOverflowX: Math.ceil(
      document.documentElement.scrollWidth -
        document.documentElement.clientWidth
    ),
  }))
  assert(batchMetrics.hasCopyAll, `${scenarioName} 批量重置缺少复制全部按钮`)
  assert(
    batchMetrics.hasFirstResetKey && batchMetrics.hasSecondResetKey,
    `${scenarioName} 批量重置未展示所有新 key: ${JSON.stringify(batchMetrics)}`
  )
  assert(
    batchMetrics.documentOverflowX <= 0,
    `${scenarioName} 批量重置结果造成页面横向溢出: ${JSON.stringify(batchMetrics)}`
  )
}

async function readKeyTableSelectionState(page) {
  return page.evaluate(() => ({
    checked: Array.from(
      document.querySelectorAll('main table tbody input[type="checkbox"]')
    ).map((node) => node.checked),
    headerChecked: Boolean(
      document.querySelector(
        'main table thead input[aria-label="选择当前页 API 凭据"]'
      )?.checked
    ),
    headerIndeterminate: Boolean(
      document.querySelector(
        'main table thead input[aria-label="选择当前页 API 凭据"]'
      )?.indeterminate
    ),
    selectedRows: document.querySelectorAll(
      'main table tbody tr.admin-table-row-selected'
    ).length,
  }))
}

async function assertModelTableVisuals(page, scenarioName) {
  const metrics = await page.evaluate((codexModelIDs) => {
    const main = document.querySelector('main')
    const table = main?.querySelector('table')
    const tableRect = table?.getBoundingClientRect()
    const mainRect = main?.getBoundingClientRect()

    return {
      hasDisableButton: Array.from(main?.querySelectorAll('button') || []).some(
        (node) => node.textContent.trim() === '禁用'
      ),
      hasPagination: document.body.innerText.includes('共 6 条'),
      hasFixedListHint:
        document.body.innerText.includes('模型列表随代码固定维护'),
      hasModelCreateButton: Array.from(
        main?.querySelectorAll('button') || []
      ).some((node) => node.textContent.trim() === '新建模型'),
      hasModelEditButton: Array.from(
        main?.querySelectorAll('button') || []
      ).some((node) => node.textContent.trim() === '编辑'),
      hasDeleteButton: Array.from(main?.querySelectorAll('button') || []).some(
        (node) => node.textContent.trim() === '删除'
      ),
      hasCodexModels: codexModelIDs.every((modelID) =>
        document.body.innerText.includes(modelID)
      ),
      hasNonCodexModel: document.body.innerText.includes('gpt-5.5-pro'),
      hasPriceHeaders:
        document.body.innerText.includes('输入 $/1M') &&
        document.body.innerText.includes('缓存输入 $/1M') &&
        document.body.innerText.includes('输出 $/1M'),
      hasPriceValues:
        document.body.innerText.includes('$1.75') &&
        document.body.innerText.includes('$0.175') &&
        document.body.innerText.includes('$14'),
      hasContextHeaders:
        document.body.innerText.includes('上下文窗口') &&
        document.body.innerText.includes('压缩阈值') &&
        document.body.innerText.includes('字节阈值'),
      hasContextValues:
        document.body.innerText.includes('400,000 tokens') &&
        document.body.innerText.includes('260,000 / 380,000') &&
        document.body.innerText.includes('1,040,000 / 1,900,000'),
      hasContextButton: Array.from(main?.querySelectorAll('button') || []).some(
        (node) => node.textContent.trim() === '上下文'
      ),
      hasSidebarModelNav: document.body.innerText.includes('模型管理'),
      mainHeight: mainRect?.height || 0,
      tableHeight: tableRect?.height || 0,
      tableWidth: tableRect?.width || 0,
    }
  }, CODEX_MODEL_IDS)

  assert(metrics.hasSidebarModelNav, `${scenarioName} 缺少后台侧栏模型入口`)
  assert(metrics.hasPagination, `${scenarioName} 缺少模型表格分页器`)
  assert(metrics.hasFixedListHint, `${scenarioName} 缺少固定模型列表说明`)
  assert(!metrics.hasModelCreateButton, `${scenarioName} 不应展示模型新增按钮`)
  assert(!metrics.hasModelEditButton, `${scenarioName} 不应展示模型编辑按钮`)
  assert(!metrics.hasDeleteButton, `${scenarioName} 不应展示模型删除操作`)
  assert(metrics.hasCodexModels, `${scenarioName} Codex 模型展示不完整`)
  assert(!metrics.hasNonCodexModel, `${scenarioName} 不应展示非 Codex 模型`)
  assert(metrics.hasPriceHeaders, `${scenarioName} 缺少模型费用列`)
  assert(metrics.hasPriceValues, `${scenarioName} 缺少模型官方费用展示`)
  assert(metrics.hasContextHeaders, `${scenarioName} 缺少上下文策略列`)
  assert(metrics.hasContextValues, `${scenarioName} 缺少上下文策略数值`)
  assert(metrics.hasContextButton, `${scenarioName} 缺少上下文策略操作`)
  assert(metrics.hasDisableButton, `${scenarioName} 缺少模型启停操作`)
  assert(metrics.mainHeight > 0, `${scenarioName} 后台内容区高度异常`)
  assert(metrics.tableHeight > 0, `${scenarioName} 模型表格高度异常`)
  assert(metrics.tableWidth > 0, `${scenarioName} 模型表格宽度异常`)
}

async function assertModelContextModalLayout(page, scenarioName) {
  const metrics = await page.evaluate(() => {
    const panel = document.querySelector('.admin-model-context-modal')
    const panelRect = panel?.getBoundingClientRect()
    const sections = Array.from(
      panel?.querySelectorAll('.admin-model-context-section') || []
    )
    const fields = Array.from(
      panel?.querySelectorAll('.admin-model-context-field') || []
    ).map((field) => {
      const rect = field.getBoundingClientRect()
      const head = field.querySelector('.admin-model-context-field-head')
      const headRect = head?.getBoundingClientRect()
      const input = field.querySelector('input')
      const inputRect = input?.getBoundingClientRect()
      const fill = field.querySelector('.admin-model-context-fill')
      const fillRect = fill?.getBoundingClientRect()
      return {
        fillBottom: Math.round(fillRect?.bottom || 0),
        fillRight: Math.round(fillRect?.right || 0),
        fillWidth: Math.round(fillRect?.width || 0),
        headBottom: Math.round(headRect?.bottom || 0),
        headClientWidth: head?.clientWidth || 0,
        headScrollWidth: head?.scrollWidth || 0,
        inputLeft: Math.round(inputRect?.left || 0),
        inputRight: Math.round(inputRect?.right || 0),
        inputTop: Math.round(inputRect?.top || 0),
        inputWidth: Math.round(inputRect?.width || 0),
        label: input?.getAttribute('aria-label') || '',
        left: Math.round(rect.left),
        right: Math.round(rect.right),
        width: Math.round(rect.width),
        clientWidth: field.clientWidth,
        scrollWidth: field.scrollWidth,
      }
    })
    const panelStyle = panel ? window.getComputedStyle(panel) : null
    const firstSectionStyle = sections[0]
      ? window.getComputedStyle(sections[0])
      : null
    return {
      bodyScrollWidth: document.body.scrollWidth,
      docScrollWidth: document.documentElement.scrollWidth,
      fieldCount: fields.length,
      fields,
      panelBackground: panelStyle?.backgroundColor || '',
      panelClientWidth: panel?.clientWidth || 0,
      panelRect: panelRect
        ? {
            height: Math.round(panelRect.height),
            left: Math.round(panelRect.left),
            right: Math.round(panelRect.right),
            width: Math.round(panelRect.width),
          }
        : null,
      panelScrollWidth: panel?.scrollWidth || 0,
      sectionBackground: firstSectionStyle?.backgroundColor || '',
      sectionCount: sections.length,
      viewportWidth: window.innerWidth,
    }
  })

  assert(metrics.panelRect, `${scenarioName} 缺少模型上下文弹窗`)
  assert.equal(metrics.sectionCount, 2, `${scenarioName} 阈值分组数量异常`)
  assert.equal(metrics.fieldCount, 6, `${scenarioName} 阈值字段数量异常`)
  assert(
    metrics.bodyScrollWidth <= metrics.viewportWidth + 2 &&
      metrics.docScrollWidth <= metrics.viewportWidth + 2,
    `${scenarioName} 页面出现横向溢出: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.panelRect.width <= metrics.viewportWidth - 32 + 2 &&
      metrics.panelScrollWidth <= metrics.panelClientWidth + 2,
    `${scenarioName} 弹窗面板横向溢出: ${JSON.stringify(metrics)}`
  )
  for (const field of metrics.fields) {
    assert(
      field.scrollWidth <= field.clientWidth + 2 &&
        field.inputWidth > 0 &&
        field.fillWidth > 0 &&
        field.inputLeft >= field.left - 1 &&
        field.inputRight <= field.right + 1 &&
        field.inputTop >= field.headBottom - 1 &&
        field.fillRight <= field.right + 1 &&
        field.fillBottom <= field.inputTop + 1 &&
        field.headScrollWidth <= field.headClientWidth + 2,
      `${scenarioName} 字段盒模型异常: ${JSON.stringify(field)}`
    )
  }
  assert(
    metrics.panelBackground && metrics.sectionBackground,
    `${scenarioName} 弹窗浅色/暗色背景未正常计算: ${JSON.stringify(metrics)}`
  )
}

async function assertSessionExpiredAlertModal(page, scenarioName) {
  const dialog = page.getByRole('dialog', { name: '登录状态已失效' })
  await dialog.waitFor({ state: 'visible' })
  await expectText(page, '登录已过期，请重新登录')
  await expectRole(page, 'button', '重新登录')

  const metrics = await dialog.evaluate((node) => {
    const rect = node.getBoundingClientRect()
    const title = node.querySelector('.admin-modal-title')
    const detail = node.querySelector('.admin-alert-message')
    const confirmButton = Array.from(node.querySelectorAll('button')).find(
      (button) => button.textContent.trim() === '重新登录'
    )
    const closeButton = node.querySelector('.admin-modal-close')
    const titleStyle = title ? window.getComputedStyle(title) : null
    const detailStyle = detail ? window.getComputedStyle(detail) : null
    const buttonStyle = confirmButton
      ? window.getComputedStyle(confirmButton)
      : null
    const panelStyle = window.getComputedStyle(node)
    return {
      bodyScrollWidth: document.body.scrollWidth,
      buttonBackground: buttonStyle?.backgroundColor || '',
      buttonBorderRadius: parseFloat(buttonStyle?.borderRadius || '0'),
      buttonHeight: Math.round(
        confirmButton?.getBoundingClientRect().height || 0
      ),
      closeButtonSize: Math.round(
        closeButton?.getBoundingClientRect().width || 0
      ),
      detailBackground: detailStyle?.backgroundColor || '',
      detailColor: detailStyle?.color || '',
      docScrollWidth: document.documentElement.scrollWidth,
      hasAdminPanelClass: node.classList.contains('admin-modal-panel'),
      hasAlertClass: node.classList.contains('admin-alert-modal'),
      panelBackground: panelStyle.backgroundColor,
      panelBorderRadius: parseFloat(panelStyle.borderRadius || '0'),
      panelHeight: Math.round(rect.height),
      panelWidth: Math.round(rect.width),
      theme: document.documentElement.dataset.adminTheme,
      titleColor: titleStyle?.color || '',
      viewportHeight: window.innerHeight,
      viewportWidth: window.innerWidth,
    }
  })

  assert(
    metrics.hasAdminPanelClass && metrics.hasAlertClass,
    `${scenarioName} 登录态弹窗未复用后台弹窗结构: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.panelWidth > 0 &&
      metrics.panelWidth <= Math.min(520, metrics.viewportWidth - 32) + 2,
    `${scenarioName} 登录态弹窗宽度异常: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.panelHeight > 0 && metrics.panelHeight <= metrics.viewportHeight,
    `${scenarioName} 登录态弹窗高度异常: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.bodyScrollWidth <= metrics.viewportWidth + 2 &&
      metrics.docScrollWidth <= metrics.viewportWidth + 2,
    `${scenarioName} 登录态弹窗导致横向溢出: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.panelBorderRadius <= 12 &&
      metrics.buttonBorderRadius <= 12 &&
      metrics.buttonHeight >= 36 &&
      metrics.closeButtonSize === 32,
    `${scenarioName} 登录态弹窗仍像旧通用弹窗而不是后台弹窗: ${JSON.stringify(metrics)}`
  )
  assert(
    metrics.panelBackground &&
      metrics.titleColor &&
      metrics.detailBackground &&
      metrics.detailColor &&
      metrics.buttonBackground,
    `${scenarioName} 登录态弹窗颜色未正常计算: ${JSON.stringify(metrics)}`
  )
  if (scenarioName.includes('dark')) {
    assert.equal(metrics.theme, 'dark', `${scenarioName} 未处于暗色主题`)
  }
}

function createFakeAdminToken() {
  const header = { alg: 'none', typ: 'JWT' }
  const payload = {
    exp: Math.floor(Date.now() / 1000) + 3600,
    role: 1,
    uid: 1,
    uname: 'admin',
  }
  return `${base64UrlJson(header)}.${base64UrlJson(payload)}.`
}

function createFakeUserToken() {
  const header = { alg: 'none', typ: 'JWT' }
  const payload = {
    exp: Math.floor(Date.now() / 1000) + 3600,
    role: 0,
    uid: 7,
    uname: 'org-user',
  }
  return `${base64UrlJson(header)}.${base64UrlJson(payload)}.`
}

function base64UrlJson(value) {
  return Buffer.from(JSON.stringify(value))
    .toString('base64url')
    .replace(/=+$/u, '')
}

async function installApiRpcMock(page) {
  const calls = []
  const state = {
    upstreamMode: 'codex_backend',
    upstreamStrategy: 'backend_only',
  }
  page.__styleL1ApiRpcCalls = calls

  await page.route('**/rpc/api', async (route) => {
    const request = route.request().postDataJSON()
    calls.push({
      method: request.method,
      params: request.params || {},
    })
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: request.id,
        jsonrpc: '2.0',
        result: {
          code: 0,
          data: getApiMockData(request.method, request.params || {}, state),
          message: 'OK',
        },
      }),
    })
  })
}

async function installApiRpcAuthExpiredMock(page) {
  await page.route('**/rpc/api', async (route) => {
    const request = route.request().postDataJSON()
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: request.id,
        jsonrpc: '2.0',
        result: {
          code: 10005,
          data: null,
          message: '登录已过期，请重新登录',
        },
      }),
    })
  })
}

async function installAuthConfigMock(page) {
  await page.route('**/auth/oauth/config', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: false, provider: '' }),
    })
  })
}

async function installCodexBalanceMock(page) {
  await page.route('**/public/codex/balance', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'ok',
        fetched_at: '2026-05-19T12:00:00Z',
        credits: {
          hasCredits: false,
          unlimited: false,
          balance: '0',
        },
        rate_limits: {
          limit_id: 'codex',
          limit_name: null,
          plan_type: 'prolite',
          credits: {
            hasCredits: false,
            unlimited: false,
            balance: '0',
          },
          primary: {
            used_percent: 16,
            remaining_percent: 84,
            window_duration_mins: 300,
            resets_at_time: '2026-05-19T17:00:00Z',
          },
          secondary: {
            used_percent: 8,
            remaining_percent: 92,
            window_duration_mins: 10080,
            resets_at_time: '2026-05-26T12:00:00Z',
          },
        },
        rate_limits_by_limit_id: {
          codex: {
            limit_id: 'codex',
            limit_name: null,
            plan_type: 'prolite',
            credits: {
              hasCredits: false,
              unlimited: false,
              balance: '0',
            },
            primary: {
              used_percent: 16,
              remaining_percent: 84,
              window_duration_mins: 300,
              resets_at_time: '2026-05-19T17:00:00Z',
            },
            secondary: {
              used_percent: 8,
              remaining_percent: 92,
              window_duration_mins: 10080,
              resets_at_time: '2026-05-26T12:00:00Z',
            },
          },
          'gpt-5.3-codex-spark': {
            limit_id: 'gpt-5.3-codex-spark',
            limit_name: 'GPT-5.3-Codex-Spark',
            plan_type: 'prolite',
            credits: {
              hasCredits: false,
              unlimited: false,
              balance: '0',
            },
            primary: {
              used_percent: 36,
              remaining_percent: 64,
              window_duration_mins: 300,
              resets_at_time: '2026-05-19T16:30:00Z',
            },
            secondary: {
              used_percent: 24,
              remaining_percent: 76,
              window_duration_mins: 10080,
              resets_at_time: '2026-05-26T12:00:00Z',
            },
          },
        },
      }),
    })
  })
}

function getApiMockData(method, params = {}, state = {}) {
  if (method === 'summary') {
    const now = Math.floor(Date.now() / 1000)
    const seconds = Number.isFinite(Number(params.start_time))
      ? Math.max(60, now - Number(params.start_time))
      : 24 * 60 * 60
    if (seconds <= 90) {
      return {
        summary: {
          average_duration_ms: 640,
          backend_requests: 3,
          cli_requests: 1,
          estimated_cost_usd: 0.0312,
          fallback_requests: 0,
          failed_requests: 0,
          input_tokens: 1860,
          output_tokens: 520,
          success_requests: 4,
          total_requests: 4,
          total_tokens: 2380,
        },
      }
    }
    return {
      summary: {
        average_duration_ms: 842,
        backend_requests: 206,
        cli_requests: 92,
        estimated_cost_usd: 1.4288,
        fallback_requests: 7,
        failed_requests: 11,
        input_tokens: 86410,
        output_tokens: 62522,
        success_requests: 287,
        total_requests: 298,
        total_tokens: 148932,
      },
    }
  }

  if (method === 'gateway_upstream_get') {
    return {
      default_strategy: 'backend_only',
      default_mode: 'codex_backend',
      fallback_enabled: false,
      mode: state.upstreamMode || 'codex_backend',
      strategy: state.upstreamStrategy || 'backend_only',
      options: [
        { label: 'Backend 直连', value: 'backend_only' },
        { label: 'Backend + CLI 兜底', value: 'backend_with_cli_fallback' },
        { label: '强制 CLI', value: 'codex_cli' },
      ],
    }
  }

  if (method === 'gateway_upstream_set') {
    const strategy = params.strategy || 'backend_only'
    state.upstreamStrategy = strategy
    state.upstreamMode =
      strategy === 'codex_cli' ? 'codex_cli' : 'codex_backend'
    return {
      default_strategy: 'backend_only',
      default_mode: 'codex_backend',
      fallback_enabled: strategy === 'backend_with_cli_fallback',
      mode: state.upstreamMode,
      strategy,
      options: [
        { label: 'Backend 直连', value: 'backend_only' },
        { label: 'Backend + CLI 兜底', value: 'backend_with_cli_fallback' },
        { label: '强制 CLI', value: 'codex_cli' },
      ],
    }
  }

  if (method === 'key_list') {
    const baseKeys = [
      {
        allowed_models: ['gpt-5.3-codex'],
        disabled: false,
        id: 1,
        key_last4: '8a2c',
        key_prefix: 'ogw_production',
        plain_key: 'ogw_productionapikey_8a2c',
        created_at: 1777900000,
        updated_at: 1777950000,
        last_used_at: 1778000000,
        name: 'productionapikey',
        upstream_strategy: 'backend_with_cli_fallback',
        quota_daily_billable_input_tokens: 450_000,
        quota_daily_input_tokens: 800_000,
        quota_daily_output_tokens: 200_000,
        quota_daily_tokens: 1_000_000,
        quota_weekly_billable_input_tokens: 2_250_000,
        quota_weekly_input_tokens: 4_000_000,
        quota_weekly_output_tokens: 1_000_000,
        quota_weekly_tokens: 5_000_000,
      },
      {
        allowed_models: [],
        disabled: true,
        id: 2,
        key_last4: '3f9d',
        key_prefix: 'ogw_stagingke',
        plain_key: 'ogw_stagingkeylongnameforoverflowcheck_3f9d',
        created_at: 1777800000,
        updated_at: 1777850000,
        last_used_at: 0,
        name: 'stagingkeylongnameforoverflowcheck',
        upstream_strategy: '',
        quota_daily_billable_input_tokens: 0,
        quota_daily_input_tokens: 0,
        quota_daily_output_tokens: 0,
        quota_daily_tokens: 0,
        quota_weekly_billable_input_tokens: 0,
        quota_weekly_input_tokens: 0,
        quota_weekly_output_tokens: 0,
        quota_weekly_tokens: 0,
      },
    ]
    const extraKeys = Array.from({ length: 8 }, (_, index) => {
      const id = index + 3
      return {
        allowed_models: ['gpt-5.3-codex'],
        disabled: index % 3 === 0,
        id,
        key_last4: `x${id}z${index}`,
        key_prefix: `ogw_extra${id}`,
        plain_key: `ogw_extraapikey${id}_x${id}z${index}`,
        created_at: 1777700000 - index * 1_000,
        updated_at: 1777750000 - index * 1_000,
        last_used_at: 1777990000 - index * 100,
        name: `extraapikey${id}`,
        upstream_strategy: index % 2 === 0 ? 'backend_only' : 'codex_cli',
        quota_daily_billable_input_tokens: 0,
        quota_daily_input_tokens: 0,
        quota_daily_output_tokens: 0,
        quota_daily_tokens: id * 1_000_000,
        quota_weekly_billable_input_tokens: 0,
        quota_weekly_input_tokens: 0,
        quota_weekly_output_tokens: 0,
        quota_weekly_tokens: (id + 1) * 1_000_000,
      }
    })
    return {
      items: [...baseKeys, ...extraKeys],
      total: baseKeys.length + extraKeys.length,
    }
  }

  if (method === 'key_reset_secret') {
    const id = Number(params.key_id || 0)
    const name =
      id === 1
        ? 'productionapikey'
        : id === 2
          ? 'stagingkeylongnameforoverflowcheck'
          : `extraapikey${id}`
    return {
      allowed_models: ['gpt-5.3-codex'],
      disabled: false,
      id,
      key_last4: `r${id}st`,
      key_prefix: `ogw_${name}`.slice(0, 12),
      plain_key: `ogw_${name}_reset_${id}_r${id}st`,
      created_at: 1777900000,
      updated_at: 1778050000,
      name,
      upstream_strategy: id === 1 ? 'backend_with_cli_fallback' : '',
      quota_daily_tokens: 0,
      quota_weekly_tokens: 0,
    }
  }

  if (method === 'model_list') {
    const contextByModel = {
      'gpt-5.5': [400_000, 260_000, 380_000, 1_040_000, 1_900_000, 8],
      'gpt-5.4': [400_000, 260_000, 380_000, 1_040_000, 1_900_000, 8],
      'gpt-5.4-mini': [400_000, 260_000, 380_000, 1_040_000, 1_900_000, 8],
      'gpt-5.3-codex': [400_000, 260_000, 380_000, 1_040_000, 1_900_000, 8],
      'gpt-5.3-codex-spark': [
        400_000, 260_000, 380_000, 1_040_000, 1_900_000, 8,
      ],
      'gpt-5.2': [400_000, 260_000, 380_000, 1_040_000, 1_900_000, 8],
    }
    const baseModels = [
      'gpt-5.5',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.3-codex',
      'gpt-5.3-codex-spark',
      'gpt-5.2',
      'gpt-5.5-pro',
    ].map((modelID, index) => ({
      enabled: true,
      id: index + 1,
      model_id: modelID,
      owned_by: 'openai',
      source: modelID === 'gpt-5.5-pro' ? 'stale' : 'seed',
      effective_context_window_tokens: contextByModel[modelID]?.[0] || 0,
      effective_context_compact_tokens: contextByModel[modelID]?.[1] || 0,
      effective_context_hard_tokens: contextByModel[modelID]?.[2] || 0,
      effective_context_compact_bytes: contextByModel[modelID]?.[3] || 0,
      effective_context_hard_bytes: contextByModel[modelID]?.[4] || 0,
      effective_context_keep_items: contextByModel[modelID]?.[5] || 0,
    }))
    const extraModels = []
    return {
      items: [...baseModels, ...extraModels],
      total: baseModels.length + extraModels.length,
    }
  }

  if (method === 'model_context_update') {
    return {
      id: params.id,
      model_id: 'gpt-5.5',
      context_window_tokens: params.context_window_tokens || 0,
      context_compact_tokens: params.context_compact_tokens || 0,
      context_hard_tokens: params.context_hard_tokens || 0,
      context_compact_bytes: params.context_compact_bytes || 0,
      context_hard_bytes: params.context_hard_bytes || 0,
      context_keep_items: params.context_keep_items || 0,
    }
  }

  if (method === 'official_model_price_list') {
    return {
      items: [
        {
          cached_input_usd_per_million: 0.5,
          input_usd_per_million: 5,
          model_id: 'gpt-5.5',
          output_usd_per_million: 30,
        },
        {
          cached_input_usd_per_million: 0.25,
          input_usd_per_million: 2.5,
          model_id: 'gpt-5.4',
          output_usd_per_million: 15,
        },
        {
          cached_input_usd_per_million: 0.075,
          input_usd_per_million: 0.75,
          model_id: 'gpt-5.4-mini',
          output_usd_per_million: 4.5,
        },
        {
          cached_input_usd_per_million: 30,
          input_usd_per_million: 30,
          model_id: 'gpt-5.5-pro',
          output_usd_per_million: 180,
        },
        {
          cached_input_usd_per_million: 0.175,
          input_usd_per_million: 1.75,
          model_id: 'gpt-5.3-codex',
          output_usd_per_million: 14,
        },
        {
          model_id: 'gpt-5.3-codex-spark',
          price_note: 'research preview，价格未定',
        },
        {
          cached_input_usd_per_million: 0.175,
          input_usd_per_million: 1.75,
          model_id: 'gpt-5.2',
          output_usd_per_million: 14,
        },
      ],
    }
  }

  if (method === 'usage_list') {
    const model = params.model || 'gpt-5.3-codex'
    const effort = params.reasoning_effort || 'high'
    return {
      items: [
        {
          api_key_name: 'productionapikey',
          api_key_prefix: 'ogw_production',
          cached_tokens: model === 'gpt-5.4' ? 272640 : 1200,
          created_at: 1778000000,
          duration_ms: 813,
          endpoint: '/v1/responses',
          error_type: '',
          estimated_cost_usd: 0.1212,
          id: 1,
          input_tokens: model === 'gpt-5.4' ? 272889 : 1900,
          model,
          output_tokens: model === 'gpt-5.4' ? 50 : 2310,
          reasoning_effort: effort,
          reasoning_tokens: model === 'gpt-5.4' ? 0 : 320,
          request_bytes: 4096,
          request_id: 'req_style_l1_prod_1',
          response_bytes: 8192,
          session_id: 'session-style-l1',
          status_code: 200,
          success: true,
          total_tokens: 4210,
          diagnostic: {
            context_compacted: true,
            context_compaction_count: 2,
            context_compaction_reason: 'request_preflight',
            context_compaction_summary:
              '自动压缩摘要：保留 /srv/app/server.go、context_length_exceeded 和 pnpm style:l1 线索。',
            context_original_bytes: 980000,
            context_compacted_bytes: 210000,
            context_original_estimated_tokens: 240000,
            context_compacted_estimated_tokens: 52000,
            request_bytes: 4096,
            response_bytes: 8192,
          },
          diagnostic_summary:
            'request=4096B, response=8192B, compact_count=2, compact_reason=request_preflight',
          upstream_configured_mode: 'codex_backend',
          upstream_error_type: '',
          upstream_fallback: false,
          upstream_mode: 'codex_backend',
        },
        {
          api_key_name: 'productionapikey',
          api_key_prefix: 'ogw_production',
          cached_tokens: model === 'gpt-5.4' ? 272512 : 40800,
          created_at: 1777999000,
          duration_ms: 1240,
          endpoint: '/v1/chat/completions',
          error_type: '',
          estimated_cost_usd: 0.7421,
          id: 2,
          input_tokens: model === 'gpt-5.4' ? 272660 : 60000,
          model,
          output_tokens: model === 'gpt-5.4' ? 179 : 1200,
          reasoning_effort: params.reasoning_effort || 'low',
          reasoning_tokens: 0,
          request_bytes: 65536,
          request_id: 'req_style_l1_prod_2',
          response_bytes: 12000,
          session_id: 'session-style-l1',
          status_code: 200,
          success: true,
          total_tokens: 61200,
          upstream_configured_mode: 'codex_backend',
          upstream_error_type: '',
          upstream_fallback: true,
          upstream_mode: 'codex_cli',
        },
        {
          api_key_name: 'stagingkeylongnameforoverflowcheck',
          api_key_prefix: 'ogw_stagingke',
          cached_tokens: 0,
          created_at: 1777998000,
          duration_ms: 330,
          endpoint: '/v1/responses',
          error_type: 'upstream_error',
          estimated_cost_usd: null,
          id: 3,
          input_tokens: 1000,
          model: params.success === false ? 'gpt-5.2' : model,
          output_tokens: 80,
          reasoning_effort: params.reasoning_effort || 'xhigh',
          reasoning_tokens: 0,
          request_bytes: 2048,
          request_id: 'req_style_l1_stage_1',
          response_bytes: 512,
          session_id: 'session-style-l1-error',
          status_code: 502,
          success: false,
          total_tokens: 1080,
          upstream_configured_mode: 'codex_cli',
          upstream_error_type: 'codex_cli_upstream_failed',
          diagnostic: {
            backend_only: true,
            fallback_blocked: true,
            request_bytes: 2048,
            response_bytes: 512,
            upstream_body: 'codex cli exited with status 1',
          },
          diagnostic_summary:
            'request=2048B, response=512B, backend-only, fallback-blocked',
          upstream_fallback: false,
          upstream_mode: 'codex_cli',
        },
      ],
      total: 12,
    }
  }

  if (method === 'usage_key_summaries') {
    return {
      items: [
        {
          api_key_id: 1,
          api_key_name: 'productionapikey',
          api_key_prefix: 'ogw_production',
          average_duration_ms: 980,
          cached_tokens: 42000,
          disabled: false,
          estimated_cost_usd: 0.96,
          failed_requests: 0,
          backend_requests: 1,
          cli_requests: 1,
          fallback_requests: 1,
          input_tokens: 61900,
          output_tokens: 3510,
          success_requests: 2,
          total_requests: 2,
          total_tokens: 65410,
        },
        {
          api_key_id: 2,
          api_key_name: 'stagingkeylongnameforoverflowcheck',
          api_key_prefix: 'ogw_stagingke',
          average_duration_ms: 330,
          cached_tokens: 0,
          disabled: true,
          estimated_cost_usd: null,
          failed_requests: 1,
          backend_requests: 0,
          cli_requests: 1,
          fallback_requests: 0,
          input_tokens: 1000,
          output_tokens: 80,
          success_requests: 0,
          total_requests: 1,
          total_tokens: 1080,
        },
      ],
    }
  }

  if (method === 'usage_session_summaries') {
    return {
      items: [
        {
          api_key_id: 1,
          api_key_name: 'productionapikey',
          api_key_prefix: 'ogw_production',
          average_duration_ms: 1026,
          cached_tokens: 42000,
          estimated_cost_usd: 0.86,
          failed_requests: 0,
          backend_requests: 1,
          cli_requests: 1,
          fallback_requests: 1,
          first_seen_at: 1777999000,
          input_tokens: 61900,
          last_seen_at: 1778000000,
          output_tokens: 3510,
          reasoning_tokens: 320,
          session_id: 'session-style-l1',
          success_requests: 2,
          total_requests: 2,
          total_tokens: 65410,
          context_compaction_count: 2,
          context_summary:
            '自动压缩摘要：保留 /srv/app/server.go、context_length_exceeded 和 pnpm style:l1 线索。',
          context_original_bytes: 980000,
          context_compacted_bytes: 210000,
          context_original_tokens: 240000,
          context_compacted_tokens: 52000,
          context_compacted_at: 1778000000,
        },
        {
          api_key_id: 2,
          api_key_name: 'stagingkeylongnameforoverflowcheck',
          api_key_prefix: 'ogw_stagingke',
          average_duration_ms: 330,
          cached_tokens: 0,
          estimated_cost_usd: null,
          failed_requests: 1,
          backend_requests: 0,
          cli_requests: 1,
          fallback_requests: 0,
          first_seen_at: 1777998000,
          input_tokens: 1000,
          last_seen_at: 1777998000,
          output_tokens: 80,
          reasoning_tokens: 0,
          session_id: 'session-style-l1-error',
          success_requests: 0,
          total_requests: 1,
          total_tokens: 1080,
        },
      ],
      total: 2,
    }
  }

  if (method === 'user_key_list') {
    return {
      items: [
        {
          allowed_models: ['gpt-5.3-codex'],
          disabled: false,
          id: 7,
          key_last4: '7abc',
          key_prefix: 'ogw_myteam',
          last_used_at: 1778000000,
          name: 'myteamkey',
          owner_user_id: 7,
        },
      ],
      total: 1,
    }
  }

  if (method === 'user_usage_summary') {
    return {
      summary: {
        estimated_cost_usd: 1.24,
        input_tokens: 12000,
        output_tokens: 2100,
        total_requests: 18,
        total_tokens: 14100,
      },
    }
  }

  if (method === 'user_usage_list') {
    return {
      items: [
        {
          api_key_prefix: 'ogw_myteam',
          created_at: 1778000000,
          endpoint: 'responses',
          estimated_cost_usd: 0.12,
          id: 11,
          model: 'gpt-5.3-codex',
          reasoning_effort: 'medium',
          status_code: 200,
          total_tokens: 1600,
        },
      ],
      total: 1,
    }
  }

  if (method === 'usage_buckets') {
    return { items: createMockUsageBuckets() }
  }

  return {}
}

function createMockUsageBuckets() {
  const now = new Date()
  const todayStart = new Date(now.getFullYear(), now.getMonth(), now.getDate())

  return Array.from({ length: 12 }, (_, index) => {
    const d = new Date(todayStart)
    d.setDate(todayStart.getDate() - (11 - index))
    const calls = index % 5 === 0 ? 4 : 18 + index * 3
    const inputTokens = calls * (1200 + index * 80)
    const cachedTokens = Math.round(inputTokens * 0.72)
    const outputTokens = calls * (220 + index * 16)
    const reasoningTokens = calls * (40 + index * 3)
    const model = index % 5 === 0 ? 'gpt-5.4-mini' : 'gpt-5.4'

    return {
      bucket_start: Math.floor(d.getTime() / 1000),
      average_duration_ms: 420 + index * 24,
      backend_requests: Math.max(0, calls - Math.ceil(calls / 3)),
      cached_tokens: cachedTokens,
      cli_requests: Math.ceil(calls / 3),
      estimated_cost_usd: Number((calls * (0.018 + index * 0.001)).toFixed(4)),
      failed_requests: index % 4 === 0 ? 2 : 0,
      fallback_requests: index % 5 === 0 ? 1 : 0,
      input_tokens: inputTokens,
      model,
      output_tokens: outputTokens,
      reasoning_tokens: reasoningTokens,
      success_requests: calls - (index % 4 === 0 ? 2 : 0),
      total_requests: calls,
      total_tokens: inputTokens + outputTokens,
    }
  })
}

function tailLogs(text) {
  return text.trim().split('\n').slice(-20).join('\n')
}

await main()
