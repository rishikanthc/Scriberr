import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Layout } from '../Layout';
import { Section } from '../components/ui/Section';
import { TableOfContents } from '../components/TableOfContents';

interface DocsLayoutProps {
  children: React.ReactNode;
}

export function DocsLayout({ children }: DocsLayoutProps) {
  const location = useLocation();

  const navItems = [
    { path: '/docs/intro', label: 'Introduction' },
    { path: '/docs/features', label: 'Features' },
    { path: '/docs/installation', label: 'Installation' },
    { path: '/docs/configuration', label: 'Configuration' },
    { path: '/docs/usage', label: 'Usage Guide' },
  ];

  return (
    <Layout>
      <Section className="py-0">
        <div className="flex w-full max-w-7xl mx-auto">
          {/* Sidebar Navigation - Fixed & Sticky */}
          <aside className="hidden lg:block w-64 shrink-0 fixed top-24 bottom-0 overflow-y-auto pr-8">
            <div className="py-2 space-y-8">
              <div>
                <ul className="space-y-1">
                  {navItems.map((item) => {
                    const isActive = location.pathname === item.path;
                    return (
                      <li key={item.path}>
                        <Link
                          to={item.path}
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

          {/* Main Content Area - Fluid with max-width - Offset for fixed sidebar */}
          <div className="flex-1 min-w-0 lg:pl-72 lg:pr-8 pt-0 pb-12">
            <article className="docs-content max-w-3xl">
              {children}
            </article>
          </div>

          {/* Right Sidebar - For TOC */}
          <div className="hidden xl:block shrink-0">
            <TableOfContents />
          </div>
        </div>
      </Section>
    </Layout>
  );
}
