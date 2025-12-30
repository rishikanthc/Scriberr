import { useState, useEffect } from "react";
import { Input } from "@/components/ui/input";

interface DebouncedSearchInputProps {
    value: string;
    onChange: (value: string) => void;
    placeholder: string;
    className?: string; // made optional
}

export function DebouncedSearchInput({
    value,
    onChange,
    placeholder,
    className
}: DebouncedSearchInputProps) {
    const [searchValue, setSearchValue] = useState(value);

    // Update internal state when external value changes
    useEffect(() => {
        setSearchValue(value);
    }, [value]);

    // Debounce the onChange callback
    useEffect(() => {
        const timeoutId = setTimeout(() => {
            onChange(searchValue);
        }, 300);

        return () => clearTimeout(timeoutId);
    }, [searchValue, onChange]);

    return (
        <Input
            placeholder={placeholder}
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            className={`h-10 rounded-[var(--radius-btn)] border-[var(--border-subtle)] bg-[var(--bg-main)] focus:ring-[var(--brand-light)] focus:border-[var(--brand-solid)] transition-all duration-200 ${className}`}
        />
    );
}
