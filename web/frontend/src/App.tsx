import { Homepage } from './components/Homepage'
import { AudioDetailView } from './components/AudioDetailView'
import { Settings } from './pages/Settings'
import { useRouter } from './contexts/RouterContext'

function App() {
  const { currentRoute } = useRouter()

  if (currentRoute.path === 'audio-detail' && currentRoute.params?.id) {
    return <AudioDetailView audioId={currentRoute.params.id} />
  }

  if (currentRoute.path === 'settings') {
    return <Settings />
  }

  return <Homepage />
}

export default App
