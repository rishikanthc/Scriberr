import { ScriberrLogo } from './ScriberrLogo';
import { useState } from 'react';
import GithubBadge from './GithubBadge';

export default function Navbar() {
  const [open, setOpen] = useState(false);
  return (
    <header className="sticky top-0 z-40 bg-white/80 backdrop-blur shadow-soft">
      <div className="container-narrow py-4 flex items-center justify-between">
        <a href="#" className="flex items-center gap-3">
          <ScriberrLogo className="h-8 sm:h-10" />
        </a>
        <nav className="hidden md:flex items-center gap-6 text-sm text-gray-600">
          <a href="/docs/intro.html" className="hover:text-gray-900">Docs</a>
          <a href="/changelog.html" className="hover:text-gray-900">Changelog</a>
          <a href="/api.html" className="hover:text-gray-900">API</a>
        </nav>
        <div className="flex items-center gap-3">
          <button
            className="md:hidden inline-flex items-center justify-center rounded-md border border-gray-200 bg-white px-2.5 py-1.5 text-gray-700 hover:bg-gray-50"
            aria-label="Menu"
            onClick={() => setOpen((v) => !v)}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="size-5">
              <path d="M4 7h16M4 12h16M4 17h16" />
            </svg>
          </button>
          <GithubBadge />
        </div>
      </div>
      {open && (
        <div className="md:hidden border-t border-gray-200 bg-white">
          <div className="container-narrow py-3 flex flex-col gap-2 text-sm text-gray-700">
            <a href="/docs/intro.html" className="hover:text-gray-900" onClick={() => setOpen(false)}>Docs</a>
            <a href="/changelog.html" className="hover:text-gray-900" onClick={() => setOpen(false)}>Changelog</a>
            <a href="/api.html" className="hover:text-gray-900" onClick={() => setOpen(false)}>API</a>
          </div>
        </div>
      )}
    </header>
  );
}
