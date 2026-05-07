import { useEffect } from 'react'

/**
 * 点击弹窗外隐藏弹窗
 * @param {触发弹窗的ref} triggerInsideRef
 * @param {弹窗ref} insideRef
 * @param {关闭弹窗的callback} callback
 */
const useClickAway = (triggerInsideRef, insideRef, callback) => {
  useEffect(() => {
    const handler = (event) => {
      // console.log(event.target, 'event.target')
      // console.log(insideRef, 'insideRef')
      // console.log(triggerInsideRef, 'triggerInsideRef')
      if (
        triggerInsideRef !== null &&
        insideRef?.current &&
        !insideRef?.current.contains(event.target) &&
        event.target !== triggerInsideRef
      ) {
        callback()
      }
    }
    window.addEventListener('click', handler)

    return () => window.removeEventListener('click', handler)
  }, [callback, insideRef, triggerInsideRef])
}

export default useClickAway
