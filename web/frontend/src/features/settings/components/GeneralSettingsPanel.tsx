import { useMemo, useState } from "react";
import type { FormEvent, ReactNode } from "react";
import { Check, Eye, EyeOff, Lock, WandSparkles, Workflow, X } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { AppButton, IconButton } from "@/shared/ui/Button";
import { useAuth } from "@/features/auth/hooks/useAuth";
import type { UpdateGeneralSettingsPayload } from "@/features/settings/api/generalSettingsApi";
import { useLLMProviderSettings } from "@/features/settings/hooks/useLLMProvider";
import { useProfiles } from "@/features/settings/hooks/useProfiles";
import { useChangePassword, useGeneralSettings, useUpdateGeneralSettings } from "@/features/settings/hooks/useGeneralSettings";

type PasswordField = "current" | "new" | "confirm";

type PasswordStrength = {
  hasMinLength: boolean;
  hasUppercase: boolean;
  hasLowercase: boolean;
  hasNumber: boolean;
  hasSpecialChar: boolean;
};

export function GeneralSettingsPanel() {
  const { logout } = useAuth();
  const settingsQuery = useGeneralSettings();
  const profilesQuery = useProfiles();
  const llmProviderQuery = useLLMProviderSettings();
  const updateSettings = useUpdateGeneralSettings();
  const changePassword = useChangePassword();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [visibleFields, setVisibleFields] = useState<Record<PasswordField, boolean>>({
    current: false,
    new: false,
    confirm: false,
  });
  const [passwordMessage, setPasswordMessage] = useState("");
  const [settingsMessage, setSettingsMessage] = useState("");

  const profiles = useMemo(() => profilesQuery.data ?? [], [profilesQuery.data]);
  const defaultProfile = useMemo(() => profiles.find((profile) => profile.is_default), [profiles]);
  const llmSettings = llmProviderQuery.data;
  const smallLLMReady = Boolean(llmSettings?.configured && llmSettings.small_model);
  const passwordStrength = useMemo(() => checkPasswordStrength(newPassword), [newPassword]);
  const isPasswordValid = Object.values(passwordStrength).every(Boolean);
  const passwordsMatch = newPassword === confirmPassword && confirmPassword.length > 0;
  const settings = settingsQuery.data;
  const settingsBusy = settingsQuery.isLoading || profilesQuery.isLoading || llmProviderQuery.isLoading || updateSettings.isPending;
  const canEnableAutoTranscribe = Boolean(defaultProfile);
  const canEnableAutoRename = smallLLMReady;

  const handlePasswordSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setPasswordMessage("");
    if (!isPasswordValid) {
      setPasswordMessage("New password does not meet the requirements.");
      return;
    }
    if (!passwordsMatch) {
      setPasswordMessage("New passwords do not match.");
      return;
    }
    try {
      await changePassword.mutateAsync({
        current_password: currentPassword,
        new_password: newPassword,
        confirm_password: confirmPassword,
      });
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setPasswordMessage("Password changed. Sign in again with the new password.");
      window.setTimeout(() => logout(), 1200);
    } catch (error) {
      setPasswordMessage(error instanceof Error ? error.message : "Could not change password.");
    }
  };

  const handleAutoTranscriptionChange = async (enabled: boolean) => {
    if (enabled && !canEnableAutoTranscribe) {
      setSettingsMessage("Set a default transcription profile before enabling automatic transcription.");
      return;
    }
    await saveSetting({ auto_transcription_enabled: enabled });
  };

  const handleAutoRenameChange = async (enabled: boolean) => {
    if (enabled && !canEnableAutoRename) {
      setSettingsMessage("Configure an LLM provider and small model before enabling automatic renaming.");
      return;
    }
    await saveSetting({ auto_rename_enabled: enabled });
  };

  const saveSetting = async (payload: UpdateGeneralSettingsPayload) => {
    setSettingsMessage("");
    try {
      await updateSettings.mutateAsync(payload);
      setSettingsMessage("Settings saved.");
    } catch (error) {
      setSettingsMessage(error instanceof Error ? error.message : "Could not save settings.");
    }
  };

  const toggleVisibility = (field: PasswordField) => {
    setVisibleFields((current) => ({ ...current, [field]: !current[field] }));
  };

  return (
    <div className="scr-general-settings">
      <section className="scr-settings-panel" aria-label="General automation settings">
        <div className="scr-settings-panel-head">
          <div>
            <h2 className="scr-settings-heading">General</h2>
            <p className="scr-settings-copy">Account security and automatic file workflows.</p>
          </div>
        </div>

        {settingsQuery.error || profilesQuery.error || llmProviderQuery.error ? (
          <div className="scr-alert" role="alert">
            {settingsQuery.error instanceof Error
              ? settingsQuery.error.message
              : profilesQuery.error instanceof Error
                ? profilesQuery.error.message
                : llmProviderQuery.error instanceof Error
                  ? llmProviderQuery.error.message
                  : "Could not load general settings."}
          </div>
        ) : null}

        {settingsMessage ? <div className="scr-alert scr-alert-neutral">{settingsMessage}</div> : null}

        <div className="scr-general-list">
          <SettingToggle
            icon={<Workflow size={18} aria-hidden="true" />}
            id="auto-transcription"
            title="Auto transcribe new audio"
            description={
              defaultProfile
                ? `Uses ${defaultProfile.name} when an audio file is ready.`
                : "Requires a default ASR profile."
            }
            checked={settings?.auto_transcription_enabled ?? false}
            disabled={settingsBusy || (!canEnableAutoTranscribe && !settings?.auto_transcription_enabled)}
            disabledReason={!canEnableAutoTranscribe ? "Set a default profile in the ASR tab." : undefined}
            onCheckedChange={handleAutoTranscriptionChange}
          />

          <SettingToggle
            icon={<WandSparkles size={18} aria-hidden="true" />}
            id="auto-rename"
            title="Auto rename audio"
            description={
              smallLLMReady
                ? `Uses ${llmSettings?.small_model} after a summary is available.`
                : "Requires an LLM provider and small model."
            }
            checked={settings?.auto_rename_enabled ?? false}
            disabled={settingsBusy || (!canEnableAutoRename && !settings?.auto_rename_enabled)}
            disabledReason={!canEnableAutoRename ? "Configure the small model in the LLM Providers tab." : undefined}
            onCheckedChange={handleAutoRenameChange}
          />
        </div>
      </section>

      <section className="scr-settings-panel" aria-label="Password settings">
        <div className="scr-settings-panel-head">
          <div>
            <h2 className="scr-settings-heading">Password</h2>
            <p className="scr-settings-copy">Update the password for this account.</p>
          </div>
          <Lock size={18} aria-hidden="true" />
        </div>

        {passwordMessage ? <div className="scr-alert scr-alert-neutral">{passwordMessage}</div> : null}

        <form className="scr-password-form" onSubmit={handlePasswordSubmit}>
          <PasswordInput
            id="current-password"
            label="Current password"
            value={currentPassword}
            visible={visibleFields.current}
            autoComplete="current-password"
            onChange={setCurrentPassword}
            onToggleVisibility={() => toggleVisibility("current")}
          />
          <PasswordInput
            id="new-password"
            label="New password"
            value={newPassword}
            visible={visibleFields.new}
            autoComplete="new-password"
            onChange={setNewPassword}
            onToggleVisibility={() => toggleVisibility("new")}
          />
          {newPassword ? <PasswordStrengthList strength={passwordStrength} /> : null}
          <PasswordInput
            id="confirm-password"
            label="Confirm new password"
            value={confirmPassword}
            visible={visibleFields.confirm}
            autoComplete="new-password"
            onChange={setConfirmPassword}
            onToggleVisibility={() => toggleVisibility("confirm")}
          />
          {confirmPassword ? (
            <div className="scr-password-check" data-valid={passwordsMatch}>
              {passwordsMatch ? <Check size={14} aria-hidden="true" /> : <X size={14} aria-hidden="true" />}
              <span>{passwordsMatch ? "Passwords match" : "Passwords do not match"}</span>
            </div>
          ) : null}
          <AppButton
            type="submit"
            disabled={changePassword.isPending || !currentPassword.trim() || !isPasswordValid || !passwordsMatch}
          >
            {changePassword.isPending ? "Changing..." : "Change password"}
          </AppButton>
        </form>
      </section>
    </div>
  );
}

function SettingToggle({
  icon,
  id,
  title,
  description,
  checked,
  disabled,
  disabledReason,
  onCheckedChange,
}: {
  icon: ReactNode;
  id: string;
  title: string;
  description: string;
  checked: boolean;
  disabled: boolean;
  disabledReason?: string;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <div className="scr-general-toggle">
      <div className="scr-general-toggle-icon">{icon}</div>
      <div className="scr-general-toggle-copy">
        <Label htmlFor={id} className="scr-general-toggle-title">
          {title}
        </Label>
        <p>{description}</p>
        {disabled && disabledReason ? <p className="scr-general-toggle-hint">{disabledReason}</p> : null}
      </div>
      <Switch id={id} checked={checked} disabled={disabled} onCheckedChange={onCheckedChange} aria-label={title} />
    </div>
  );
}

function PasswordInput({
  id,
  label,
  value,
  visible,
  autoComplete,
  onChange,
  onToggleVisibility,
}: {
  id: string;
  label: string;
  value: string;
  visible: boolean;
  autoComplete: string;
  onChange: (value: string) => void;
  onToggleVisibility: () => void;
}) {
  return (
    <div className="scr-password-field">
      <Label htmlFor={id}>{label}</Label>
      <div className="scr-password-input-wrap">
        <Input
          id={id}
          type={visible ? "text" : "password"}
          value={value}
          autoComplete={autoComplete}
          onChange={(event) => onChange(event.target.value)}
          required
        />
        <IconButton label={visible ? `Hide ${label.toLowerCase()}` : `Show ${label.toLowerCase()}`} onClick={onToggleVisibility}>
          {visible ? <EyeOff size={15} aria-hidden="true" /> : <Eye size={15} aria-hidden="true" />}
        </IconButton>
      </div>
    </div>
  );
}

function PasswordStrengthList({ strength }: { strength: PasswordStrength }) {
  return (
    <div className="scr-password-rules" aria-label="Password requirements">
      <PasswordRule label="At least 8 characters" met={strength.hasMinLength} />
      <PasswordRule label="One uppercase letter" met={strength.hasUppercase} />
      <PasswordRule label="One lowercase letter" met={strength.hasLowercase} />
      <PasswordRule label="One number" met={strength.hasNumber} />
      <PasswordRule label="One special character" met={strength.hasSpecialChar} />
    </div>
  );
}

function PasswordRule({ label, met }: { label: string; met: boolean }) {
  return (
    <div className="scr-password-rule" data-met={met}>
      {met ? <Check size={13} aria-hidden="true" /> : <X size={13} aria-hidden="true" />}
      <span>{label}</span>
    </div>
  );
}

function checkPasswordStrength(password: string): PasswordStrength {
  return {
    hasMinLength: password.length >= 8,
    hasUppercase: /[A-Z]/.test(password),
    hasLowercase: /[a-z]/.test(password),
    hasNumber: /\d/.test(password),
    hasSpecialChar: /[!@#$%^&*(),.?":{}|<>]/.test(password),
  };
}
