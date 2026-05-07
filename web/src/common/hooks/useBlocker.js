import { useEffect, useContext } from 'react'
import { UNSAFE_NavigationContext } from 'react-router-dom'

export const useBlocker = (blocker, when = true) => {
  const { navigator } = useContext(UNSAFE_NavigationContext)

  useEffect(() => {
    if (!when) return
    console.log(navigator, 'navigator')
    const unblock = navigator.block((tx) => {
      const autoUnblockingTx = {
        ...tx,
        retry() {
          unblock()
          tx.retry()
        },
      }

      blocker(autoUnblockingTx)
    })

    return unblock
  }, [navigator, blocker, when])
}
