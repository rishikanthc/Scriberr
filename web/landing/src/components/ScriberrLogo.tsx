export function ScriberrLogo({ className = '' }: { className?: string }) {
  return (
    <span
      className={
        `logo-font-poiret text-2xl sm:text-3xl bg-clip-text text-transparent bg-gradient-to-r from-blue-600 to-purple-400 select-none ${className}`
      }
      aria-label="Scriberr"
    >
      Scriberr
    </span>
  );
}
