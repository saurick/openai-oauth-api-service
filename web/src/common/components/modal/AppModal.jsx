import React from 'react'
import SurfacePanel from '@/common/components/layout/SurfacePanel'

export default function AppModal({ open, onClose, children, className = '' }) {
  if (!open) return null

  return (
    <div className="bg-slate-950/72 fixed inset-0 z-50 flex items-center justify-center px-4 py-6 backdrop-blur-sm">
      <button
        type="button"
        aria-label="关闭弹窗"
        className="absolute inset-0 cursor-default"
        onClick={onClose}
      />
      <SurfacePanel
        className={`relative z-10 w-full max-w-[640px] px-6 py-6 sm:px-8 sm:py-8 ${className}`}
      >
        {children}
      </SurfacePanel>
    </div>
  )
}
