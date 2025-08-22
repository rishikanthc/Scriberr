import { Homepage } from './components/Homepage'
import { AudioDetailView } from './components/AudioDetailView'
import { useRouter } from './contexts/RouterContext'

function App() {
  const { currentRoute } = useRouter()

  if (currentRoute.path === 'audio-detail' && currentRoute.params?.id) {
    return <AudioDetailView audioId={currentRoute.params.id} />
  }

  return <Homepage />
}

export default App
