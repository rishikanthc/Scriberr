import type { ReactNode } from "react";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Slider } from "@/components/ui/slider";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Accordion,
    AccordionContent,
    AccordionItem,
    AccordionTrigger,
} from "@/components/ui/accordion";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { Info } from "lucide-react";

// ============================================================================
// Shared CSS class constants for form inputs
// ============================================================================

export const inputClassName = `
  h-11 bg-[var(--bg-main)] border border-[var(--border-subtle)] rounded-xl
  text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)]
  focus:border-[var(--brand-solid)] focus:ring-2 focus:ring-[var(--brand-solid)]/20
  transition-all duration-200
  [color-scheme:light] dark:[color-scheme:dark]
`;

const selectTriggerClassName = `
  h-11 bg-[var(--bg-main)] border border-[var(--border-subtle)] rounded-xl
  text-[var(--text-primary)] shadow-none
  focus:border-[var(--brand-solid)] focus:ring-2 focus:ring-[var(--brand-solid)]/20
`;

const selectContentClassName = `
  bg-[var(--bg-card)] border border-[var(--border-subtle)] rounded-xl
`;

const selectItemClassName = `
  text-[var(--text-primary)] rounded-lg mx-1 cursor-pointer
  focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)]
`;

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

// ============================================================================
// Composite Field Helpers
// ============================================================================

/**
 * SelectField - FormField + Select boilerplate in one component.
 * Accepts either {value, label}[] objects or plain string[] arrays.
 */
export function SelectField({ label, description, optional, value, onValueChange, options }: {
    label: string;
    description?: string;
    optional?: boolean;
    value: string;
    onValueChange: (value: string) => void;
    options: readonly { value: string; label: string }[] | string[];
}) {
    return (
        <FormField label={label} description={description} optional={optional}>
            <Select value={value} onValueChange={onValueChange}>
                <SelectTrigger className={selectTriggerClassName}>
                    <SelectValue />
                </SelectTrigger>
                <SelectContent className={selectContentClassName}>
                    {options.map((opt) => {
                        const v = typeof opt === 'string' ? opt : opt.value;
                        const l = typeof opt === 'string' ? opt : opt.label;
                        return <SelectItem key={v} value={v} className={selectItemClassName}>{l}</SelectItem>;
                    })}
                </SelectContent>
            </Select>
        </FormField>
    );
}

/**
 * SwitchField - Switch toggle with label in a consistent layout.
 */
export function SwitchField({ id, label, checked, onCheckedChange }: {
    id: string;
    label: string;
    checked: boolean;
    onCheckedChange: (checked: boolean) => void;
}) {
    return (
        <div className="flex items-center gap-3">
            <Switch id={id} checked={checked} onCheckedChange={onCheckedChange} />
            <label htmlFor={id} className="text-sm text-[var(--text-primary)] cursor-pointer">
                {label}
            </label>
        </div>
    );
}

/**
 * SliderField - FormField + Slider with min/max range labels and current value display.
 */
export function SliderField({ label, value, onValueChange, min, max, step }: {
    label: string;
    value: number;
    onValueChange: (value: number) => void;
    min: number;
    max: number;
    step: number;
}) {
    return (
        <div className="space-y-3">
            <FormField label={label}>
                <Slider
                    value={[value]}
                    onValueChange={(v) => onValueChange(v[0])}
                    min={min}
                    max={max}
                    step={step}
                    className="w-full"
                />
                <div className="flex justify-between text-xs text-[var(--text-tertiary)]">
                    <span>{min}</span>
                    <span className="font-medium text-[var(--text-primary)]">{value}</span>
                    <span>{max}</span>
                </div>
            </FormField>
        </div>
    );
}

/**
 * AdvancedAccordion - Collapsible "Advanced Settings" section.
 */
export function AdvancedAccordion({ children }: { children: ReactNode }) {
    return (
        <Accordion type="single" collapsible className="w-full">
            <AccordionItem value="advanced" className="border border-[var(--border-subtle)] rounded-xl px-4">
                <AccordionTrigger className="text-sm font-medium text-[var(--text-primary)] hover:no-underline py-4">
                    Advanced Settings
                </AccordionTrigger>
                <AccordionContent className="pb-4 space-y-4">
                    {children}
                </AccordionContent>
            </AccordionItem>
        </Accordion>
    );
}
