import { useState, useEffect, useCallback } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Loader2 } from "lucide-react";
import type { WhisperXParams } from "./TranscriptionConfigDialog";
import { useAuth } from "@/features/auth/hooks/useAuth";

interface TranscriptionProfile {
  id: string;
  name: string;
  description?: string;
  is_default: boolean;
  parameters: WhisperXParams;
  created_at: string;
  updated_at: string;
}

interface TranscribeDDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onStartTranscription: (params: WhisperXParams, profileId?: string) => void;
  loading?: boolean;
  title?: string;
}

export function TranscribeDDialog({
  open,
  onOpenChange,
  onStartTranscription,
  loading = false,
  title,
}: TranscribeDDialogProps) {
  const { getAuthHeaders } = useAuth();
  const [profiles, setProfiles] = useState<TranscriptionProfile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string>("");
  const [profilesLoading, setProfilesLoading] = useState(false);
  const [defaultProfile, setDefaultProfile] = useState<TranscriptionProfile | null>(null);

  const fetchProfiles = useCallback(async () => {
    try {
      setProfilesLoading(true);

      // Fetch all profiles
      const profilesResponse = await fetch("/api/v1/profiles", {
        headers: {
          ...getAuthHeaders(),
        },
      });

      if (profilesResponse.ok) {
        const profilesData: TranscriptionProfile[] = await profilesResponse.json();
        setProfiles(profilesData);

        // Fetch user's default profile
        const defaultResponse = await fetch("/api/v1/user/default-profile", {
          headers: {
            ...getAuthHeaders(),
          },
        });

        if (defaultResponse.ok) {
          const defaultData: TranscriptionProfile = await defaultResponse.json();
          setDefaultProfile(defaultData);
          setSelectedProfileId(defaultData.id);
        } else if (defaultResponse.status === 404) {
          // No default profile set, use the first available profile
          setDefaultProfile(null);
          if (profilesData.length > 0) {
            setSelectedProfileId(profilesData[0].id);
          }
        }
      } else {
        console.error("Failed to fetch profiles");
      }
    } catch (error) {
      console.error("Error fetching profiles:", error);
    } finally {
      setProfilesLoading(false);
    }
  }, [getAuthHeaders]);

  // Fetch profiles when dialog opens
  useEffect(() => {
    if (open) {
      fetchProfiles();
    }
  }, [open, fetchProfiles]);

  const handleStartTranscription = () => {
    if (!selectedProfileId) return;

    const selectedProfile = profiles.find(p => p.id === selectedProfileId);
    if (selectedProfile) {
      onStartTranscription(selectedProfile.parameters, selectedProfile.id);
    }
  };

  const handleProfileChange = (value: string) => {
    setSelectedProfileId(value);
  };



  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md glass-card rounded-[var(--radius-card)] p-0 gap-0 overflow-hidden border border-[var(--border-subtle)] shadow-[var(--shadow-float)]">
        <DialogHeader className="p-6 pb-2">
          <DialogTitle className="text-xl font-bold tracking-tight text-[var(--text-primary)]">
            {title || "Transcribe with Profile"}
          </DialogTitle>
          <DialogDescription className="text-[var(--text-secondary)] text-sm mt-1.5">
            Choose a saved profile to start transcription with your preferred settings.
          </DialogDescription>
        </DialogHeader>



        <div className="space-y-4 px-6 py-2">
          <div className="space-y-2">
            <Label htmlFor="profile" className="text-[var(--text-secondary)] font-medium">
              Select Profile
            </Label>

            {profilesLoading ? (
              <div className="flex items-center space-x-2 p-3 bg-[var(--bg-main)]/50 rounded-[var(--radius-btn)] border border-[var(--border-subtle)]">
                <Loader2 className="h-4 w-4 animate-spin text-[var(--text-tertiary)]" />
                <span className="text-sm text-[var(--text-secondary)]">Loading profiles...</span>
              </div>
            ) : profiles.length === 0 ? (
              <div className="p-3 bg-[var(--bg-main)]/50 rounded-[var(--radius-btn)] border border-[var(--border-subtle)]">
                <span className="text-sm text-[var(--text-secondary)]">No profiles available</span>
              </div>
            ) : (
              <Select
                value={selectedProfileId}
                onValueChange={handleProfileChange}
              >
                <SelectTrigger className="h-11 rounded-[var(--radius-btn)] bg-[var(--bg-main)] border border-[var(--border-subtle)] text-[var(--text-primary)] focus:ring-[var(--brand-light)] focus:border-[var(--brand-solid)] shadow-none">
                  <SelectValue placeholder="Choose a profile..." />
                </SelectTrigger>
                <SelectContent className="glass-card rounded-[var(--radius-btn)] border border-[var(--border-subtle)] shadow-[var(--shadow-float)]">
                  {/* All profiles */}
                  {profiles.map((profile) => (
                    <SelectItem
                      key={profile.id}
                      value={profile.id}
                      className="text-[var(--text-primary)] focus:bg-[var(--brand-light)] focus:text-[var(--brand-solid)] rounded-[8px] my-1 mx-1 cursor-pointer"
                    >
                      <div className="flex flex-col space-y-1">
                        <div className="flex items-center space-x-2">
                          <span>{profile.name}</span>
                          {defaultProfile && profile.id === defaultProfile.id && (
                            <span className="text-xs text-[var(--success-solid)] bg-[var(--success-translucent)] px-1.5 py-0.5 rounded">
                              Default
                            </span>
                          )}
                        </div>
                        {profile.description && (
                          <span className="text-xs text-[var(--text-tertiary)] truncate">
                            {profile.description}
                          </span>
                        )}
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>


        </div>

        <DialogFooter className="p-6 pt-2 gap-3">
          <Button
            variant="ghost"
            onClick={() => onOpenChange(false)}
            className="rounded-[var(--radius-btn)] text-[var(--text-secondary)] hover:bg-[var(--secondary)] hover:text-[var(--text-primary)]"
          >
            Cancel
          </Button>
          <Button
            onClick={handleStartTranscription}
            disabled={loading || !selectedProfileId || profilesLoading || profiles.length === 0}
            className="min-w-[140px] !bg-[var(--brand-gradient)] hover:!opacity-90 !text-black dark:!text-white border-none shadow-lg shadow-orange-500/20"
          >
            {loading ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Starting...
              </>
            ) : (
              "Start Transcription"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog >
  );
}
