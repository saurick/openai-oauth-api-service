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
    name: 'legacy-oauth-login-redirect',
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
    name: 'admin-usage-desktop',
    path: '/admin-usage',
    viewport: { width: 1440, height: 900 },
    adminAuth: true,
    mockApiRpc: true,
    verify: async (page) => {
      await expectText(page, 'OAuth API 管理后台')
      await expectText(page, '调用明细')
      await expectText(page, '最近调用')
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
      await expectText(page, 'OAuth API 管理后台')
      await expectText(page, '调用明细')
      await expectText(page, '最近调用')
      await expectNoText(page, '返回控制台')
      await assertAdminChrome(page, 'admin-usage-mobile')
      await assertUsageTableVisuals(page, 'admin-usage-mobile')
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
      await expectRole(page, 'button', '生成 API 凭据')
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
      await expectRole(page, 'button', '生成 API 凭据')
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
      await expectText(page, '保存模型')
      await assertAdminChrome(page, 'admin-models-desktop')
      await assertModelTableVisuals(page, 'admin-models-desktop')
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
      await expectText(page, '保存模型')
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
      await expectText(page, 'OAuth API 管理后台')
      await expectText(page, '业务看板')
      await expectText(page, '30 天调用趋势')
      await expectText(page, 'Token 构成')
      await expectText(page, '调用状态概览')
      await expectText(page, '启用 API 凭据')
      await expectText(page, '30 天按天统计')
      await expectText(page, '最近调用')
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
      await expectText(page, '30 天调用趋势')
      await expectText(page, 'Token 构成')
      await expectText(page, '调用状态概览')
      await expectText(page, '启用 API 凭据')
      await assertAdminChrome(page, 'admin-dashboard-mobile')
      await assertApiVisuals(page, 'admin-dashboard-mobile')
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
      await expectText(page, 'OAuth API 管理后台')
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
      await expectText(page, 'OAuth API 管理后台')
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
      await expectText(page, 'OAuth API 管理后台')
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
      hasAccountNav: document.body.innerText.includes('账号目录'),
      hasGuideNav: document.body.innerText.includes('功能路线'),
      hasGlobalCustomerFilter: document.body.innerText.includes('全局客户'),
      hasGlobalSalesFilter: document.body.innerText.includes('全局业务员'),
      hasOAuthNav: document.body.innerText.includes('授权登录'),
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
      .filter((title) => title.includes('Token'))

    return {
      barCount: barTitles.length,
      hasDailyTable: document.body.innerText.includes('缓存输入'),
      hasEndpointPanel: headings.includes('接口分布'),
      hasKeyUsagePanel: headings.includes('24h 凭据消耗'),
      hasModelPanel: headings.includes('模型用量分布'),
      hasTokenPanel: headings.includes('Token 构成'),
    }
  })

  assert(metrics.hasTokenPanel, `${scenarioName} 缺少 token 构成面板`)
  assert(metrics.hasKeyUsagePanel, `${scenarioName} 缺少 key 消耗面板`)
  assert(metrics.hasModelPanel, `${scenarioName} 缺少模型用量分布面板`)
  assert(metrics.hasEndpointPanel, `${scenarioName} 缺少接口分布面板`)
  assert(metrics.hasDailyTable, `${scenarioName} 缺少按天 token 统计表`)
  assert(
    metrics.barCount >= 20,
    `${scenarioName} usage 趋势柱状图数量异常: ${JSON.stringify(metrics)}`
  )
}

async function assertUsageTableVisuals(page, scenarioName) {
  const metrics = await page.evaluate(() => {
    const main = document.querySelector('main')
    const table = main?.querySelector('table')
    const tableRect = table?.getBoundingClientRect()
    const mainRect = main?.getBoundingClientRect()

    return {
      hasSidebarUsageNav: document.body.innerText.includes('调用明细'),
      mainHeight: mainRect?.height || 0,
      tableHeight: tableRect?.height || 0,
      tableWidth: tableRect?.width || 0,
    }
  })

  assert(metrics.hasSidebarUsageNav, `${scenarioName} 缺少后台侧栏 usage 入口`)
  assert(metrics.mainHeight > 0, `${scenarioName} 后台内容区高度异常`)
  assert(metrics.tableHeight > 0, `${scenarioName} usage 表格高度异常`)
  assert(metrics.tableWidth > 0, `${scenarioName} usage 表格宽度异常`)
}

async function assertKeyTableVisuals(page, scenarioName) {
  const metrics = await page.evaluate(() => {
    const main = document.querySelector('main')
    const table = main?.querySelector('table')
    const tableRect = table?.getBoundingClientRect()
    const mainRect = main?.getBoundingClientRect()
    const createButton = Array.from(
      main?.querySelectorAll('button') || []
    ).find((node) => node.textContent.trim() === '生成 API 凭据')

    return {
      createButtonDisabled: Boolean(createButton?.disabled),
      hasFullPlainKey: document.body.innerText.includes('ogw_mock_prod_8a2c'),
      hasOptionalRemarkInput: Boolean(
        main?.querySelector('input[placeholder="例如内部测试 key"]')
      ),
      hasRemarkHeader: document.body.innerText.includes('备注'),
      hasSidebarKeyNav: document.body.innerText.includes('API 凭据'),
      mainHeight: mainRect?.height || 0,
      tableHeight: tableRect?.height || 0,
      tableWidth: tableRect?.width || 0,
    }
  })

  assert(metrics.hasSidebarKeyNav, `${scenarioName} 缺少后台侧栏 API 凭据入口`)
  assert(metrics.hasFullPlainKey, `${scenarioName} 缺少完整 key 展示`)
  assert(metrics.hasOptionalRemarkInput, `${scenarioName} 缺少可选备注输入框`)
  assert(metrics.hasRemarkHeader, `${scenarioName} 缺少备注列表列`)
  assert(
    !metrics.createButtonDisabled,
    `${scenarioName} 生成 API 凭据按钮不应默认禁用`
  )
  assert(metrics.mainHeight > 0, `${scenarioName} 后台内容区高度异常`)
  assert(metrics.tableHeight > 0, `${scenarioName} key 表格高度异常`)
  assert(metrics.tableWidth > 0, `${scenarioName} key 表格宽度异常`)
}

async function assertModelTableVisuals(page, scenarioName) {
  const metrics = await page.evaluate(() => {
    const main = document.querySelector('main')
    const table = main?.querySelector('table')
    const tableRect = table?.getBoundingClientRect()
    const mainRect = main?.getBoundingClientRect()

    return {
      formCount: main?.querySelectorAll('form').length || 0,
      hasDisableButton: Array.from(main?.querySelectorAll('button') || []).some(
        (node) => node.textContent.trim() === '禁用'
      ),
      hasDeleteButton: Array.from(main?.querySelectorAll('button') || []).some(
        (node) => node.textContent.trim() === '删除'
      ),
      hasModelCreateInput: Boolean(
        main?.querySelector('input[placeholder="例如 gpt-5.5"]')
      ),
      hasModel54: document.body.innerText.includes('gpt-5.4'),
      hasModel55: document.body.innerText.includes('gpt-5.5'),
      hasSidebarModelNav: document.body.innerText.includes('模型管理'),
      mainHeight: mainRect?.height || 0,
      tableHeight: tableRect?.height || 0,
      tableWidth: tableRect?.width || 0,
    }
  })

  assert.equal(metrics.formCount, 1, `${scenarioName} 应保留一个模型新增表单`)
  assert(metrics.hasSidebarModelNav, `${scenarioName} 缺少后台侧栏模型入口`)
  assert(metrics.hasModelCreateInput, `${scenarioName} 缺少模型新增输入框`)
  assert(metrics.hasModel54, `${scenarioName} 缺少 gpt-5.4 展示`)
  assert(metrics.hasModel55, `${scenarioName} 缺少 gpt-5.5 展示`)
  assert(metrics.hasDisableButton, `${scenarioName} 缺少模型启停操作`)
  assert(metrics.hasDeleteButton, `${scenarioName} 缺少模型删除操作`)
  assert(metrics.mainHeight > 0, `${scenarioName} 后台内容区高度异常`)
  assert(metrics.tableHeight > 0, `${scenarioName} 模型表格高度异常`)
  assert(metrics.tableWidth > 0, `${scenarioName} 模型表格宽度异常`)
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
          allowed_models: ['gpt-5.4', 'gpt-5.5'],
          disabled: false,
          id: 1,
          key_last4: '8a2c',
          key_prefix: 'sk-api-prod',
          last_used_at: 1778000000,
          name: 'production-api-key',
          plain_key: 'ogw_mock_prod_8a2c',
        },
        {
          allowed_models: [],
          disabled: true,
          id: 2,
          key_last4: '3f9d',
          key_prefix: 'sk-api-stage',
          last_used_at: 0,
          name: 'staging-key-with-long-name-for-overflow-check',
          plain_key: 'ogw_mock_stage_3f9d',
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
          model_id: 'gpt-5.5',
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
          model: 'gpt-5.5',
          output_tokens: 80,
          status_code: 502,
          success: false,
          total_tokens: 1080,
        },
      ],
      total: 3,
    }
  }

  if (method === 'usage_key_summaries') {
    return {
      items: [
        {
          api_key_id: 1,
          api_key_name: 'production-api-key',
          api_key_prefix: 'sk-api-prod',
          average_duration_ms: 980,
          cached_tokens: 42000,
          disabled: false,
          estimated_cost_usd: 0.96,
          failed_requests: 0,
          input_tokens: 61900,
          output_tokens: 3510,
          success_requests: 2,
          total_requests: 2,
          total_tokens: 65410,
        },
        {
          api_key_id: 2,
          api_key_name: 'staging-key-with-long-name-for-overflow-check',
          api_key_prefix: 'sk-api-stage',
          average_duration_ms: 330,
          cached_tokens: 0,
          disabled: true,
          estimated_cost_usd: null,
          failed_requests: 1,
          input_tokens: 1000,
          output_tokens: 80,
          success_requests: 0,
          total_requests: 1,
          total_tokens: 1080,
        },
      ],
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
