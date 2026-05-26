import React from 'react'
import AdminFrame from '@/common/components/layout/AdminFrame'
import ClientConfigBuilder from './ClientConfigBuilder'

export default function AdminClientConfigPage() {
  return (
    <AdminFrame
      title="客户端配置生成器"
      description="生成 Codex 和 opencode 的 macOS / Windows 最小配置。填写 Base URL、API Key 和必要的 profile 后，可以复制内容或下载配置文件。"
      breadcrumb="转发配置 / 客户端配置生成器"
      actions={
        <a
          className="admin-button"
          href="/client-config"
          target="_blank"
          rel="noreferrer noopener"
        >
          打开公开页
        </a>
      }
    >
      <ClientConfigBuilder />
    </AdminFrame>
  )
}
