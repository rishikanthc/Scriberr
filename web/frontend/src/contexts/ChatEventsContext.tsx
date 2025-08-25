import { createContext, useCallback, useContext, useMemo, useRef, type PropsWithChildren } from 'react'

type TitleUpdate = { sessionId: string; title: string }

type ChatEvents = {
  emitSessionTitleUpdated: (payload: TitleUpdate) => void
  subscribeSessionTitleUpdated: (fn: (payload: TitleUpdate) => void) => () => void
}

const ChatEventsContext = createContext<ChatEvents | null>(null)

export function ChatEventsProvider({ children }: PropsWithChildren<{}>) {
  const listenersRef = useRef(new Set<(p: TitleUpdate) => void>())

  const emitSessionTitleUpdated = useCallback((payload: TitleUpdate) => {
    for (const l of listenersRef.current) l(payload)
  }, [])

  const subscribeSessionTitleUpdated = useCallback((fn: (p: TitleUpdate) => void) => {
    listenersRef.current.add(fn)
    return () => listenersRef.current.delete(fn)
  }, [])

  const value = useMemo<ChatEvents>(() => ({ emitSessionTitleUpdated, subscribeSessionTitleUpdated }), [emitSessionTitleUpdated, subscribeSessionTitleUpdated])

  return <ChatEventsContext.Provider value={value}>{children}</ChatEventsContext.Provider>
}

export function useChatEvents() {
  const ctx = useContext(ChatEventsContext)
  if (!ctx) throw new Error('useChatEvents must be used within ChatEventsProvider')
  return ctx
}
