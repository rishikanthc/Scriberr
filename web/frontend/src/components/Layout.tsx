import { Header } from './Header'

interface LayoutProps {
    children: React.ReactNode
}

export function Layout({ children }: LayoutProps) {
    const handleFileSelect = () => {
        // Default behavior: do nothing or maybe navigate to home?
        // For now, consistent with Settings page behavior
    }

    return (
        <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
            <div className="mx-auto w-full max-w-6xl px-2 sm:px-6 md:px-8 py-3 sm:py-6">
                <Header onFileSelect={handleFileSelect} />
                <div className="mt-4 sm:mt-6">
                    {children}
                </div>
            </div>
        </div>
    )
}
