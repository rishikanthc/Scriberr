export function ScriberrLogo({ className = '' }: { className?: string }) {
  return (
    <img
      src="/scriberr-logo.png"
      alt="Scriberr"
      className={`w-auto select-none ${className}`}
    />
  );
}
