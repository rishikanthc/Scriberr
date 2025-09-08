export function ScriberrLogo({ className = "", onClick }: { className?: string; onClick?: () => void }) {
  const clickable = typeof onClick === 'function'
  return (
    <div className={`${className}`}>
      <img
        src="/scriberr-logo.png"
        alt="Scriberr"
        className={`h-8 sm:h-10 w-auto select-none ${clickable ? 'cursor-pointer hover:opacity-90 focus:opacity-90 outline-none' : ''}`}
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
      />
    </div>
  )
}
