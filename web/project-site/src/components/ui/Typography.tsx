import React, { type ElementType } from 'react';

interface HeadingProps {
    children: React.ReactNode;
    level?: 1 | 2 | 3 | 4 | 5 | 6;
    className?: string;
}

export function Heading({ children, level = 1, className = '' }: HeadingProps) {
    const Tag = `h${level}` as ElementType;

    const styles = {
        1: "text-5xl md:text-7xl font-bold tracking-tight text-[#1a1a1a] font-[family-name:var(--font-heading)]",
        2: "text-3xl md:text-4xl font-bold text-[#1a1a1a] font-[family-name:var(--font-heading)]",
        3: "text-2xl font-semibold text-[#1a1a1a] font-[family-name:var(--font-heading)]",
        4: "text-xl font-semibold text-[#1a1a1a] font-[family-name:var(--font-heading)]",
        5: "text-lg font-medium text-[#1a1a1a] font-[family-name:var(--font-heading)]",
        6: "text-base font-medium text-[#1a1a1a] font-[family-name:var(--font-heading)]"
    };

    return <Tag className={`${styles[level]} ${className}`}>{children}</Tag>;
}

export function Paragraph({ children, className = '', size = 'md' }: { children: React.ReactNode, className?: string, size?: 'sm' | 'md' | 'lg' }) {
    const sizes = {
        sm: "text-sm",
        md: "text-base", // 16px base, was text-lg
        lg: "text-xl"
    };
    return <p className={`text-[#333333] leading-relaxed font-[family-name:var(--font-body)] ${sizes[size]} ${className}`}>{children}</p>;
}

export function GradientText({ children, className = '' }: { children: React.ReactNode, className?: string }) {
    return (
        <span className={`text-transparent bg-clip-text bg-[image:var(--image-brand-gradient)] ${className}`}>
            {children}
        </span>
    );
}
