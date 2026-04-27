import type { ButtonHTMLAttributes, ReactNode } from "react";
import { cn } from "@/lib/utils";

type AppButtonVariant = "primary" | "secondary";

type AppButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: AppButtonVariant;
  children: ReactNode;
};

export function AppButton({ variant = "primary", className, children, ...props }: AppButtonProps) {
  return (
    <button
      className={cn("scr-button", variant === "primary" ? "scr-button-primary" : "scr-button-secondary", className)}
      {...props}
    >
      {children}
    </button>
  );
}

type IconButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  label: string;
};

export function IconButton({ label, className, children, ...props }: IconButtonProps) {
  return (
    <button aria-label={label} title={label} className={cn("scr-icon-button", className)} {...props}>
      {children}
    </button>
  );
}
