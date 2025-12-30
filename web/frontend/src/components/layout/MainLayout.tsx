
import React from 'react';

interface MainLayoutProps {
    children: React.ReactNode;
    header: React.ReactNode;
    className?: string;
}

export const MainLayout = ({ children, header, className = "" }: MainLayoutProps) => {
    return (
        <div className={`min-h-screen bg-[var(--bg-main)] ${className}`}>
            <div className="mx-auto w-full max-w-[960px] px-4 sm:px-6 py-6">
                <div className="mb-8 pb-6">
                    {header}
                </div>
                <main className="w-full">
                    {children}
                </main>
            </div>
        </div>
    );
};
