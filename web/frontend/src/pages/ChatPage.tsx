import { useEffect } from 'react'
import { useRouter } from '../contexts/RouterContext'
import { ChatInterface } from '../components/ChatInterface'
import { Button } from '../components/ui/button'
import { ArrowLeft } from 'lucide-react'
import { ThemeSwitcher } from '../components/ThemeSwitcher'

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
      {/* Header */}
      <div className="bg-background/80 backdrop-blur supports-[backdrop-filter]:bg-background/60 sticky top-0 z-10 shadow-sm">
        <div className="max-w-[1400px] mx-auto px-4 py-3 flex items-center justify-between gap-3">
          <Button variant="ghost" size="sm" onClick={() => navigate({ path: 'audio-detail', params: { id: audioId } })} className="gap-2">
            <ArrowLeft className="h-4 w-4" />
            Back to Transcript
          </Button>
          <div className="flex items-center gap-3">
            <div className="text-sm text-muted-foreground">Audio: {audioId}</div>
            <ThemeSwitcher />
          </div>
        </div>
      </div>

      <div className="h-[calc(100vh-49px)]">
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
