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

    await page.addInitScript(() => {
      if (!window.localStorage.getItem('admin_theme')) {
        window.localStorage.setItem('admin_theme', 'system')
      }
    })

    if (scenario.userAuth) {
      await page.addInitScript((token) => {
        window.localStorage.setItem('user_access_token', token)
      }, createFakeUserToken())
    }

    await installAuthConfigMock(page)

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
      .filter((title) => /请求|错误|费用|延迟|Token/u.test(title))
    const table = main?.querySelector('table')
    const tableRect = table?.getBoundingClientRect()
    const tableText = table?.innerText || ''

    return {
      barCount: barTitles.length,
      hasCoreCards: [
        '今日消费',
        '今日请求',
        '错误率',
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
        tableText.includes('production-api-key') &&
        tableText.includes('sk-api-prod') &&
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
    ['请求', '错误', '费用', '延迟', 'Token'].every((text) =>
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
  await page.getByRole('button', { name: '错误', exact: true }).click()
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
      metrics.pressed[0] === '错误' &&
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
      hoverMetrics.text.includes('失败请求') &&
      hoverMetrics.text.includes('错误率') &&
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
        main?.querySelector(
          'input[placeholder="搜索备注、完整凭据、前缀或后四位"]'
        )
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
    const tabTexts = Array.from(
      main?.querySelectorAll('[role="tab"]') || []
    ).map((node) => node.textContent.trim())

    return {
      hasDailySummary: document.body.innerText.includes('每日模型汇总'),
      hasDetailButton: document.body.innerText.includes('详情'),
      hasPagination: document.body.innerText.includes('共 12 条'),
      hasSidebarUsageNav: document.body.innerText.includes('用量日志'),
      hasTableRefreshAction: Array.from(
        main?.querySelectorAll('button') || []
      ).some((node) => node.textContent.trim().includes('刷新当前页')),
      hasTimeRangeFilter: Boolean(
        main?.querySelector('[role="combobox"][aria-label="时间范围"]')
      ),
      hasUpstreamFilter: Boolean(
        main?.querySelector('[role="combobox"][aria-label="实际执行上游"]')
      ),
      hasUpstreamStats: document.body.innerText.includes('上游分布'),
      hasUsageTabs: [
        '每日模型',
        '凭据统计',
        '会话聚合',
        '调用明细',
        '异常请求',
      ].every((text) => tabTexts.includes(text)),
      hasUsageWindowSummary: document.body.innerText.includes('24h 范围内第'),
      mainHeight: mainRect?.height || 0,
      tableHeight: tableRect?.height || 0,
      tableWidth: tableRect?.width || 0,
    }
  })

  assert(metrics.hasSidebarUsageNav, `${scenarioName} 缺少后台侧栏 usage 入口`)
  assert(metrics.hasTimeRangeFilter, `${scenarioName} 缺少 usage 时间范围筛选`)
  assert(metrics.hasUpstreamFilter, `${scenarioName} 缺少实际上游筛选`)
  assert(metrics.hasUpstreamStats, `${scenarioName} 缺少上游分布统计`)
  assert(metrics.hasUsageTabs, `${scenarioName} 缺少 usage 分段视图`)
  assert(metrics.hasDailySummary, `${scenarioName} 缺少每日模型默认视图`)
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
  await assertUsageDailyModelDetail(page, scenarioName)
  await assertUsageKeyStatsTab(page, scenarioName)
  await assertUsageSessionTab(page, scenarioName)
  await assertUsageDetailsTab(page, scenarioName)
  await assertUsageTimeRangeRequest(page, scenarioName)
  await assertUsagePaginationRequest(page, scenarioName)
  await assertUsageErrorsTab(page, scenarioName)
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
    `${scenarioName} 未请求完整凭据 token 窗口: ${JSON.stringify(calls)}`
  )
}

async function assertUsageDailyModelDetail(page, scenarioName) {
  await expectText(page, '每日模型汇总')
  await expectText(page, 'gpt-5.4')
  await page.getByRole('button', { name: '详情', exact: true }).first().click()
  await expectText(page, '输入 Tokens')
  await expectText(page, '凭据备注')
  await expectText(page, 'production-api-key')
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
      document.querySelector(
        'main input[placeholder="搜索备注、完整凭据、前缀或后四位"]'
      )
    ),
    hasStatsRows:
      document.body.innerText.includes('production-api-key') &&
      document.body.innerText.includes('staging-key-with-long-name'),
  }))
  assert(metrics.hasSearchInput, `${scenarioName} 凭据统计缺少搜索框`)
  assert(metrics.hasStatsRows, `${scenarioName} 凭据统计缺少统计行`)
}

async function assertUsageSessionTab(page, scenarioName) {
  await page.getByRole('tab', { name: '会话聚合', exact: true }).click()
  await expectText(page, '会话聚合')
  await expectText(page, 'session-style-l1')
  await expectText(page, 'production-api-key')
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
  await expectText(page, 'production-api-key')
  await expectText(page, '缓存输入 / 推理输出')
  await expectText(page, '缓存输入')
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
            call.method === 'usage_list' && call.params?.success === false
        )
    ) {
      return
    }
    await delay(100)
  }

  assert.fail(
    `${scenarioName} 切换异常请求后未按失败状态查询: ${JSON.stringify(
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
      hasFullPlainKey: document.body.innerText.includes('ogw_mock_prod_8a2c'),
      hasCurrentOperationRow: document.body.innerText.includes('当前操作'),
      hasPagination: Boolean(main?.querySelector('.admin-table-pagination')),
      hasRemarkHeader: document.body.innerText.includes('备注'),
      hasCreatedAtHeader: document.body.innerText.includes('创建时间'),
      hasUpdatedAtHeader: document.body.innerText.includes('更新时间'),
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
        main?.querySelector(
          'input[placeholder="搜索备注、完整凭据、前缀或后四位"]'
        )
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
      tableWidth: tableRect?.width || 0,
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
          const cell = row.children[7]
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
  assert(metrics.hasFullPlainKey, `${scenarioName} 缺少完整 key 展示`)
  assert(metrics.hasRemarkHeader, `${scenarioName} 缺少备注列表列`)
  assert(metrics.hasCreatedAtHeader, `${scenarioName} 缺少创建时间列`)
  assert(metrics.hasUpdatedAtHeader, `${scenarioName} 缺少更新时间列`)
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
    nextText: 'extra-api-key-9',
    previousText: 'production-api-key',
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
        node.querySelector('input[placeholder="例如内部测试 key"]')
      ),
      hasTokenLimitInput: Boolean(
        node.querySelector('input[placeholder="0 表示不限，1 = 100 万 token"]')
      ),
      hasDetailedTokenLabels:
        node.innerText.includes('细分 Token 限制') &&
        node.innerText.includes('每日输入 Token（百万）') &&
        node.innerText.includes('每日非缓存输入（百万）') &&
        node.innerText.includes('每日输出 Token（百万）'),
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
  assert(
    metrics.hasTokenLimitInput,
    `${scenarioName} 新建弹窗缺少百万 token 输入框`
  )
  assert(
    metrics.hasDetailedTokenLabels,
    `${scenarioName} 新建弹窗缺少细分 token 标签`
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
      node.querySelectorAll(
        'input[placeholder="0 表示不限，1 = 100 万 token"]'
      )
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
    .locator('input[placeholder="例如内部测试 key"]')
    .inputValue()
  assert.equal(
    value,
    'production-api-key',
    `${scenarioName} 双击 key 行未打开对应编辑弹窗`
  )
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
    const currentStyle = current ? window.getComputedStyle(current) : null
    const pageSizeStyle = pageSize ? window.getComputedStyle(pageSize) : null
    const currentRect = current?.getBoundingClientRect()
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
      paginationMetrics.currentWidth >= 40 &&
      paginationMetrics.currentHeight >= 40 &&
      paginationMetrics.currentBorderRadius.includes('50%') &&
      !paginationMetrics.overflowsX &&
      !paginationMetrics.overflowsY,
    `${scenarioName} 分页数字页码盒模型异常: ${JSON.stringify(
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
      selectedRows: 1,
    },
    `${scenarioName} 单击第一行后应只选中第一条`
  )

  await rows.nth(1).click()
  assert.deepEqual(
    await readKeyTableSelectionState(page),
    {
      checked: [false, true, false, false, false, false, false, false],
      selectedRows: 1,
    },
    `${scenarioName} 单击第二行后应互斥切换选择`
  )

  await checkboxes.nth(1).click()
  assert.deepEqual(
    await readKeyTableSelectionState(page),
    {
      checked: [false, false, false, false, false, false, false, false],
      selectedRows: 0,
    },
    `${scenarioName} 再次点击已选选择框应清空选择`
  )

  await rows.nth(1).click()
  assert.deepEqual(
    await readKeyTableSelectionState(page),
    {
      checked: [false, true, false, false, false, false, false, false],
      selectedRows: 1,
    },
    `${scenarioName} 选择清空后应可重新单击行选中`
  )
}

async function readKeyTableSelectionState(page) {
  return page.evaluate(() => ({
    checked: Array.from(
      document.querySelectorAll('main table tbody input[type="checkbox"]')
    ).map((node) => node.checked),
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
  assert(metrics.hasDisableButton, `${scenarioName} 缺少模型启停操作`)
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

async function installAuthConfigMock(page) {
  await page.route('**/auth/oauth/config', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ enabled: false, provider: '' }),
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
    state.upstreamMode = strategy === 'codex_cli' ? 'codex_cli' : 'codex_backend'
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
        key_prefix: 'sk-api-prod',
        created_at: 1777900000,
        updated_at: 1777950000,
        last_used_at: 1778000000,
        name: 'production-api-key',
        plain_key: 'ogw_mock_prod_8a2c',
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
        key_prefix: 'sk-api-stage',
        created_at: 1777800000,
        updated_at: 1777850000,
        last_used_at: 0,
        name: 'staging-key-with-long-name-for-overflow-check',
        plain_key: 'ogw_mock_stage_3f9d',
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
        key_prefix: `sk-api-extra-${id}`,
        created_at: 1777700000 - index * 1_000,
        updated_at: 1777750000 - index * 1_000,
        last_used_at: 1777990000 - index * 100,
        name: `extra-api-key-${id}`,
        plain_key: `ogw_mock_extra_${id}`,
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

  if (method === 'model_list') {
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
    }))
    const extraModels = []
    return {
      items: [...baseModels, ...extraModels],
      total: baseModels.length + extraModels.length,
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
          api_key_name: 'production-api-key',
          api_key_prefix: 'sk-api-prod',
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
          upstream_configured_mode: 'codex_backend',
          upstream_error_type: '',
          upstream_fallback: false,
          upstream_mode: 'codex_backend',
        },
        {
          api_key_name: 'production-api-key',
          api_key_prefix: 'sk-api-prod',
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
          api_key_name: 'staging-key-with-long-name-for-overflow-check',
          api_key_prefix: 'sk-api-stage',
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
          api_key_name: 'production-api-key',
          api_key_prefix: 'sk-api-prod',
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
          api_key_name: 'staging-key-with-long-name-for-overflow-check',
          api_key_prefix: 'sk-api-stage',
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
          api_key_name: 'production-api-key',
          api_key_prefix: 'sk-api-prod',
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
        },
        {
          api_key_id: 2,
          api_key_name: 'staging-key-with-long-name-for-overflow-check',
          api_key_prefix: 'sk-api-stage',
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
