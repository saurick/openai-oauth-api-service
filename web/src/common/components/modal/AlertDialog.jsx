import React from 'react'
import AppModal from '@/common/components/modal/AppModal'

export default function AlertDialog({
  open,
  onClose,
  title = '提示',
  message = '',
  confirmText = '确定',
  onConfirm = null,
  className = '',
}) {
  const handleConfirm = () => {
    onConfirm?.()
    onClose?.()
  }

  const titleId = 'app-alert-title'
  const descriptionId = message ? 'app-alert-description' : undefined

  return (
    <AppModal
      open={open}
      onClose={onClose}
      className={`admin-alert-modal ${className}`.trim()}
      panelProps={{
        role: 'dialog',
        'aria-modal': 'true',
        'aria-labelledby': title ? titleId : undefined,
        'aria-describedby': descriptionId,
      }}
    >
      <div className="admin-modal-header">
        <div>
          {title ? (
            <h2 id={titleId} className="admin-modal-title">
              {title}
            </h2>
          ) : null}
          <p className="admin-modal-description">
            系统需要你重新确认登录状态。
          </p>
        </div>
        <button
          type="button"
          aria-label="关闭弹窗"
          onClick={onClose}
          className="admin-modal-close"
        >
          ×
        </button>
      </div>

      <div className="admin-confirm-body admin-alert-body">
        {message ? (
          <div
            id={descriptionId}
            className="admin-confirm-detail admin-alert-message"
          >
            {message}
          </div>
        ) : null}
      </div>

      <div className="admin-modal-footer admin-confirm-footer admin-alert-footer">
        <button
          type="button"
          onClick={handleConfirm}
          className="admin-button admin-button-primary"
        >
          {confirmText}
        </button>
      </div>
    </AppModal>
  )
}
