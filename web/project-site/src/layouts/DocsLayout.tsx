import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Layout } from '../Layout';
import { Section } from '../components/ui/Section';
import { TableOfContents } from '../components/TableOfContents';
import { useState } from 'react';
import { Menu, X } from 'lucide-react';

interface DocsLayoutProps {
  children: React.ReactNode;
}

export function DocsLayout({ children }: DocsLayoutProps) {
  const location = useLocation();
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);

  const navItems = [
    { path: '/docs/intro', label: 'Introduction' },
    { path: '/docs/features', label: 'Features' },
    { path: '/docs/installation', label: 'Installation' },
    { path: '/docs/diarization', label: 'Diarization' },
    { path: '/docs/usage', label: 'Usage Guide' },
    { path: '/docs/troubleshooting', label: 'Troubleshooting' },
  ];

  return (
    <Layout>
      <Section className="py-0">
        <div className="flex w-full max-w-7xl mx-auto">
          {/* Mobile Sidebar Backdrop */}
          {isSidebarOpen && (
            <div
              className="fixed inset-0 bg-black/20 z-30 lg:hidden"
              onClick={() => setIsSidebarOpen(false)}
            />
          )}

          {/* Sidebar Navigation */}
          <aside className={`
            fixed top-16 bottom-0 left-0 w-64 bg-white border-r border-gray-100 z-40 transform transition-transform duration-300 lg:translate-x-0 lg:static lg:border-r-0 lg:block lg:z-auto
            ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full'}
          `}>
            <div className="h-full overflow-y-auto px-4 lg:px-0 py-6 lg:py-2 space-y-8">
              <div>
                <ul className="space-y-1">
                  {navItems.map((item) => {
                    const isActive = location.pathname === item.path;
                    return (
                      <li key={item.path}>
                        <Link
                          to={item.path}
                          onClick={() => setIsSidebarOpen(false)}
                          className={`block px-3 py-1.5 rounded-md text-sm font-[family-name:var(--font-heading)] transition-colors duration-200 ${isActive
                            ? 'text-[#FF6D20] font-medium bg-orange-50'
                            : 'text-gray-500 hover:text-gray-900 hover:bg-gray-50'
                            }`}
                        >
                          {item.label}
                        </Link>
                      </li>
                    );
                  })}
                </ul>
              </div>
            </div>
          </aside>

          {/* Main Content Area */}
          <div className="flex-1 min-w-0 px-4 sm:px-6 lg:pl-12 lg:pr-8 pt-6 lg:pt-0 pb-12 w-full">
            {/* Mobile Sidebar Toggle */}
            <div className="lg:hidden mb-6 flex items-center justify-between sticky top-0 bg-white/80 backdrop-blur-md py-4 z-20 border-b border-gray-100 -mx-4 px-4 sm:-mx-6 sm:px-6">
              <button
                onClick={() => setIsSidebarOpen(!isSidebarOpen)}
                className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-orange-50 border border-orange-100 text-[#FF6D20] font-semibold text-sm hover:bg-orange-100 transition-colors"
              >
                {isSidebarOpen ? <X className="w-4 h-4" strokeWidth={2.5} /> : <Menu className="w-4 h-4" strokeWidth={2.5} />}
                <span>Menu</span>
              </button>
              <div className="text-xs font-bold text-gray-900 uppercase tracking-wider bg-gray-100 px-2 py-1 rounded">
                {navItems.find(i => i.path === location.pathname)?.label || 'Docs'}
              </div>
            </div>

            <article className="docs-content max-w-3xl mx-auto lg:mx-0">
              {children}
            </article>
          </div>

          {/* Right Sidebar - For TOC */}
          <div className="hidden xl:block w-64 shrink-0">
            <div className="sticky top-24">
              <TableOfContents />
            </div>
          </div>
        </div>
      </Section>
    </Layout>
  );
}
