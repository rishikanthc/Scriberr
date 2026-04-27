import type { InputHTMLAttributes, ReactNode } from "react";

type FieldProps = {
  id: string;
  label: string;
  children?: ReactNode;
} & InputHTMLAttributes<HTMLInputElement>;

export function Field({ id, label, children, ...inputProps }: FieldProps) {
  return (
    <div className="scr-field">
      <label className="scr-label" htmlFor={id}>
        {label}
      </label>
      <input id={id} className="scr-input" {...inputProps} />
      {children}
    </div>
  );
}
