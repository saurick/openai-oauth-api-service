// web/src/index.jsx
import React, { StrictMode } from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter as Router } from 'react-router-dom'
import { HelmetProvider } from 'react-helmet-async'
import './tailwind.css'
import App from './App'
import { AppAlertProvider } from '@/common/components/modal/AppAlertProvider'

// 只在开发环境 & 打开开关时启用 mock
if (import.meta.env.DEV && import.meta.env.VITE_ENABLE_RPC_MOCK === 'true') {
  (async () => {
    const { setupJsonRpcMockServer } = await import('./mocks/jsonRpcMockServer')
    setupJsonRpcMockServer()
  })()
}

const rootElement = document.getElementById('root')
if (!rootElement) {
  throw new Error('Root element #root not found')
}

const root = ReactDOM.createRoot(rootElement)

root.render(
  <StrictMode>
    <HelmetProvider>
      <Router basename={import.meta.env.BASE_URL}>
        <AppAlertProvider>
          <App />
        </AppAlertProvider>
      </Router>
    </HelmetProvider>
  </StrictMode>
)
