// web/src/common/auth/authBus.js
const listeners = new Set()

export const authBus = {
  onUnauthorized(fn) {
    listeners.add(fn)
    return () => listeners.delete(fn)
  },
  emitUnauthorized(payload) {
    for (const fn of listeners) fn(payload)
  },
}
