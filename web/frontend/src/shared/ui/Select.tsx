import { useEffect, useId, useRef, useState } from "react";
import { Check, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";

export type SelectOption = {
  value: string;
  label: string;
  description?: string;
};

type SelectProps = {
  label?: string;
  value: string;
  options: SelectOption[];
  onChange: (value: string) => void;
  className?: string;
  disabled?: boolean;
};

export function Select({ label, value, options, onChange, className, disabled = false }: SelectProps) {
  const [open, setOpen] = useState(false);
  const id = useId();
  const containerRef = useRef<HTMLDivElement>(null);
  const selected = options.find((option) => option.value === value) || options[0];

  useEffect(() => {
    if (!open) return;
    const closeOnOutside = (event: PointerEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
      }
    };
    document.addEventListener("pointerdown", closeOnOutside);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("pointerdown", closeOnOutside);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [open]);

  const choose = (nextValue: string) => {
    if (disabled) return;
    onChange(nextValue);
    setOpen(false);
  };

  return (
    <div className={cn("scr-select-field", className)} ref={containerRef}>
      {label ? <span className="scr-select-label">{label}</span> : null}
      <button
        id={id}
        className="scr-select-trigger"
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        disabled={disabled}
        onClick={() => setOpen((current) => !current)}
      >
        <span className="scr-select-value">
          <span>{selected?.label || "Select"}</span>
          {selected?.description ? <small>{selected.description}</small> : null}
        </span>
        <ChevronDown className="scr-select-chevron" size={16} aria-hidden="true" />
      </button>

      {open ? (
        <div className="scr-select-menu" role="listbox" aria-labelledby={id}>
          {options.map((option) => {
            const active = option.value === value;
            return (
              <button
                key={option.value}
                className="scr-select-option"
                data-active={active}
                type="button"
                role="option"
                aria-selected={active}
                onClick={() => choose(option.value)}
              >
                <span>
                  <span>{option.label}</span>
                  {option.description ? <small>{option.description}</small> : null}
                </span>
                {active ? <Check size={16} aria-hidden="true" /> : null}
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}
