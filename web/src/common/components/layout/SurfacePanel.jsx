import React from 'react'

// 通用内容容器：默认保留深色控制台质感，后台看板使用浅色企业面板。
export default function SurfacePanel({
  children,
  className = '',
  variant = 'dark',
}) {
  if (variant === 'admin') {
    return (
      <div
        className={`relative overflow-hidden rounded-lg border border-[#dde8df] bg-white shadow-[0_2px_10px_rgba(24,61,42,0.08)] ${className}`}
      >
        {children}
      </div>
    )
  }

  return (
    <div
      className={`border-white/12 relative overflow-hidden rounded-[32px] border bg-slate-950/75 shadow-[0_24px_80px_rgba(15,23,42,0.45)] backdrop-blur ${className}`}
    >
      <div className="pointer-events-none absolute inset-x-6 top-0 h-px bg-gradient-to-r from-transparent via-cyan-300/50 to-transparent" />
      <div className="pointer-events-none absolute -left-10 bottom-0 h-40 w-40 rounded-full bg-amber-300/10 blur-3xl" />
      <div className="bg-cyan-300/12 pointer-events-none absolute -right-12 top-0 h-40 w-40 rounded-full blur-3xl" />
      <div className="border-white/6 from-white/6 relative h-full w-full rounded-[32px] border bg-gradient-to-b via-transparent to-transparent">
        {children}
      </div>
    </div>
  )
}
