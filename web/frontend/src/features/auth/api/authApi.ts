export type AuthUser = {
  id: string;
  username: string;
};

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  user: AuthUser;
};

type TokenResponse = {
  access_token?: string;
  refresh_token?: string;
  user?: AuthUser;
};

type ApiErrorResponse = {
  error?: {
    message?: string;
  } | string;
};

async function parseError(response: Response, fallback: string): Promise<string> {
  try {
    const body = (await response.json()) as ApiErrorResponse;
    if (typeof body.error === "string") return body.error;
    if (body.error?.message) return body.error.message;
  } catch {
    return fallback;
  }
  return fallback;
}

function normalizeSession(data: TokenResponse): AuthSession {
  if (!data.access_token || !data.refresh_token || !data.user) {
    throw new Error("Authentication response was incomplete");
  }
  return {
    accessToken: data.access_token,
    refreshToken: data.refresh_token,
    user: data.user,
  };
}

export async function getRegistrationStatus(): Promise<boolean> {
  const response = await fetch("/api/v1/auth/registration-status");
  if (!response.ok) {
    throw new Error(await parseError(response, "Could not read registration status"));
  }
  const data = (await response.json()) as { registration_enabled?: boolean; requiresRegistration?: boolean };
  return typeof data.registration_enabled === "boolean" ? data.registration_enabled : !!data.requiresRegistration;
}

export async function loginUser(username: string, password: string): Promise<AuthSession> {
  const response = await fetch("/api/v1/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });
  if (!response.ok) {
    throw new Error(await parseError(response, "Login failed"));
  }
  return normalizeSession((await response.json()) as TokenResponse);
}

export async function registerUser(username: string, password: string, confirmPassword: string): Promise<AuthSession> {
  const response = await fetch("/api/v1/auth/register", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      username,
      password,
      confirm_password: confirmPassword,
    }),
  });
  if (!response.ok) {
    throw new Error(await parseError(response, "Registration failed"));
  }
  return normalizeSession((await response.json()) as TokenResponse);
}

export async function refreshSession(refreshToken: string): Promise<AuthSession> {
  const originalFetch = window.__scriberr_original_fetch || window.fetch;
  const response = await originalFetch("/api/v1/auth/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
  if (!response.ok) {
    throw new Error(await parseError(response, "Session refresh failed"));
  }
  return normalizeSession((await response.json()) as TokenResponse);
}

export async function logoutSession(refreshToken: string | null): Promise<void> {
  if (!refreshToken) return;
  await fetch("/api/v1/auth/logout", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
}

export async function getCurrentUser(accessToken: string): Promise<AuthUser> {
  const originalFetch = window.__scriberr_original_fetch || window.fetch;
  const response = await originalFetch("/api/v1/auth/me", {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  if (!response.ok) {
    throw new Error(await parseError(response, "Session is no longer valid"));
  }
  return (await response.json()) as AuthUser;
}
