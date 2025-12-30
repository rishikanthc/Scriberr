import type { ReactNode } from "react";
import { Label } from "@/components/ui/label";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Info } from "lucide-react";

interface FormFieldProps {
    label: string;
    htmlFor?: string;
    description?: string;
    optional?: boolean;
    children: ReactNode;
}

/**
 * FormField - A consistent wrapper for form inputs with optional tooltip description
 * Follows Scriberr design system
 */
export function FormField({ label, htmlFor, description, optional, children }: FormFieldProps) {
    return (
        <div className="space-y-2">
            <div className="flex items-center gap-2">
                <Label
                    htmlFor={htmlFor}
                    className="text-sm font-medium text-[var(--text-primary)]"
                >
                    {label}
                    {optional && (
                        <span className="ml-1 text-[var(--text-tertiary)] font-normal">(optional)</span>
                    )}
                </Label>
                {description && (
                    <HoverCard>
                        <HoverCardTrigger asChild>
                            <Info className="h-4 w-4 text-[var(--text-tertiary)] cursor-help hover:text-[var(--text-secondary)] transition-colors" />
                        </HoverCardTrigger>
                        <HoverCardContent
                            className="w-80 bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-xl p-4"
                            style={{ boxShadow: 'var(--shadow-float)' }}
                        >
                            <p className="text-sm text-[var(--text-secondary)] leading-relaxed">{description}</p>
                        </HoverCardContent>
                    </HoverCard>
                )}
            </div>
            {children}
        </div>
    );
}

interface SectionProps {
    title: string;
    description?: string;
    children: ReactNode;
    className?: string;
}

/**
 * Section - A grouped section with title and optional description
 */
export function Section({ title, description, children, className = "" }: SectionProps) {
    return (
        <div className={`space-y-4 ${className}`}>
            <div>
                <h3 className="text-base font-semibold text-[var(--text-primary)]">{title}</h3>
                {description && (
                    <p className="text-sm text-[var(--text-secondary)] mt-1">{description}</p>
                )}
            </div>
            {children}
        </div>
    );
}

interface InfoBannerProps {
    variant: 'info' | 'warning' | 'success';
    title: string;
    children: ReactNode;
}

/**
 * InfoBanner - Alert/notice banner with consistent styling
 */
export function InfoBanner({ variant, title, children }: InfoBannerProps) {
    const styles = {
        info: {
            bg: 'bg-[var(--brand-light)]',
            border: 'border-[var(--brand-solid)]/20',
            icon: 'text-[var(--brand-solid)]',
            title: 'text-[var(--text-primary)]',
        },
        warning: {
            bg: 'bg-[var(--warning-translucent)]',
            border: 'border-[var(--warning-solid)]/20',
            icon: 'text-[var(--warning-solid)]',
            title: 'text-[var(--text-primary)]',
        },
        success: {
            bg: 'bg-[var(--success-translucent)]',
            border: 'border-[var(--success-solid)]/20',
            icon: 'text-[var(--success-solid)]',
            title: 'text-[var(--text-primary)]',
        },
    };

    const s = styles[variant];
    const icon = variant === 'warning' ? '⚠️' : variant === 'success' ? '✓' : 'ℹ️';

    return (
        <div className={`p-4 rounded-xl border ${s.bg} ${s.border}`}>
            <div className="flex items-start gap-3">
                <span className={`mt-0.5 ${s.icon}`}>{icon}</span>
                <div>
                    <h4 className={`text-sm font-medium ${s.title} mb-1`}>{title}</h4>
                    <div className="text-sm text-[var(--text-secondary)]">{children}</div>
                </div>
            </div>
        </div>
    );
}
