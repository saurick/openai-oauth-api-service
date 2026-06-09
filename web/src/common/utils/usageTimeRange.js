export const DAY_SECONDS = 24 * 60 * 60

export const DEFAULT_USAGE_TIME_RANGE = 'today'
export const DEFAULT_DAILY_USAGE_TIME_RANGE = '30d'

export const USAGE_TIME_RANGE_OPTIONS = [
  { label: '今天', value: 'today' },
  { label: '24h', value: '24h', seconds: DAY_SECONDS },
  { label: '7 天', value: '7d', seconds: 7 * DAY_SECONDS },
  { label: '30 天', value: '30d', seconds: 30 * DAY_SECONDS },
  { label: '90 天', value: '90d', seconds: 90 * DAY_SECONDS },
  { label: '180 天', value: '180d', seconds: 180 * DAY_SECONDS },
  { label: '1 年', value: '1y', seconds: 365 * DAY_SECONDS },
  { label: '2 年', value: '2y', seconds: 2 * 365 * DAY_SECONDS },
  { label: '3 年', value: '3y', seconds: 3 * 365 * DAY_SECONDS },
  { label: '5 年', value: '5y', seconds: 5 * 365 * DAY_SECONDS },
]

export function getUsageTimeRange(
  value,
  fallbackValue = DEFAULT_USAGE_TIME_RANGE
) {
  return (
    USAGE_TIME_RANGE_OPTIONS.find((item) => item.value === value) ||
    USAGE_TIME_RANGE_OPTIONS.find((item) => item.value === fallbackValue) ||
    USAGE_TIME_RANGE_OPTIONS.find(
      (item) => item.value === DEFAULT_USAGE_TIME_RANGE
    ) ||
    USAGE_TIME_RANGE_OPTIONS[0]
  )
}

export function startOfLocalDayUnix(date = new Date()) {
  return Math.floor(
    new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime() /
      1000
  )
}

export function getUsageTimeWindow(
  value,
  now = Math.floor(Date.now() / 1000),
  fallbackValue = DEFAULT_USAGE_TIME_RANGE
) {
  const endTime = Math.trunc(Number(now))
  const safeEndTime = Number.isFinite(endTime)
    ? endTime
    : Math.floor(Date.now() / 1000)
  const range = getUsageTimeRange(value, fallbackValue)
  const startTime =
    range.value === 'today'
      ? startOfLocalDayUnix(new Date(safeEndTime * 1000))
      : safeEndTime - range.seconds

  return {
    endTime: safeEndTime,
    range,
    startTime: Math.min(startTime, safeEndTime),
  }
}
