import { useCallback, useEffect, useState } from 'react'
import { useLocation, useNavigate } from 'react-router'
import { useBlocker } from './useBlocker'

const useCallbackPrompt = (when) => {
  const navigate = useNavigate()
  const location = useLocation()
  const [showPrompt, setShowPrompt] = useState(false)
  const [lastLocation, setLastLocation] = useState(null)
  const [confirmedNavigation, setConfirmedNavigation] = useState(false)

  const cancelNavigation = useCallback(() => {
    setShowPrompt(false)
    setLastLocation(null)
  }, [])

  const handleBlockedNavigation = useCallback(
    (nextLocation) => {
      console.log(nextLocation, 'nextLocation')
      if (
        nextLocation.pathname !== '/' &&
        !confirmedNavigation &&
        nextLocation.location.pathname !== location.pathname
      ) {
        setShowPrompt(true)
        setLastLocation(nextLocation)
        return false
      }
      return true
    },
    [confirmedNavigation, location]
  )

  const confirmNavigation = useCallback(() => {
    setShowPrompt(false)
    setConfirmedNavigation(true)
  }, [])

  useEffect(() => {
    console.log(confirmedNavigation, 'confirmedNavigation')
    console.log(lastLocation, 'lastLocation')
    if (confirmedNavigation && lastLocation) {
      navigate(lastLocation.location?.pathname)

      setConfirmedNavigation(false)
    }
  }, [confirmedNavigation, lastLocation])

  useBlocker(handleBlockedNavigation, when)

  return [showPrompt, confirmNavigation, cancelNavigation]
}

export default useCallbackPrompt
