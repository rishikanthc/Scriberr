import { lazy, Suspense } from 'react'
import { useRouter } from './contexts/RouterContext'

// Lazy load route components for better performance
const Homepage = lazy(() => import('./components/Homepage').then(module => ({ default: module.Homepage })))
const AudioDetailView = lazy(() => import('./components/AudioDetailView').then(module => ({ default: module.AudioDetailView })))
const Settings = lazy(() => import('./pages/Settings').then(module => ({ default: module.Settings })))
const ChatPage = lazy(() => import('./pages/ChatPage').then(module => ({ default: module.ChatPage })))

// Loading component
const PageLoader = () => (
  <div className="flex items-center justify-center min-h-screen">
    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
  </div>
)

function App() {
  const { currentRoute } = useRouter()

  return (
    <Suspense fallback={<PageLoader />}>
      {currentRoute.path === 'audio-detail' && currentRoute.params?.id ? (
        <AudioDetailView audioId={currentRoute.params.id} />
      ) : currentRoute.path === 'settings' ? (
        <Settings />
      ) : currentRoute.path === 'chat' ? (
        <ChatPage />
      ) : (
        <Homepage />
      )}
    </Suspense>
  )
}

export default App
