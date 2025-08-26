export function ScriberrLogo({ className = "" }: { className?: string }) {
  return (
    <div className={`${className}`}>
      <span
        className="logo-font-poiret text-3xl sm:text-4xl font-normal bg-clip-text text-transparent bg-gradient-to-r from-[var(--brand-accent-start)] to-[var(--brand-accent-end)] select-none"
        aria-label="Scriberr"
      >
        Scriberr
      </span>
    </div>
  )
}
