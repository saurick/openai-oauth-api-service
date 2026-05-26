import React from 'react'
import AppShell from '@/common/components/layout/AppShell'
import AdminThemeToggle from '@/common/components/layout/AdminThemeToggle'
import ClientConfigBuilder from '@/pages/AdminClientConfig/ClientConfigBuilder'

export default function PublicClientConfigPage() {
  return (
    <AppShell variant="admin" className="admin-frame">
      <header className="border-b border-[#dce8df] bg-white/95 backdrop-blur">
        <div className="mx-auto flex min-h-[76px] max-w-6xl flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-5">
          <div>
            <div className="text-sm font-semibold text-[#173b59]">
              Saurick API Console
            </div>
            <div className="mt-1 text-xs text-[#7b8780]">
              公开客户端配置生成器
            </div>
          </div>
          <AdminThemeToggle />
        </div>
      </header>

      <main className="mx-auto grid max-w-6xl gap-5 px-4 py-5 sm:px-5 lg:py-6">
        <div>
          <h1 className="text-2xl font-bold text-[#1f2d25]">
            客户端配置生成器
          </h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[#7b8780]">
            免登录生成 Codex 和 opencode 的 macOS / Windows 最小配置。填写自己的 API Key 后复制或下载配置文件即可。
          </p>
        </div>

        <ClientConfigBuilder publicMode />
      </main>
    </AppShell>
  )
}
