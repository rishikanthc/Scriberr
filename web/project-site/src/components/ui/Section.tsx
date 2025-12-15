
import React from 'react';

interface SectionProps {
    children: React.ReactNode;
    id?: string;
    className?: string;
    containerParams?: string;
}

export function Section({ children, id, className = '', containerParams = 'max-w-7xl' }: SectionProps) {
    return (
        <section id={id} className={`py-20 px-4 sm:px-6 lg:px-8 ${className}`}>
            <div className={`${containerParams} mx-auto`}>
                {children}
            </div>
        </section>
    );
}
