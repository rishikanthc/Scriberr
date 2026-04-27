import { useMemo, useState } from "react";
import { AlertCircle, Check, Loader2, X } from "lucide-react";
import { registerUser, type AuthSession } from "@/features/auth/api/authApi";
import { AppButton } from "@/shared/ui/Button";
import { Field } from "@/shared/ui/Field";

type RegisterProps = {
  onRegister: (session: AuthSession) => void;
};

type PasswordRule = {
  label: string;
  met: boolean;
};

export function Register({ onRegister }: RegisterProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const passwordRules = useMemo<PasswordRule[]>(
    () => [
      { label: "At least 8 characters", met: password.length >= 8 },
      { label: "One uppercase letter", met: /[A-Z]/.test(password) },
      { label: "One lowercase letter", met: /[a-z]/.test(password) },
      { label: "One number", met: /\d/.test(password) },
      { label: "One special character", met: /[!@#$%^&*(),.?":{}|<>]/.test(password) },
    ],
    [password],
  );

  const passwordIsValid = passwordRules.every((rule) => rule.met);
  const passwordsMatch = password.length > 0 && password === confirmPassword;
  const formIsValid = username.trim().length >= 3 && passwordIsValid && passwordsMatch;

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError("");

    if (!formIsValid) {
      setError("Check the username and password requirements.");
      return;
    }

    setLoading(true);
    try {
      const session = await registerUser(username.trim(), password, confirmPassword);
      onRegister(session);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="scr-auth-page">
      <section className="scr-auth-panel" aria-labelledby="register-title">
        <div className="scr-auth-logo">
          <img src="/logo-text.svg" alt="Scriberr" />
        </div>
        <h1 id="register-title" className="scr-auth-title">
          Set up Scriberr
        </h1>
        <p className="scr-auth-copy">Create the first account for this instance.</p>

        <div className="scr-card">
          <form className="scr-form" onSubmit={handleSubmit}>
            {error ? (
              <div className="scr-alert" role="alert">
                <AlertCircle size={16} aria-hidden="true" /> {error}
              </div>
            ) : null}

            <Field
              id="register-username"
              label="Username"
              type="text"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              disabled={loading}
              required
              minLength={3}
              maxLength={50}
              autoComplete="username"
              placeholder="Enter a username"
            />

            <Field
              id="register-password"
              label="Password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              disabled={loading}
              required
              autoComplete="new-password"
              placeholder="Enter a password"
            >
              {password ? (
                <div className="scr-password-rules">
                  {passwordRules.map((rule) => (
                    <div className="scr-rule" data-met={rule.met} key={rule.label}>
                      {rule.met ? <Check size={14} aria-hidden="true" /> : <X size={14} aria-hidden="true" />}
                      <span>{rule.label}</span>
                    </div>
                  ))}
                </div>
              ) : null}
            </Field>

            <Field
              id="register-confirm-password"
              label="Confirm Password"
              type="password"
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
              disabled={loading}
              required
              autoComplete="new-password"
              placeholder="Re-enter your password"
            >
              {confirmPassword ? (
                <div className="scr-rule" data-met={passwordsMatch}>
                  {passwordsMatch ? <Check size={14} aria-hidden="true" /> : <X size={14} aria-hidden="true" />}
                  <span>{passwordsMatch ? "Passwords match" : "Passwords do not match"}</span>
                </div>
              ) : null}
            </Field>

            <AppButton type="submit" disabled={loading || !formIsValid}>
              {loading ? <Loader2 size={16} className="animate-spin" aria-hidden="true" /> : null}
              {loading ? "Creating account" : "Create account"}
            </AppButton>
          </form>
        </div>
      </section>
    </main>
  );
}
