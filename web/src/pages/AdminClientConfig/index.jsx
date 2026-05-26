import React from 'react'
import AdminFrame from '@/common/components/layout/AdminFrame'
import ClientConfigBuilder from './ClientConfigBuilder'

export default function AdminClientConfigPage() {
  return (
    <AdminFrame
      title="客户端配置模板"
      description="导出 Codex 和 opencode 的 macOS / Windows 最小配置模板，只保留 Base URL、API Key 和 Codex profile 等必要字段；历史记录、工作区信任、通知路径等个人字段不写入模板。"
      breadcrumb="转发配置 / 客户端配置模板"
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
