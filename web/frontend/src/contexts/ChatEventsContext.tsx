import { createContext, useCallback, useContext, useMemo, useRef, type PropsWithChildren } from 'react'

type TitleUpdate = { sessionId: string; title: string }
type TitleGenerating = { sessionId: string; isGenerating: boolean }

type ChatEvents = {
  emitSessionTitleUpdated: (payload: TitleUpdate) => void
  subscribeSessionTitleUpdated: (fn: (payload: TitleUpdate) => void) => () => void
  emitTitleGenerating: (payload: TitleGenerating) => void
  subscribeTitleGenerating: (fn: (payload: TitleGenerating) => void) => () => void
}

const ChatEventsContext = createContext<ChatEvents | null>(null)

// eslint-disable-next-line @typescript-eslint/no-empty-object-type
export function ChatEventsProvider({ children }: PropsWithChildren<{}>) {
  const titleUpdateListenersRef = useRef(new Set<(p: TitleUpdate) => void>())
  const titleGeneratingListenersRef = useRef(new Set<(p: TitleGenerating) => void>())

  const emitSessionTitleUpdated = useCallback((payload: TitleUpdate) => {
    for (const l of titleUpdateListenersRef.current) l(payload)
  }, [])

  const subscribeSessionTitleUpdated = useCallback((fn: (p: TitleUpdate) => void) => {
    titleUpdateListenersRef.current.add(fn)
    return () => titleUpdateListenersRef.current.delete(fn)
  }, [])

  const emitTitleGenerating = useCallback((payload: TitleGenerating) => {
    for (const l of titleGeneratingListenersRef.current) l(payload)
  }, [])

  const subscribeTitleGenerating = useCallback((fn: (p: TitleGenerating) => void) => {
    titleGeneratingListenersRef.current.add(fn)
    return () => titleGeneratingListenersRef.current.delete(fn)
  }, [])

  const value = useMemo<ChatEvents>(() => ({
    emitSessionTitleUpdated,
    subscribeSessionTitleUpdated,
    emitTitleGenerating,
    subscribeTitleGenerating
  }), [emitSessionTitleUpdated, subscribeSessionTitleUpdated, emitTitleGenerating, subscribeTitleGenerating])

  return <ChatEventsContext.Provider value={value}>{children}</ChatEventsContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useChatEvents() {
  const ctx = useContext(ChatEventsContext)
  if (!ctx) throw new Error('useChatEvents must be used within ChatEventsProvider')
  return ctx
}
