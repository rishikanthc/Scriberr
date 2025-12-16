import { createContext, useCallback, useContext, useMemo, useState, type PropsWithChildren } from 'react'

type Toast = {
  id: number
  title: string
  description?: string
}

type ToastContextValue = {
  toast: (t: Omit<Toast, 'id'>) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function ToastProvider({ children }: PropsWithChildren) {
  const [toasts, setToasts] = useState<Toast[]>([])

  const toast = useCallback((t: Omit<Toast, 'id'>) => {
    const id = Date.now() + Math.random()
    const toastItem: Toast = { id, ...t }
    setToasts(prev => [...prev, toastItem])
    setTimeout(() => {
      setToasts(prev => prev.filter(x => x.id !== id))
    }, 2600)
  }, [])

  const value = useMemo(() => ({ toast }), [toast])

  return (
    <ToastContext.Provider value={value}>
      {children}
      {/* Toaster container */}
      <div className="pointer-events-none fixed bottom-4 right-4 z-[60] flex flex-col gap-2">
        {toasts.map(t => (
          <div
            key={t.id}
            className="pointer-events-auto min-w-[220px] max-w-[360px] rounded-md bg-carbon-900 text-white shadow-lg ring-1 ring-black/10 dark:bg-carbon-800 dark:text-carbon-100 transition-all"
          >
            <div className="px-3 py-2">
              <div className="text-sm font-medium">{t.title}</div>
              {t.description && (
                <div className="text-xs text-carbon-300 dark:text-carbon-300 mt-0.5">{t.description}</div>
              )}
            </div>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useToast() {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within ToastProvider')
  return ctx
}

