import { useEffect } from 'react'
import { useRouter } from '../contexts/RouterContext'
import { ChatInterface } from '../components/ChatInterface'

export function ChatPage() {
  const { currentRoute, navigate } = useRouter()
  const audioId = currentRoute.params?.audioId
  const sessionId = currentRoute.params?.sessionId

  useEffect(() => {
    // If we somehow landed on chat without required params, bounce home
    if (!audioId) {
      navigate({ path: 'home' })
    }
  }, [audioId, navigate])

  if (!audioId) return null

  return (
    <div className="min-h-screen h-screen bg-gray-50 dark:bg-gray-900">
      <div className="h-full">
        <ChatInterface
          transcriptionId={audioId}
          activeSessionId={sessionId}
          onSessionChange={(newSessionId) => {
            if (!newSessionId) {
              // If no session, send back to audio detail
              navigate({ path: 'audio-detail', params: { id: audioId } })
            } else {
              navigate({ path: 'chat', params: { audioId, sessionId: newSessionId } })
            }
          }}
        />
      </div>
    </div>
  )
}

