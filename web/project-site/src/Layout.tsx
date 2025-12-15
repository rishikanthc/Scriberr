import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { ScriberrLogo } from './components/ScriberrLogo';
import { Github, Book, Menu, X } from 'lucide-react';
import { useState } from 'react';


import { GithubBadge } from './components/GithubBadge';

interface LayoutProps {
    children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
    const location = useLocation();
    const isDocs = location.pathname.startsWith('/docs');
    const [isMenuOpen, setIsMenuOpen] = useState(false);

    return (
        <div className="min-h-screen flex flex-col font-sans selection:bg-[#FF6D20] selection:text-white">
            {/* Header */}
            <header className="fixed top-0 left-0 right-0 z-50 glass-panel transition-all duration-300">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
                    <div className="flex-shrink-0 flex items-center gap-3">
                        <Link to="/">
                            <ScriberrLogo />
                        </Link>
                        {isDocs && (
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold bg-orange-100 text-[#FF6D20] border border-orange-200 font-heading">
                                Docs
                            </span>
                        )}
                    </div>

                    <nav className="hidden md:flex items-center gap-8">
                        <Link to="/docs/intro" className="text-sm text-gray-600 hover:text-gray-900 transition-colors font-heading">Documentation</Link>
                        <Link to="/api" className="text-sm text-gray-600 hover:text-gray-900 transition-colors font-heading">API</Link>
                    </nav>

                    <div className="flex items-center gap-4">
                        <div className="hidden md:block">
                            <GithubBadge />
                        </div>
                        <button
                            onClick={() => setIsMenuOpen(!isMenuOpen)}
                            className="md:hidden flex items-center justify-center w-10 h-10 rounded-xl bg-white/80 border border-gray-200 shadow-sm text-gray-600 hover:text-[#FF6D20] hover:border-orange-200 hover:bg-orange-50 transition-all duration-200"
                            aria-label="Toggle menu"
                        >
                            {isMenuOpen ? <X className="w-5 h-5" strokeWidth={2.5} /> : <Menu className="w-5 h-5" strokeWidth={2.5} />}
                        </button>
                    </div>
                </div>

                {/* Mobile Menu Overlay */}
                {isMenuOpen && (
                    <div className="md:hidden absolute top-16 left-0 right-0 bg-white border-b border-gray-100 shadow-lg animate-fade-in z-40">
                        <nav className="flex flex-col p-4 space-y-4">
                            <Link
                                to="/docs/intro"
                                className="text-sm font-medium text-gray-600 hover:text-gray-900 py-2 border-b border-gray-50 font-heading"
                                onClick={() => setIsMenuOpen(false)}
                            >
                                Documentation
                            </Link>
                            <Link
                                to="/api"
                                className="text-sm font-medium text-gray-600 hover:text-gray-900 py-2 border-b border-gray-50 font-heading"
                                onClick={() => setIsMenuOpen(false)}
                            >
                                API
                            </Link>
                            <div className="pt-2">
                                <GithubBadge />
                            </div>
                        </nav>
                    </div>
                )}
            </header>

            {/* Main Content */}
            <main className="flex-grow pt-16">
                {children}
            </main>

            {/* Footer */}
            <footer className="border-t border-gray-100 bg-gray-50 py-6">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col md:flex-row justify-between items-center gap-6">
                    <div className="flex items-center gap-2">
                        <span className="text-sm text-gray-500">© 2025 Scriberr. All rights reserved.</span>
                    </div>
                    <div className="flex items-center gap-6">
                        <a
                            href="https://github.com/rishikanthc/scriberr"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-gray-400 hover:text-gray-600 transition-colors"
                        >
                            <Github className="w-5 h-5" />
                        </a>
                        <Link to="/docs/intro" className="text-gray-400 hover:text-gray-600 transition-colors">
                            <Book className="w-5 h-5" />
                        </Link>
                    </div>
                </div>
            </footer>
        </div>
    );
}
