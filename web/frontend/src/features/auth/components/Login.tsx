import { useState } from "react";
import { AlertCircle, Loader2 } from "lucide-react";
import { loginUser, type AuthSession } from "@/features/auth/api/authApi";
import { AppButton } from "@/shared/ui/Button";
import { Field } from "@/shared/ui/Field";

type LoginProps = {
  onLogin: (session: AuthSession) => void;
};

export function Login({ onLogin }: LoginProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setError("");
    setLoading(true);

    try {
      const session = await loginUser(username.trim(), password);
      onLogin(session);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  };

  const isFormValid = username.trim().length > 0 && password.length > 0;

  return (
    <main className="scr-auth-page">
      <section className="scr-auth-panel" aria-labelledby="login-title">
        <div className="scr-auth-logo">
          <img src="/logo-text.svg" alt="Scriberr" />
        </div>
        <h1 id="login-title" className="scr-auth-title">
          Welcome back
        </h1>
        <p className="scr-auth-copy">Sign in to continue.</p>

        <div className="scr-card">
          <form className="scr-form" onSubmit={handleSubmit}>
            {error ? (
              <div className="scr-alert" role="alert">
                <AlertCircle size={16} aria-hidden="true" /> {error}
              </div>
            ) : null}

            <Field
              id="username"
              label="Username"
              type="text"
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              disabled={loading}
              required
              autoComplete="username"
              placeholder="Enter your username"
            />

            <Field
              id="password"
              label="Password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              disabled={loading}
              required
              autoComplete="current-password"
              placeholder="Enter your password"
            />

            <AppButton type="submit" disabled={loading || !isFormValid}>
              {loading ? <Loader2 size={16} className="animate-spin" aria-hidden="true" /> : null}
              {loading ? "Signing in" : "Sign in"}
            </AppButton>
          </form>
        </div>
      </section>
    </main>
  );
}
