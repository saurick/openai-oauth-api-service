import React, { useEffect, useState } from 'react'
import {
  ADMIN_THEME_MODES,
  ADMIN_THEME_CHANGE_EVENT,
  ADMIN_THEME_STORAGE_KEY,
  applyAdminThemeMode,
  getInitialAdminThemeMode,
  setAdminThemeMode,
} from '@/common/theme/adminTheme'

const THEME_OPTIONS = [
  { icon: SystemIcon, label: '跟系统', mode: ADMIN_THEME_MODES.SYSTEM },
  { icon: SunIcon, label: '浅色', mode: ADMIN_THEME_MODES.LIGHT },
  { icon: MoonIcon, label: '暗色', mode: ADMIN_THEME_MODES.DARK },
]

export default function AdminThemeToggle({ className = '' }) {
  const [currentMode, setCurrentMode] = useState(() => getInitialAdminThemeMode())

  useEffect(() => {
    setCurrentMode(applyAdminThemeMode(getInitialAdminThemeMode()).mode)

    const handleThemeChange = (event) => {
      const nextMode = event.detail?.mode || getInitialAdminThemeMode()
      setCurrentMode(applyAdminThemeMode(nextMode).mode)
    }
    const handleStorage = (event) => {
      if (event.key !== ADMIN_THEME_STORAGE_KEY) return
      setCurrentMode(
        applyAdminThemeMode(event.newValue || getInitialAdminThemeMode()).mode
      )
    }
    const handleSystemThemeChange = () => {
      setCurrentMode(applyAdminThemeMode(getInitialAdminThemeMode()).mode)
    }
    const mediaQuery = window.matchMedia?.('(prefers-color-scheme: dark)')

    window.addEventListener(ADMIN_THEME_CHANGE_EVENT, handleThemeChange)
    window.addEventListener('storage', handleStorage)
    mediaQuery?.addEventListener?.('change', handleSystemThemeChange)
    return () => {
      window.removeEventListener(ADMIN_THEME_CHANGE_EVENT, handleThemeChange)
      window.removeEventListener('storage', handleStorage)
      mediaQuery?.removeEventListener?.('change', handleSystemThemeChange)
    }
  }, [])

  const handleSetMode = (nextMode) => {
    setCurrentMode(setAdminThemeMode(nextMode).mode)
  }

  return (
    <div
      data-admin-theme-toggle
      className={`admin-theme-toggle ${className}`}
      role="group"
      aria-label="界面主题"
    >
      {THEME_OPTIONS.map(({ icon: Icon, label, mode }) => {
        const active = currentMode === mode
        return (
          <button
            key={mode}
            type="button"
            data-admin-theme-option={mode}
            aria-pressed={active}
            className="admin-theme-toggle-option"
            onClick={() => handleSetMode(mode)}
          >
            <span className="admin-theme-toggle-icon" aria-hidden="true">
              <Icon />
            </span>
            <span>{label}</span>
          </button>
        )
      })}
    </div>
  )
}

function SystemIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path
        d="M4 5h16v11H4V5Zm5 15h6m-3-4v4"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeLinejoin="round"
        strokeWidth="2"
      />
    </svg>
  )
}

function MoonIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path
        d="M20.2 14.8A7.6 7.6 0 0 1 9.2 3.8 8.5 8.5 0 1 0 20.2 14.8Z"
        fill="none"
        stroke="currentColor"
        strokeLinejoin="round"
        strokeWidth="2"
      />
    </svg>
  )
}

function SunIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-4 w-4" aria-hidden="true">
      <path
        d="M12 4V2m0 20v-2m8-8h2M2 12h2m13.7-5.7 1.4-1.4M4.9 19.1l1.4-1.4m0-11.4L4.9 4.9m14.2 14.2-1.4-1.4M12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z"
        fill="none"
        stroke="currentColor"
        strokeLinecap="round"
        strokeWidth="2"
      />
    </svg>
  )
}
