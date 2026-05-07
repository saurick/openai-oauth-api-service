import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react'
import AlertDialog from '@/common/components/modal/AlertDialog'
import { registerAlert } from '@/common/components/modal/alertBridge'

const AlertContext = createContext(null)

export function AppAlertProvider({ children }) {
  const [state, setState] = useState({
    open: false,
    title: '提示',
    message: '',
    confirmText: '确定',
    onConfirm: null,
  })

  const close = useCallback(() => {
    setState((current) => ({ ...current, open: false }))
  }, [])

  const alert = useCallback((opts = {}) => {
    setState({
      open: true,
      title: opts.title ?? '提示',
      message: opts.message ?? '',
      confirmText: opts.confirmText ?? '确定',
      onConfirm: opts.onConfirm ?? null,
    })
  }, [])

  useEffect(() => {
    registerAlert(alert)
  }, [alert])

  const value = useMemo(() => ({ alert, close }), [alert, close])

  return (
    <AlertContext.Provider value={value}>
      {children}
      <AlertDialog
        open={state.open}
        onClose={close}
        title={state.title}
        message={state.message}
        confirmText={state.confirmText}
        onConfirm={state.onConfirm}
      />
    </AlertContext.Provider>
  )
}

export function useAppAlert() {
  const ctx = useContext(AlertContext)
  if (!ctx) {
    throw new Error('useAppAlert must be used within <AppAlertProvider />')
  }
  return ctx
}
