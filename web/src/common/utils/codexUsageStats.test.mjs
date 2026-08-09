import assert from 'node:assert/strict'
import test from 'node:test'
import fs from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'

function loadModule() {
  const filePath = path.resolve(import.meta.dirname, './codexUsageStats.js')
  const source = fs.readFileSync(filePath, 'utf8')
  const transformed = source
    .replace(/export function /g, 'function ')
    .concat(
      '\nmodule.exports = { calculateRateLimitPace, buildCodexUsageOverview, formatCodexUsageCost };\n'
    )
  const sandbox = { module: { exports: {} }, exports: {} }
  vm.runInNewContext(transformed, sandbox, { filename: filePath })
  return sandbox.module.exports
}

const {
  buildCodexUsageOverview,
  calculateRateLimitPace,
  formatCodexUsageCost,
} = loadModule()

test('codexUsageStats: 真实大额费用保持合法的两位小数格式', () => {
  assert.equal(formatCodexUsageCost(1545.710638), '$1,545.71')
  assert.equal(formatCodexUsageCost(32.032506), '$32.03')
  assert.equal(formatCodexUsageCost(null), '未配置价格')
})

test('codexUsageStats: 线性节奏允许正常状态同时预测重置前用尽', () => {
  const now = Date.parse('2026-08-09T00:00:00Z')
  const pace = calculateRateLimitPace(
    {
      resets_at_time: '2026-08-15T16:00:00Z',
      used_percent: 6,
      window_duration_mins: 7 * 24 * 60,
    },
    now
  )

  assert.equal(pace.stage, 'on_track')
  assert.equal(pace.willLastToReset, false)
  assert(pace.deltaPercent > 1 && pace.deltaPercent < 2)
  assert(pace.etaSeconds > 5 * 24 * 60 * 60)
  assert(pace.etaSeconds < 5.5 * 24 * 60 * 60)
})

test('codexUsageStats: 窗口前 3% 不输出节奏预测', () => {
  const now = Date.parse('2026-08-09T00:00:00Z')
  const pace = calculateRateLimitPace(
    {
      resets_at_time: '2026-08-15T22:00:00Z',
      used_percent: 1,
      window_duration_mins: 7 * 24 * 60,
    },
    now
  )

  assert.equal(pace, null)
})

test('codexUsageStats: 慢于可持续节奏时标记可持续到重置', () => {
  const now = Date.parse('2026-08-09T00:00:00Z')
  const pace = calculateRateLimitPace(
    {
      resets_at_time: '2026-08-14T00:00:00Z',
      used_percent: 10,
      window_duration_mins: 7 * 24 * 60,
    },
    now
  )

  assert.equal(pace.willLastToReset, true)
  assert.equal(pace.etaSeconds, null)
  assert(pace.deltaPercent < 0)
})

test('codexUsageStats: 汇总 Today/30d 并按费用优先选主模型', () => {
  const overview = buildCodexUsageOverview({
    startTime: 1778284800,
    endTime: 1778544000,
    todaySummary: {
      estimated_cost_usd: 3.25,
      total_requests: 4,
      total_tokens: 1200,
    },
    periodSummary: {
      estimated_cost_usd: 18.5,
      total_requests: 20,
      total_tokens: 9000,
    },
    buckets: [
      {
        bucket_start: 1778371200,
        estimated_cost_usd: 2,
        model: 'gpt-5.6-sol',
        total_requests: 2,
        total_tokens: 3000,
      },
      {
        bucket_start: 1778371200,
        estimated_cost_usd: 3,
        model: 'gpt-5.6-terra',
        total_requests: 1,
        total_tokens: 1000,
      },
      {
        bucket_start: 1778457600,
        estimated_cost_usd: 2,
        model: 'gpt-5.6-sol',
        total_requests: 3,
        total_tokens: 2500,
      },
    ],
  })

  assert.equal(overview.todayCostUSD, 3.25)
  assert.equal(overview.periodCostUSD, 18.5)
  assert.equal(overview.periodTokens, 9000)
  assert.equal(overview.topModelName, 'gpt-5.6-sol')
  assert.equal(overview.topModelBasis, 'cost')
  assert.equal(overview.daily.length, 2)
  assert.equal(overview.daily[0].totalTokens, 4000)
  assert.equal(overview.latestTokens, 2500)
})

test('codexUsageStats: 价格缺失时不伪造费用并回退按 Token 选模型', () => {
  const overview = buildCodexUsageOverview({
    periodSummary: { estimated_cost_usd: null, total_tokens: 5000 },
    todaySummary: { estimated_cost_usd: null, total_tokens: 500 },
    buckets: [
      {
        bucket_start: 1778371200,
        estimated_cost_usd: null,
        model: 'gpt-5.6-luna',
        total_tokens: 4000,
      },
      {
        bucket_start: 1778371200,
        estimated_cost_usd: null,
        model: 'gpt-5.6-sol',
        total_tokens: 1000,
      },
    ],
  })

  assert.equal(overview.todayCostUSD, null)
  assert.equal(overview.periodCostUSD, null)
  assert.equal(overview.daily[0].costUSD, null)
  assert.equal(overview.topModelName, 'gpt-5.6-luna')
  assert.equal(overview.topModelBasis, 'tokens')
})

test('codexUsageStats: 部分模型缺价格时也不把局部费用当完整排名', () => {
  const overview = buildCodexUsageOverview({
    buckets: [
      {
        bucket_start: 1778371200,
        estimated_cost_usd: 10,
        model: 'gpt-5.6-sol',
        total_tokens: 1000,
      },
      {
        bucket_start: 1778371200,
        estimated_cost_usd: null,
        model: 'unknown-model',
        total_tokens: 9000,
      },
    ],
  })

  assert.equal(overview.topModelName, 'unknown-model')
  assert.equal(overview.topModelBasis, 'tokens')
})
