// web/src/common/components/modal/alertBridge.js
let _alert = null

export function registerAlert(fn) {
  _alert = fn
}

export function appAlert(opts) {
  if (_alert) return _alert(opts)
  console.warn('[alertBridge] alert not registered yet:', opts)
}
