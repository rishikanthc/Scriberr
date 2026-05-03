export type GeneralSettings = {
  auto_transcription_enabled: boolean;
  auto_rename_enabled: boolean;
  default_profile_id: string | null;
  local_only: boolean;
  max_upload_size_mb: number;
};

export type UpdateGeneralSettingsPayload = {
  auto_transcription_enabled?: boolean;
  auto_rename_enabled?: boolean;
  default_profile_id?: string | null;
};

export type ChangePasswordPayload = {
  current_password: string;
  new_password: string;
  confirm_password: string;
};

export async function getGeneralSettings(headers?: Record<string, string>) {
  const response = await fetch("/api/v1/settings", { headers });
  if (!response.ok) throw new Error(await readError(response));
  return normalizeSettings((await response.json()) as Partial<GeneralSettings>);
}

export async function updateGeneralSettings(payload: UpdateGeneralSettingsPayload, headers?: Record<string, string>) {
  const response = await fetch("/api/v1/settings", {
    method: "PATCH",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await readError(response));
  return normalizeSettings((await response.json()) as Partial<GeneralSettings>);
}

export async function changePassword(payload: ChangePasswordPayload, headers?: Record<string, string>) {
  const response = await fetch("/api/v1/auth/change-password", {
    method: "POST",
    headers: { ...headers, "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!response.ok) throw new Error(await readError(response));
}

function normalizeSettings(settings: Partial<GeneralSettings>): GeneralSettings {
  return {
    auto_transcription_enabled: Boolean(settings.auto_transcription_enabled),
    auto_rename_enabled: Boolean(settings.auto_rename_enabled),
    default_profile_id: typeof settings.default_profile_id === "string" ? settings.default_profile_id : null,
    local_only: settings.local_only ?? true,
    max_upload_size_mb: settings.max_upload_size_mb ?? 0,
  };
}

async function readError(response: Response) {
  try {
    const data = await response.json();
    return data?.error?.message || data?.message || response.statusText;
  } catch {
    return response.statusText;
  }
}
