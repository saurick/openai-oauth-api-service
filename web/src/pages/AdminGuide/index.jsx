import React from 'react'
import AdminFrame from '@/common/components/layout/AdminFrame'
import SurfacePanel from '@/common/components/layout/SurfacePanel'

const DEFAULT_ITEMS = [
  '管理员登录入口、受保护路由和 API 运营后台',
  '健康检查、PostgreSQL 迁移、Compose 部署骨架和质量门禁',
  '历史 Python MVP 仅作早期用量记录参考，不作为主路径',
]

const CUSTOMIZE_ITEMS = [
  'API key 创建、启停、模型白名单和完整 key 展示',
  'OpenAI 兼容 /v1/models、/v1/chat/completions、/v1/responses 调用入口',
  '模型列表管理、usage 看板、最近请求、导出、价格、告警和 Codex CLI 上游配置',
]

const NEXT_ITEMS = [
  '组织用户管理和普通用户门户不作为当前前端主路径',
  'key + 模型级 RPM、TPM、日/月配额、审计日志和站内告警已落库',
  '后续按真实生产反馈继续收敛 API 转发、计量和后台运维能力',
]

function GuideList({ title, description, items, accentClass }) {
  return (
    <SurfacePanel variant="admin" className="p-5 sm:p-6">
      <div className="space-y-4">
        <div
          className={`inline-flex rounded-full border px-3 py-1 text-xs font-medium uppercase tracking-[0.2em] ${accentClass}`}
        >
          {title}
        </div>
        <div className="text-sm leading-6 text-slate-300">{description}</div>
        <div className="space-y-3">
          {items.map((item) => (
            <div
              key={item}
              className="rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-3 text-sm text-slate-100"
            >
              {item}
            </div>
          ))}
        </div>
      </div>
    </SurfacePanel>
  )
}

export default function AdminGuidePage() {
  return (
    <AdminFrame
      breadcrumb="API / 接入路线"
      title="长期维护路线"
      description="当前仓库已经收口为长期维护结构，API key 生成、OpenAI 兼容转发和 token/usage 统计能力已迁入 Go 后端。后台前端保留管理登录、key、模型和 usage 主路径。"
    >
      <div className="grid gap-6 lg:grid-cols-2">
        <GuideList
          title="默认保留"
          description="这些能力已经作为当前仓库基线保留，用于支撑后续登录、转发和统计功能。"
          items={DEFAULT_ITEMS}
          accentClass="border-emerald-300/30 bg-emerald-300/10 text-emerald-100"
        />
        <GuideList
          title="API key 与计量"
          description="这些是当前已落到主路径的 API key 生成、模型管理和用量统计能力。"
          items={CUSTOMIZE_ITEMS}
          accentClass="border-amber-300/30 bg-amber-300/10 text-amber-100"
        />
      </div>

      <SurfacePanel variant="admin" className="p-5 sm:p-6">
        <div className="space-y-4">
          <div className="inline-flex rounded-full border border-white/10 bg-white/[0.04] px-3 py-1 text-xs font-medium uppercase tracking-[0.2em] text-slate-100">
            下一步
          </div>
          <div className="text-sm leading-6 text-slate-300">
            下一阶段路线已补齐到当前主路径，后续只按真实生产反馈继续收敛。
          </div>
          <div className="grid gap-3 md:grid-cols-3">
            {NEXT_ITEMS.map((item) => (
              <div
                key={item}
                className="rounded-2xl border border-white/10 bg-black/20 px-4 py-3 text-sm text-cyan-100"
              >
                {item}
              </div>
            ))}
          </div>
        </div>
      </SurfacePanel>
    </AdminFrame>
  )
}
