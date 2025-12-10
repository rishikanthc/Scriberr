import { useState, useEffect } from "react";
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

  // Fetch profiles when dialog opens
  useEffect(() => {
    if (open) {
      fetchProfiles();
    }
  }, [open]);

  const fetchProfiles = async () => {
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
  };

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

  const getSelectedProfileName = () => {
    const profile = profiles.find(p => p.id === selectedProfileId);
    if (profile && defaultProfile && profile.id === defaultProfile.id) {
      return `${profile.name} (Default)`;
    }
    return profile?.name || "";
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700">
        <DialogHeader>
          <DialogTitle className="text-carbon-900 dark:text-carbon-100">
            {title || "Transcribe with Profile"}
          </DialogTitle>
          <DialogDescription className="text-carbon-600 dark:text-carbon-400">
            Choose a saved profile to start transcription with your preferred settings.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="profile" className="text-carbon-700 dark:text-carbon-300 font-medium">
              Select Profile
            </Label>

            {profilesLoading ? (
              <div className="flex items-center space-x-2 p-3 bg-carbon-50 dark:bg-carbon-800 rounded-md border border-carbon-200 dark:border-carbon-700">
                <Loader2 className="h-4 w-4 animate-spin text-carbon-500 dark:text-carbon-400" />
                <span className="text-sm text-carbon-600 dark:text-carbon-400">Loading profiles...</span>
              </div>
            ) : profiles.length === 0 ? (
              <div className="p-3 bg-carbon-50 dark:bg-carbon-800 rounded-md border border-carbon-200 dark:border-carbon-700">
                <span className="text-sm text-carbon-600 dark:text-carbon-400">No profiles available</span>
              </div>
            ) : (
              <Select
                value={selectedProfileId}
                onValueChange={handleProfileChange}
              >
                <SelectTrigger className="bg-white dark:bg-carbon-800 border-carbon-300 dark:border-carbon-600 text-carbon-900 dark:text-carbon-100 focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400">
                  <SelectValue placeholder="Choose a profile..." />
                </SelectTrigger>
                <SelectContent className="bg-white dark:bg-carbon-800 border-carbon-200 dark:border-carbon-700 max-h-60">
                  {/* All profiles */}
                  {profiles.map((profile) => (
                    <SelectItem
                      key={profile.id}
                      value={profile.id}
                      className="text-carbon-900 dark:text-carbon-100 focus:bg-carbon-100 dark:focus:bg-carbon-700"
                    >
                      <div className="flex flex-col space-y-1">
                        <div className="flex items-center space-x-2">
                          <span>{profile.name}</span>
                          {defaultProfile && profile.id === defaultProfile.id && (
                            <span className="text-xs text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30 px-1.5 py-0.5 rounded">
                              Default
                            </span>
                          )}
                        </div>
                        {profile.description && (
                          <span className="text-xs text-carbon-500 dark:text-carbon-400 truncate">
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

          {/* Show selected profile details */}
          {selectedProfileId && !profilesLoading && (
            <div className="p-3 bg-carbon-50 dark:bg-carbon-800 rounded-md border border-carbon-200 dark:border-carbon-700">
              <div className="text-sm">
                <span className="font-medium text-carbon-700 dark:text-carbon-300">Selected: </span>
                <span className="text-carbon-600 dark:text-carbon-400">{getSelectedProfileName()}</span>
              </div>
              {(() => {
                const profile = profiles.find(p => p.id === selectedProfileId);
                return profile?.description ? (
                  <div className="text-xs text-carbon-500 dark:text-carbon-400 mt-1">
                    {profile.description}
                  </div>
                ) : null;
              })()}
            </div>
          )}
        </div>

        <DialogFooter className="gap-2">
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            className="bg-white dark:bg-carbon-800 border-carbon-300 dark:border-carbon-600 text-carbon-700 dark:text-carbon-200 hover:bg-carbon-50 dark:hover:bg-carbon-700"
          >
            Cancel
          </Button>
          <Button
            onClick={handleStartTranscription}
            disabled={loading || !selectedProfileId || profilesLoading || profiles.length === 0}
            className="bg-blue-600 dark:bg-blue-700 hover:bg-blue-700 dark:hover:bg-blue-800 text-white min-w-[120px]"
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
    </Dialog>
  );
}
