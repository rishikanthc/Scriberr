import { ReactNode, useState } from 'react';
import GithubBadge from './GithubBadge';

type DocsLayoutProps = {
  active?: 'intro' | 'installation' | 'diarization' | 'contributing';
  children: ReactNode;
};

export default function DocsLayout({ active = 'intro', children }: DocsLayoutProps) {
  const [mobileOpen, setMobileOpen] = useState(false);
  return (
    <div className="min-h-screen bg-white">
      <header className="api-topbar">
        <div className="container-narrow py-3 flex items-center justify-between gap-3">
          <a href="/" className="flex items-center gap-2 select-none min-w-0">
            <span className="logo-font-poiret text-lg text-gray-900">Scriberr</span>
            <span className="text-gray-300">/</span>
            <span className="text-sm text-gray-600">Docs</span>
          </a>
          <div className="flex items-center gap-2">
            <button
              className="md:hidden inline-flex items-center justify-center rounded-md border border-gray-200 bg-white px-2.5 py-1.5 text-gray-700 hover:bg-gray-50"
              aria-label="Toggle sidebar"
              onClick={() => setMobileOpen((v) => !v)}
            >
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" className="size-5">
                <path d="M4 7h16M4 12h16M4 17h16" />
              </svg>
            </button>
            <div className="hidden md:block"><GithubBadge /></div>
          </div>
        </div>
      </header>

      <main className="container-narrow py-10">
        {mobileOpen && (
          <div className="md:hidden mb-4 border border-gray-200 rounded-lg p-3">
            <MobileNav active={active} onClick={() => setMobileOpen(false)} />
          </div>
        )}
        <div className="grid grid-cols-1 md:grid-cols-[240px_minmax(0,1fr)] gap-8">
          <aside className="api-sidebar">
            <div className="sticky top-24 pr-6">
              <div className="text-[11px] font-medium text-gray-500 mb-2">Docs</div>
              <nav className="text-sm">
                <ul className="space-y-2">
                  <li>
                    <a href="/docs/intro.html" className={linkCls(active === 'intro')}>Introduction</a>
                  </li>
                  <li>
                    <a href="/docs/installation.html" className={linkCls(active === 'installation')}>Installation</a>
                  </li>
                  <li>
                    <a href="/docs/diarization.html" className={linkCls(active === 'diarization')}>Diarization</a>
                  </li>
                  <li>
                    <a href="/docs/contributing.html" className={linkCls(active === 'contributing')}>Contributing</a>
                  </li>
                </ul>
              </nav>
            </div>
          </aside>

          <section className="space-y-8 docs-prose">
            {children}
          </section>
        </div>
      </main>
    </div>
  );
}

function linkCls(active: boolean) {
  return `block rounded px-2 py-1 ${active ? 'bg-gray-100 text-gray-900' : 'text-gray-600 hover:text-gray-900'}`;
}

function MobileNav({ active, onClick }: { active?: DocsLayoutProps['active']; onClick?: () => void }) {
  return (
    <nav className="text-sm">
      <ul className="grid grid-cols-2 gap-2">
        <li>
          <a href="/docs/intro.html" className={linkCls(active === 'intro')} onClick={onClick}>Introduction</a>
        </li>
        <li>
          <a href="/docs/installation.html" className={linkCls(active === 'installation')} onClick={onClick}>Installation</a>
        </li>
        <li>
          <a href="/docs/diarization.html" className={linkCls(active === 'diarization')} onClick={onClick}>Diarization</a>
        </li>
        <li className="col-span-2">
          <a href="/docs/contributing.html" className={linkCls(active === 'contributing')} onClick={onClick}>Contributing</a>
        </li>
      </ul>
    </nav>
  );
}
