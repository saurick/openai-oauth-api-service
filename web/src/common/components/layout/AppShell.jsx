import React from 'react'

const VARIANTS = {
  dark: {
    root: 'bg-[#0b1220] text-slate-100',
    bg: (
      <>
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.16),_transparent_28%),radial-gradient(circle_at_bottom_right,_rgba(245,158,11,0.12),_transparent_24%),linear-gradient(160deg,#0b1220_0%,#111827_44%,#0f172a_100%)]" />
        <div className="pointer-events-none absolute inset-0 opacity-40 [background-image:linear-gradient(rgba(148,163,184,0.08)_1px,transparent_1px),linear-gradient(90deg,rgba(148,163,184,0.08)_1px,transparent_1px)] [background-size:36px_36px]" />
      </>
    ),
  },
  adminLogin: {
    root: 'bg-[#f7fbfc] text-slate-950',
    bg: (
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(155deg,#eaf6fb_0%,#f9fcfd_48%,#fff7df_100%)]" />
    ),
  },
  admin: {
    root: 'bg-[#f4f8f5] text-slate-950',
    bg: (
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(180deg,#fbfdfb_0%,#f3f8f4_100%)]" />
    ),
  },
}

// 提供应用级背景和承载层；后台页通过 variant 收口不同视觉。
export default function AppShell({
  children,
  className = '',
  variant = 'dark',
}) {
  const current = VARIANTS[variant] || VARIANTS.dark

  return (
    <div
      className={`relative min-h-screen ${
        variant === 'dark' ? 'overflow-hidden' : 'overflow-x-hidden'
      } ${current.root} ${className}`}
    >
      {current.bg}
      <div className="relative min-h-screen">{children}</div>
    </div>
  )
}
