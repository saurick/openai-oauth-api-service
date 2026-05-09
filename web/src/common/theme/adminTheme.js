export const ADMIN_THEME_STORAGE_KEY = 'admin_theme'
export const ADMIN_THEME_CHANGE_EVENT = 'admin-theme-change'

export const ADMIN_THEME_MODES = {
  DARK: 'dark',
  LIGHT: 'light',
  SYSTEM: 'system',
}

export const ADMIN_THEMES = {
  DARK: 'dark',
  LIGHT: 'light',
}

export function getSystemAdminTheme() {
  if (
    typeof window !== 'undefined' &&
    window.matchMedia?.('(prefers-color-scheme: dark)').matches
  ) {
    return ADMIN_THEMES.DARK
  }
  return ADMIN_THEMES.LIGHT
}

function normalizeThemeMode(value) {
  return Object.values(ADMIN_THEME_MODES).includes(value)
    ? value
    : ADMIN_THEME_MODES.SYSTEM
}

export function resolveAdminTheme(mode) {
  const normalizedMode = normalizeThemeMode(mode)
  return normalizedMode === ADMIN_THEME_MODES.SYSTEM
    ? getSystemAdminTheme()
    : normalizedMode
}

export function getInitialAdminThemeMode() {
  if (typeof window === 'undefined') return ADMIN_THEME_MODES.SYSTEM

  try {
    return normalizeThemeMode(
      window.localStorage.getItem(ADMIN_THEME_STORAGE_KEY)
    )
  } catch (e) {
    return ADMIN_THEME_MODES.SYSTEM
  }
}

export function applyAdminThemeMode(mode) {
  const nextMode = normalizeThemeMode(mode)
  const nextTheme = resolveAdminTheme(nextMode)

  if (typeof document !== 'undefined') {
    document.documentElement.dataset.adminTheme = nextTheme
    document.documentElement.dataset.adminThemeMode = nextMode
    document.documentElement.style.colorScheme = nextTheme
  }

  return { mode: nextMode, theme: nextTheme }
}

export function setAdminThemeMode(mode) {
  const nextState = applyAdminThemeMode(mode)

  if (typeof window !== 'undefined') {
    try {
      window.localStorage.setItem(ADMIN_THEME_STORAGE_KEY, nextState.mode)
    } catch (e) {
      // localStorage 可能被浏览器策略禁用；主题仍在当前页面生效。
    }
    window.dispatchEvent(
      new CustomEvent(ADMIN_THEME_CHANGE_EVENT, {
        detail: nextState,
      })
    )
  }

  return nextState
}
