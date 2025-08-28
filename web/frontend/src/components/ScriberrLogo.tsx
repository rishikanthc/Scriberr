export function ScriberrLogo({ className = "", onClick }: { className?: string; onClick?: () => void }) {
  const clickable = typeof onClick === 'function'
  return (
    <div className={`${className}`}>
      <span
        className={`logo-font-poiret text-3xl sm:text-4xl font-normal bg-clip-text text-transparent bg-gradient-to-r from-[var(--brand-accent-start)] to-[var(--brand-accent-end)] select-none ${clickable ? 'cursor-pointer hover:opacity-90 focus:opacity-90 outline-none' : ''}`}
        aria-label="Scriberr"
        role={clickable ? 'button' as const : undefined}
        tabIndex={clickable ? 0 : undefined}
        onClick={onClick}
        onKeyDown={(e) => {
          if (!clickable) return
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onClick?.()
          }
        }}
      >
        Scriberr
      </span>
    </div>
  )
}
