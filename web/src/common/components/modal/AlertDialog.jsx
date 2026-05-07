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

  return (
    <AppModal open={open} onClose={onClose} className={className}>
      <div className="flex flex-col items-center gap-5 py-2 text-center">
        {title ? (
          <div className="text-xl font-semibold tracking-wide text-slate-50 sm:text-2xl">
            {title}
          </div>
        ) : null}

        {message ? (
          <div className="whitespace-pre-line text-sm leading-7 text-slate-300 sm:text-base">
            {message}
          </div>
        ) : null}

        <button
          type="button"
          onClick={handleConfirm}
          className="min-w-[152px] rounded-full bg-cyan-300 px-6 py-2.5 text-sm font-semibold text-slate-950 transition hover:bg-cyan-200 active:bg-cyan-400"
        >
          {confirmText}
        </button>
      </div>
    </AppModal>
  )
}
