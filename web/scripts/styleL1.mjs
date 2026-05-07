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

let devServerProcess = null
let devServerLogs = ''

const scenarios = [
  {
    name: 'home-desktop',
    path: '/',
    viewport: { width: 1440, height: 900 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'OAuth API 管理后台')
      await expectRole(page, 'button', '登录')
      await expectText(page, 'OpenAI OAuth API Service')
    },
  },
  {
    name: 'home-mobile',
    path: '/',
    viewport: { width: 390, height: 844 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'OAuth API 管理后台')
      await expectRole(page, 'button', '登录')
      await expectText(page, 'OpenAI OAuth API Service')
    },
  },
  {
    name: 'login-desktop',
    path: '/login',
    viewport: { width: 1280, height: 800 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'OAuth API 管理后台')
      await expectRole(page, 'button', '登录')
      await expectText(page, 'OpenAI OAuth API Service')
    },
  },
  {
    name: 'register-mobile',
    path: '/register',
    viewport: { width: 390, height: 844 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'OAuth API 管理后台')
      await expectRole(page, 'button', '登录')
    },
  },
  {
    name: 'oauth-login-redirect',
    path: '/oauth-login',
    viewport: { width: 1280, height: 800 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'OAuth API 管理后台')
      await expectRole(page, 'button', '登录')
    },
  },
  {
    name: 'admin-login-mobile',
    path: '/admin-login',
    viewport: { width: 390, height: 844 },
    verify: async (page) => {
      await expectText(page, 'OAuth API 管理后台')
      await expectRole(page, 'button', '登录')
      await expectText(page, 'OpenAI OAuth API Service')
    },
  },
  {
    name: 'admin-login-desktop',
    path: '/admin-login',
    viewport: { width: 1440, height: 900 },
    verify: async (page) => {
      await expectText(page, 'OAuth API 管理后台')
      await expectRole(page, 'button', '登录')
      await expectText(page, 'OpenAI OAuth API Service')
    },
  },
  {
    name: 'admin-menu-redirect',
    path: '/admin-menu',
    viewport: { width: 1280, height: 800 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'OAuth API 管理后台')
      await expectRole(page, 'button', '登录')
    },
  },
  {
    name: 'portal-redirect',
    path: '/portal',
    viewport: { width: 390, height: 844 },
    expectPath: '/admin-login',
    verify: async (page) => {
      await expectText(page, 'OAuth API 管理后台')
      await expectRole(page, 'button', '登录')
    },
  },
  {
    name: 'admin-dashboard-desktop',
    path: '/admin-dashboard',
    viewport: { width: 1440, height: 900 },
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'OAuth API 管理后台')
      await expectText(page, '业务看板')
      await expectText(page, '30 天 usage 趋势')
      await expectText(page, 'Token 构成')
      await expectText(page, '30 天按天统计')
      await expectText(page, '最近 usage')
      await assertAdminChrome(page, 'admin-dashboard-desktop')
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
      await expectText(page, 'OAuth API 管理后台')
      await expectText(page, '业务看板')
      await expectText(page, '30 天 usage 趋势')
      await expectText(page, 'Token 构成')
      await assertAdminChrome(page, 'admin-dashboard-mobile')
      await assertApiVisuals(page, 'admin-dashboard-mobile')
    },
  },
  {
    name: 'admin-oauth-desktop',
    path: '/admin-oauth',
    viewport: { width: 1280, height: 800 },
    adminAuth: true,
    mockOAuthConfig: true,
    verify: async (page) => {
      await expectText(page, 'OAuth/SSO 登录配置')
      await expectText(page, '管理员登录入口')
      await expectText(page, '/auth/oauth/start?scope=admin&redirect=/admin-dashboard')
      await assertAdminChrome(page, 'admin-oauth-desktop')
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

    const browser = await chromium.launch({ headless })
    try {
      for (const scenario of scenarios) {
        await runScenario(browser, scenario)
      }
    } finally {
      await browser.close()
    }

    console.log(`[style:l1] 通过，共验证 ${scenarios.length} 个场景`)
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

    if (scenario.userAuth) {
      await page.addInitScript((token) => {
        window.localStorage.setItem('user_access_token', token)
      }, createFakeUserToken())
    }

    if (scenario.mockApiRpc) {
      await installApiRpcMock(page)
    }

    if (scenario.mockOAuthConfig) {
      await installOAuthConfigMock(page)
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
  const locator = page.getByRole(role, { name })
  await locator.waitFor({ state: 'visible', timeout: 10_000 })
}

async function expectText(page, text) {
  const locator = page.getByText(text, { exact: false })
  await locator.first().waitFor({ state: 'visible', timeout: 10_000 })
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
      hasGlobalCustomerFilter: document.body.innerText.includes('全局客户'),
      hasGlobalSalesFilter: document.body.innerText.includes('全局业务员'),
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
  assert(metrics.main.width > 0, `${scenarioName} 内容区宽度异常`)
  assert(metrics.main.height > 0, `${scenarioName} 内容区高度异常`)

  if (metrics.viewportWidth >= 1024) {
    assert(
      metrics.aside.right <= metrics.header.left + 1,
      `${scenarioName} 桌面侧边栏和顶部栏发生重叠: ${JSON.stringify(metrics)}`
    )
  } else {
    assert(
      metrics.aside.bottom <= metrics.header.top + 1,
      `${scenarioName} 移动端侧边栏和顶部栏发生重叠: ${JSON.stringify(metrics)}`
    )
  }
}

async function assertApiVisuals(page, scenarioName) {
  const metrics = await page.evaluate(() => {
    const headings = Array.from(document.querySelectorAll('h2')).map((node) =>
      node.textContent.trim()
    )
    const main = document.querySelector('main')
    const barTitles = Array.from(main?.querySelectorAll('[title]') || [])
      .map((node) => node.getAttribute('title') || '')
      .filter((title) => title.includes('tokens'))

    return {
      barCount: barTitles.length,
      hasDailyTable: document.body.innerText.includes('cached_input_tokens'),
      hasEndpointPanel: headings.includes('Endpoint 分布'),
      hasModelPanel: headings.includes('模型用量分布'),
      hasTokenPanel: headings.includes('Token 构成'),
    }
  })

  assert(metrics.hasTokenPanel, `${scenarioName} 缺少 token 构成面板`)
  assert(metrics.hasModelPanel, `${scenarioName} 缺少模型用量分布面板`)
  assert(metrics.hasEndpointPanel, `${scenarioName} 缺少 endpoint 分布面板`)
  assert(metrics.hasDailyTable, `${scenarioName} 缺少按天 token 统计表`)
  assert(
    metrics.barCount >= 20,
    `${scenarioName} usage 趋势柱状图数量异常: ${JSON.stringify(metrics)}`
  )
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
  await page.route('**/rpc/api', async (route) => {
    const request = route.request().postDataJSON()
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: request.id,
        jsonrpc: '2.0',
        result: {
          code: 0,
          data: getApiMockData(request.method),
          message: 'OK',
        },
      }),
    })
  })
}

async function installOAuthConfigMock(page) {
  await page.route('**/auth/oauth/config', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        enabled: true,
        login_url: '/auth/oauth/start',
        provider_name: 'OpenAI',
      }),
    })
  })
}

function getApiMockData(method) {
  if (method === 'summary') {
    return {
      summary: {
        average_duration_ms: 842,
        failed_requests: 11,
        input_tokens: 86410,
        output_tokens: 62522,
        success_requests: 287,
        total_requests: 298,
        total_tokens: 148932,
      },
    }
  }

  if (method === 'key_list') {
    return {
      items: [
        {
          allowed_models: ['gpt-5.4', 'gpt-5.4-mini'],
          disabled: false,
          id: 1,
          key_last4: '8a2c',
          key_prefix: 'sk-api-prod',
          last_used_at: 1778000000,
          name: 'production-api-key',
        },
        {
          allowed_models: [],
          disabled: true,
          id: 2,
          key_last4: '3f9d',
          key_prefix: 'sk-api-stage',
          last_used_at: 0,
          name: 'staging-key-with-long-name-for-overflow-check',
        },
      ],
    }
  }

  if (method === 'model_list') {
    return {
      items: [
        {
          enabled: true,
          id: 1,
          model_id: 'gpt-5.4',
          owned_by: 'openai',
          source: 'manual',
        },
        {
          enabled: false,
          id: 2,
          model_id: 'gpt-5.4-mini',
          owned_by: 'openai',
          source: 'manual',
        },
      ],
    }
  }

  if (method === 'usage_list') {
    return {
      items: [
        {
          api_key_prefix: 'sk-api-prod',
          created_at: 1778000000,
          duration_ms: 813,
          endpoint: '/v1/responses',
          error_type: '',
          id: 1,
          input_tokens: 1900,
          model: 'gpt-5.4',
          output_tokens: 2310,
          status_code: 200,
          success: true,
          total_tokens: 4210,
        },
        {
          api_key_prefix: 'sk-api-prod',
          created_at: 1777999000,
          duration_ms: 1240,
          endpoint: '/v1/chat/completions',
          error_type: '',
          id: 2,
          input_tokens: 60000,
          model: 'gpt-5.4',
          output_tokens: 1200,
          status_code: 200,
          success: true,
          total_tokens: 61200,
        },
        {
          api_key_prefix: 'sk-api-stage',
          created_at: 1777998000,
          duration_ms: 330,
          endpoint: '/v1/responses',
          error_type: 'upstream_error',
          id: 3,
          input_tokens: 1000,
          model: 'gpt-5.4-mini',
          output_tokens: 80,
          status_code: 502,
          success: false,
          total_tokens: 1080,
        },
      ],
      total: 3,
    }
  }

  if (method === 'user_key_list') {
    return {
      items: [
        {
          allowed_models: ['gpt-5.4'],
          disabled: false,
          id: 7,
          key_last4: '7abc',
          key_prefix: 'sk-api-user',
          last_used_at: 1778000000,
          name: 'my-team-key',
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
          api_key_prefix: 'sk-api-user',
          created_at: 1778000000,
          endpoint: 'responses',
          estimated_cost_usd: 0.12,
          id: 11,
          model: 'gpt-5.4',
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

    return {
      bucket_start: Math.floor(d.getTime() / 1000),
      cached_tokens: cachedTokens,
      failed_requests: index % 4 === 0 ? 2 : 0,
      input_tokens: inputTokens,
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
