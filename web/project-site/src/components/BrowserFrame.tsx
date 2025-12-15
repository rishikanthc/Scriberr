import React from 'react';

interface BrowserFrameProps {
    children: React.ReactNode;
    className?: string;
}

export function BrowserFrame({ children, className = "" }: BrowserFrameProps) {
    return (
        <div className={`rounded-xl overflow-hidden border border-gray-200 bg-white shadow-lg ${className} my-8`}>
            {/* Content */}
            <div className="relative">
                {children}
            </div>
        </div>
    );
}
