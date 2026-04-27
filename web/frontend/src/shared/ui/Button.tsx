import type { ButtonHTMLAttributes, ReactNode } from "react";
import { cn } from "@/lib/utils";

type AppButtonVariant = "primary" | "secondary";

type AppButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: AppButtonVariant;
  children: ReactNode;
};

const buttonInteractionClasses = [
  "cursor-pointer",
  "select-none",
  "outline-none",
  "focus-visible:ring-2",
  "focus-visible:ring-[color-mix(in_srgb,var(--scr-brand-solid)_24%,transparent)]",
  "focus-visible:ring-offset-2",
  "focus-visible:ring-offset-[var(--scr-surface-canvas)]",
  "disabled:pointer-events-none",
  "disabled:cursor-not-allowed",
].join(" ");

export function AppButton({ variant = "primary", className, children, type = "button", ...props }: AppButtonProps) {
  return (
    <button
      type={type}
      className={cn(
        "scr-button",
        buttonInteractionClasses,
        variant === "primary" ? "scr-button-primary" : "scr-button-secondary",
        className
      )}
      {...props}
    >
      {children}
    </button>
  );
}

type IconButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  label: string;
};

export function IconButton({ label, className, children, type = "button", ...props }: IconButtonProps) {
  return (
    <button
      type={type}
      aria-label={label}
      title={label}
      className={cn("scr-icon-button", buttonInteractionClasses, className)}
      {...props}
    >
      {children}
    </button>
  );
}
