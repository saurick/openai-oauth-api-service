const DAY_SECONDS = 24 * 60 * 60
const MINIMUM_PACE_EXPECTED_PERCENT = 3

function asFiniteNumber(value) {
  if (value == null || value === '') return null
  const number = Number(value)
  return Number.isFinite(number) ? number : null
}

function asNonNegativeNumber(value) {
  const number = asFiniteNumber(value)
  return number == null ? 0 : Math.max(0, number)
}

function clampPercent(value) {
  return Math.min(100, Math.max(0, asNonNegativeNumber(value)))
}

export function formatCodexUsageCost(value) {
  const number = Number(value)
  if (value == null || !Number.isFinite(number)) return '未配置价格'
  return new Intl.NumberFormat('en-US', {
    currency: 'USD',
    maximumFractionDigits: 2,
    minimumFractionDigits: 2,
    style: 'currency',
  }).format(number)
}

export function rateLimitWindowLabel(window, slot = 'primary') {
  const durationMinutes = asFiniteNumber(window?.window_duration_mins)
  if (durationMinutes == null || durationMinutes <= 0) {
    return slot === 'secondary' ? '次级额度窗口' : '主额度窗口'
  }
  if (durationMinutes === (7 * DAY_SECONDS) / 60) return '每周额度'
  if (durationMinutes % (DAY_SECONDS / 60) === 0) {
    return `${durationMinutes / (DAY_SECONDS / 60)} 天额度`
  }
  if (durationMinutes % 60 === 0) {
    return `${durationMinutes / 60} 小时额度`
  }
  return `${Math.round(durationMinutes)} 分钟额度`
}

function toDateMilliseconds(value) {
  if (value == null || value === '') return null
  if (value instanceof Date) return value.getTime()
  if (typeof value === 'number') return value
  const parsed = new Date(value).getTime()
  return Number.isFinite(parsed) ? parsed : null
}

function resetAtMilliseconds(window) {
  const explicitTime = toDateMilliseconds(window?.resets_at_time)
  if (explicitTime != null) return explicitTime

  const raw = asFiniteNumber(window?.resets_at)
  if (raw == null || raw <= 0) return null
  return raw > 1_000_000_000_000 ? raw : raw * 1000
}

function paceStage(deltaPercent) {
  const magnitude = Math.abs(deltaPercent)
  if (magnitude <= 2) return 'on_track'
  if (magnitude <= 6) {
    return deltaPercent >= 0 ? 'slightly_ahead' : 'slightly_behind'
  }
  if (magnitude <= 12) return deltaPercent >= 0 ? 'ahead' : 'behind'
  return deltaPercent >= 0 ? 'far_ahead' : 'far_behind'
}

export function calculateRateLimitPace(window, now = Date.now()) {
  const durationMinutes = asFiniteNumber(window?.window_duration_mins)
  const resetAt = resetAtMilliseconds(window)
  const nowAt = toDateMilliseconds(now)
  if (
    durationMinutes == null ||
    durationMinutes <= 0 ||
    resetAt == null ||
    nowAt == null
  ) {
    return null
  }

  const durationSeconds = durationMinutes * 60
  const timeUntilResetSeconds = (resetAt - nowAt) / 1000
  if (timeUntilResetSeconds <= 0 || timeUntilResetSeconds > durationSeconds) {
    return null
  }

  const elapsedSeconds = Math.min(
    durationSeconds,
    Math.max(0, durationSeconds - timeUntilResetSeconds)
  )
  const actualUsedPercent = clampPercent(window?.used_percent)
  if (elapsedSeconds === 0 && actualUsedPercent > 0) return null

  const expectedUsedPercent = clampPercent(
    (elapsedSeconds / durationSeconds) * 100
  )
  if (expectedUsedPercent < MINIMUM_PACE_EXPECTED_PERCENT) return null

  const deltaPercent = actualUsedPercent - expectedUsedPercent
  let etaSeconds = null
  let willLastToReset = false

  if (actualUsedPercent >= 100) {
    etaSeconds = 0
  } else if (elapsedSeconds > 0 && actualUsedPercent > 0) {
    const usedPercentPerSecond = actualUsedPercent / elapsedSeconds
    const candidateSeconds = (100 - actualUsedPercent) / usedPercentPerSecond
    if (candidateSeconds >= timeUntilResetSeconds) {
      willLastToReset = true
    } else {
      etaSeconds = Math.max(0, candidateSeconds)
    }
  } else if (elapsedSeconds > 0) {
    willLastToReset = true
  }

  return {
    actualUsedPercent,
    deltaPercent,
    elapsedSeconds,
    etaSeconds,
    expectedUsedPercent,
    resetAt,
    stage: paceStage(deltaPercent),
    timeUntilResetSeconds,
    willLastToReset,
  }
}

function summaryNumber(summary, field) {
  return asNonNegativeNumber(summary?.[field])
}

function summaryCost(summary) {
  if (
    !Object.prototype.hasOwnProperty.call(summary || {}, 'estimated_cost_usd')
  ) {
    return null
  }
  const cost = asFiniteNumber(summary?.estimated_cost_usd)
  return cost == null ? null : Math.max(0, cost)
}

function bucketDay(timestamp) {
  const seconds = asFiniteNumber(timestamp)
  if (seconds == null || seconds <= 0) return null
  const date = new Date(seconds * 1000)
  if (Number.isNaN(date.getTime())) return null
  return date.toISOString().slice(0, 10)
}

function topModelFromScores(scores) {
  const entries = [...scores.entries()]
  const hasCompleteCost =
    entries.length > 0 &&
    entries.every(([, score]) => score.hasKnownCost && !score.hasUnknownCost)
  const candidates = entries
  const basis = hasCompleteCost ? 'cost' : 'tokens'

  candidates.sort(([leftName, left], [rightName, right]) => {
    if (basis === 'cost' && left.costUSD !== right.costUSD) {
      return right.costUSD - left.costUSD
    }
    if (left.totalTokens !== right.totalTokens) {
      return right.totalTokens - left.totalTokens
    }
    return leftName.localeCompare(rightName)
  })

  if (candidates.length === 0) return { basis: null, name: null }
  return { basis, name: candidates[0][0] }
}

export function buildCodexUsageOverview({
  buckets = [],
  endTime,
  periodSummary = {},
  startTime,
  todaySummary = {},
} = {}) {
  const safeBuckets = Array.isArray(buckets) ? buckets : []
  const start = asFiniteNumber(startTime)
  const end = asFiniteNumber(endTime)
  const days = new Map()
  const models = new Map()

  safeBuckets.forEach((bucket) => {
    const timestamp = asFiniteNumber(bucket?.bucket_start)
    if (timestamp == null) return
    if (start != null && timestamp < start - DAY_SECONDS) return
    if (end != null && timestamp > end) return

    const day = bucketDay(timestamp)
    if (!day) return

    const totalTokens = asNonNegativeNumber(bucket?.total_tokens)
    const totalRequests = asNonNegativeNumber(bucket?.total_requests)
    const rawCost = asFiniteNumber(bucket?.estimated_cost_usd)
    const costUSD = rawCost == null ? null : Math.max(0, rawCost)
    const currentDay = days.get(day) || {
      costComplete: true,
      costUSD: 0,
      date: day,
      totalRequests: 0,
      totalTokens: 0,
    }
    currentDay.totalTokens += totalTokens
    currentDay.totalRequests += totalRequests
    if (costUSD == null) {
      currentDay.costComplete = false
    } else {
      currentDay.costUSD += costUSD
    }
    days.set(day, currentDay)

    const model = String(bucket?.model || '').trim()
    if (!model || model === '-') return
    const currentModel = models.get(model) || {
      costUSD: 0,
      hasKnownCost: false,
      hasUnknownCost: false,
      totalTokens: 0,
    }
    currentModel.totalTokens += totalTokens
    if (costUSD != null) {
      currentModel.costUSD += costUSD
      currentModel.hasKnownCost = true
    } else {
      currentModel.hasUnknownCost = true
    }
    models.set(model, currentModel)
  })

  const daily = [...days.values()]
    .sort((left, right) => left.date.localeCompare(right.date))
    .slice(-30)
    .map((day) => ({
      costUSD: day.costComplete ? day.costUSD : null,
      date: day.date,
      totalRequests: day.totalRequests,
      totalTokens: day.totalTokens,
    }))
  const latest = daily.at(-1) || null
  const topModel = topModelFromScores(models)

  return {
    daily,
    latestDay: latest?.date || null,
    latestTokens: latest?.totalTokens ?? null,
    periodCostUSD: summaryCost(periodSummary),
    periodRequests: summaryNumber(periodSummary, 'total_requests'),
    periodTokens: summaryNumber(periodSummary, 'total_tokens'),
    todayCostUSD: summaryCost(todaySummary),
    todayRequests: summaryNumber(todaySummary, 'total_requests'),
    todayTokens: summaryNumber(todaySummary, 'total_tokens'),
    topModelBasis: topModel.basis,
    topModelName: topModel.name,
  }
}
