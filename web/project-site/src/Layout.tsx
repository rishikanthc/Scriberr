import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { ScriberrLogo } from './components/ScriberrLogo';
import { Github, Book, Code } from 'lucide-react';
import { Button } from './components/ui/Button';

interface LayoutProps {
    children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
    const location = useLocation();
    const isDocs = location.pathname.startsWith('/docs');

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
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold bg-orange-100 text-[#FF6D20] border border-orange-200">
                                Docs
                            </span>
                        )}
                    </div>

                    <nav className="hidden md:flex items-center gap-8">
                        <Link to="/#features" className="text-sm font-medium text-gray-600 hover:text-gray-900 transition-colors">Features</Link>
                        <Link to="/docs/intro" className="text-sm font-medium text-gray-600 hover:text-gray-900 transition-colors">Documentation</Link>
                        <a href="#api" className="text-sm font-medium text-gray-600 hover:text-gray-900 transition-colors">API</a>
                    </nav>

                    <div className="flex items-center gap-4">
                        <a
                            href="https://github.com/rishikanthc/Scriberr"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="p-2 text-gray-500 hover:text-gray-900 transition-colors"
                            aria-label="GitHub"
                        >
                            <Github className="w-5 h-5" />
                        </a>
                        <div className="hidden sm:block">
                            <Button variant="primary" size="sm">Get Started</Button>
                        </div>
                    </div>
                </div>
            </header>

            {/* Main Content */}
            <main className="flex-grow pt-16">
                {children}
            </main>

            {/* Footer */}
            <footer className="border-t border-gray-100 bg-gray-50 py-12">
                <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col md:flex-row justify-between items-center gap-6">
                    <div className="flex items-center gap-2">
                        <span className="text-sm text-gray-500">© 2025 Scriberr. All rights reserved.</span>
                    </div>
                    <div className="flex items-center gap-6">
                        <a href="#" className="text-gray-400 hover:text-gray-600 transition-colors">
                            <Github className="w-5 h-5" />
                        </a>
                        <Link to="/docs" className="text-gray-400 hover:text-gray-600 transition-colors">
                            <Book className="w-5 h-5" />
                        </Link>
                        <a href="#" className="text-gray-400 hover:text-gray-600 transition-colors">
                            <Code className="w-5 h-5" />
                        </a>
                    </div>
                </div>
            </footer>
        </div>
    );
}
