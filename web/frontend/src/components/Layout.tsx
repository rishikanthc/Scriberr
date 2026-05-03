import { Link } from "react-router-dom";

interface LayoutProps {
    children: React.ReactNode
}

export function Layout({ children }: LayoutProps) {
    return (
        <div className="min-h-screen scr-app">
            <div className="mx-auto w-full max-w-6xl px-3 sm:px-6 lg:px-8 pb-12 space-y-4 sm:space-y-8">
                <header className="flex h-16 items-center justify-between border-b border-[var(--scr-border)]">
                    <Link to="/" className="text-lg font-semibold text-[var(--scr-text)]">
                        Scriberr
                    </Link>
                    <Link to="/settings" className="text-sm text-[var(--scr-muted)] hover:text-[var(--scr-text)]">
                        Settings
                    </Link>
                </header>
                <main className="animate-fade-in">
                    {children}
                </main>
            </div>
        </div>
    )
}
