import React from 'react'

export default function AppModal({
  open,
  onClose,
  children,
  className = '',
  panelProps = {},
}) {
  if (!open) return null

  const { className: panelClassName = '', ...restPanelProps } = panelProps

  return (
    <div className="admin-modal-backdrop admin-modal-theme-scope">
      <button
        type="button"
        aria-label="关闭弹窗"
        className="admin-modal-overlay cursor-default"
        onClick={onClose}
      />
      <div
        {...restPanelProps}
        className={`admin-modal-panel ${className} ${panelClassName}`.trim()}
      >
        {children}
      </div>
    </div>
  )
}
