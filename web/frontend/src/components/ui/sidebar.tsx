import { createContext, useContext, useMemo, useState, type PropsWithChildren } from 'react'

type SidebarContextValue = {
  open: boolean
  setOpen: (v: boolean) => void
  toggle: () => void
  width: number
}

const SidebarContext = createContext<SidebarContextValue | null>(null)

export function SidebarProvider({ children, defaultOpen = true, width = 320 }: PropsWithChildren<{ defaultOpen?: boolean; width?: number }>) {
  const [open, setOpen] = useState<boolean>(defaultOpen)
  const value = useMemo(() => ({ open, setOpen, toggle: () => setOpen(v => !v), width }), [open, width])
  return <SidebarContext.Provider value={value}>{children}</SidebarContext.Provider>
}

// eslint-disable-next-line react-refresh/only-export-components
export function useSidebar() {
  const ctx = useContext(SidebarContext)
  if (!ctx) throw new Error('useSidebar must be used within SidebarProvider')
  return ctx
}

export function Sidebar({ children, topOffset = 56, className = '' }: PropsWithChildren<{ topOffset?: number; className?: string }>) {
  const { open, width } = useSidebar()
  return (
    <div
      className={`fixed left-0 z-30 bg-sidebar text-sidebar-foreground shadow-md ${className}`}
      style={{
        top: topOffset,
        bottom: 0,
        width,
        transform: open ? 'translateX(0)' : 'translateX(-100%)',
        transition: 'transform 150ms ease-in-out',
      }}
    >
      <div className="h-full flex flex-col overflow-hidden">{children}</div>
    </div>
  )
}

export function SidebarInset({ children, className = '' }: PropsWithChildren<{ className?: string }>) {
  const { open, width } = useSidebar()
  return (
    <div
      className={className}
      style={{ marginLeft: open ? width : 0, transition: 'margin-left 150ms ease-in-out' }}
    >
      {children}
    </div>
  )
}

export function SidebarTrigger({ children, className = '', onClick }: PropsWithChildren<{ className?: string; onClick?: () => void }>) {
  const { toggle } = useSidebar()
  return (
    <button
      type="button"
      onClick={() => {
        toggle()
        onClick?.()
      }}
      className={`inline-flex items-center justify-center rounded-md border border-input bg-background px-2 py-1 text-sm hover:bg-accent hover:text-accent-foreground ${className}`}
    >
      {children}
    </button>
  )
}
