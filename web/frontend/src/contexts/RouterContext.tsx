
import { createContext, useContext, useEffect, useState } from 'react'

export type Route = {
  path: 'home' | 'audio-detail' | 'settings' | 'settings-cli' | 'chat' | 'auth-cli-authorize'
  params?: Record<string, string | undefined>
}

interface RouterContextType {
  currentRoute: Route
  navigate: (route: Route) => void
}

const RouterContext = createContext<RouterContextType | undefined>(undefined)

export function RouterProvider({ children }: { children: React.ReactNode }) {
  const [currentRoute, setCurrentRoute] = useState<Route>(() => {
    // Parse initial URL
    const path = window.location.pathname
    const search = window.location.search

    // /audio/<audioId>/chat/<chatSessionId>
    const chatMatch = path.match(/^\/audio\/([^\/]+)\/chat\/(.+)$/)
    if (chatMatch) {
      return { path: 'chat', params: { audioId: chatMatch[1], sessionId: chatMatch[2] } }
    }

    // /audio/<audioId>/chat (no session specified)
    const chatBase = path.match(/^\/audio\/([^\/]+)\/chat\/?$/)
    if (chatBase) {
      return { path: 'chat', params: { audioId: chatBase[1] } }
    }

    // /audio/<audioId>
    if (path.startsWith('/audio/')) {
      const audioId = path.split('/audio/')[1]
      return { path: 'audio-detail', params: { id: audioId } }
    } else if (path === '/settings/cli') {
      return { path: 'settings-cli' }
    } else if (path === '/settings') {
      return { path: 'settings' }
    } else if (path === '/auth/cli/authorize') {
      const params = new URLSearchParams(search)
      return {
        path: 'auth-cli-authorize',
        params: {
          callback_url: params.get('callback_url') || undefined,
          device_name: params.get('device_name') || undefined
        }
      }
    }

    return { path: 'home' }
  })

  const navigate = (route: Route) => {
    setCurrentRoute(route)

    // Update browser URL
    let url = '/'
    if (route.path === 'audio-detail' && route.params?.id) {
      url = `/ audio / ${route.params.id} `
    } else if (route.path === 'chat' && route.params?.audioId && route.params?.sessionId) {
      url = `/ audio / ${route.params.audioId} /chat/${route.params.sessionId} `
    } else if (route.path === 'chat' && route.params?.audioId) {
      url = `/ audio / ${route.params.audioId}/chat`
    } else if (route.path === 'settings') {
      url = '/settings'
    } else if (route.path === 'settings-cli') {
      url = '/settings/cli'
    } else if (route.path === 'auth-cli-authorize') {
      url = '/auth/cli/authorize'
      if (route.params) {
        const searchParams = new URLSearchParams()
        if (route.params.callback_url) searchParams.set('callback_url', route.params.callback_url)
        if (route.params.device_name) searchParams.set('device_name', route.params.device_name)
        const search = searchParams.toString()
        if (search) url += `?${search}`
      }
    }

    window.history.pushState({ route }, '', url)
  }

  useEffect(() => {
    const handlePopState = (event: PopStateEvent) => {
      if (event.state?.route) {
        setCurrentRoute(event.state.route)
      } else {
        // Fallback to parsing URL
        const path = window.location.pathname
        const search = window.location.search

        const chatMatch = path.match(/^\/audio\/([^\/]+)\/chat\/(.+)$/)
        if (chatMatch) {
          setCurrentRoute({ path: 'chat', params: { audioId: chatMatch[1], sessionId: chatMatch[2] } })
        } else {
          const chatBase = path.match(/^\/audio\/([^\/]+)\/chat\/?$/)
          if (chatBase) {
            setCurrentRoute({ path: 'chat', params: { audioId: chatBase[1] } })
            return
          } else if (path.startsWith('/audio/')) {
            const audioId = path.split('/audio/')[1]
            setCurrentRoute({ path: 'audio-detail', params: { id: audioId } })
          } else if (path === '/settings/cli') {
            setCurrentRoute({ path: 'settings-cli' })
          } else if (path === '/settings') {
            setCurrentRoute({ path: 'settings' })
          } else if (path === '/auth/cli/authorize') {
            const params = new URLSearchParams(search)
            setCurrentRoute({
              path: 'auth-cli-authorize',
              params: {
                callback_url: params.get('callback_url') || undefined,
                device_name: params.get('device_name') || undefined
              }
            })
          } else {
            setCurrentRoute({ path: 'home' })
          }
        }
      }
    }

    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  return (
    <RouterContext.Provider value={{ currentRoute, navigate }}>
      {children}
    </RouterContext.Provider>
  )
}

export function useRouter() {
  const context = useContext(RouterContext)
  if (context === undefined) {
    throw new Error('useRouter must be used within a RouterProvider')
  }
  return context
}
