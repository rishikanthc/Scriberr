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
import { useAuth } from "../contexts/AuthContext";

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
}

export function TranscribeDDialog({
  open,
  onOpenChange,
  onStartTranscription,
  loading = false,
}: TranscribeDDialogProps) {
  const { getAuthHeaders } = useAuth();
  const [profiles, setProfiles] = useState<TranscriptionProfile[]>([]);
  const [selectedProfileId, setSelectedProfileId] = useState<string>("");
  const [profilesLoading, setProfilesLoading] = useState(false);

  // Fetch profiles when dialog opens
  useEffect(() => {
    if (open) {
      fetchProfiles();
    }
  }, [open]);

  const fetchProfiles = async () => {
    try {
      setProfilesLoading(true);
      const response = await fetch("/api/v1/profiles", {
        headers: {
          ...getAuthHeaders(),
        },
      });

      if (response.ok) {
        const data: TranscriptionProfile[] = await response.json();
        setProfiles(data);
        
        // Select the default profile by default, or first profile if no default
        const defaultProfile = data.find(p => p.is_default);
        if (defaultProfile) {
          setSelectedProfileId("default");
        } else if (data.length > 0) {
          setSelectedProfileId(data[0].id);
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

    if (selectedProfileId === "default") {
      // Find the default profile
      const defaultProfile = profiles.find(p => p.is_default);
      if (defaultProfile) {
        onStartTranscription(defaultProfile.parameters, defaultProfile.id);
      }
    } else {
      const selectedProfile = profiles.find(p => p.id === selectedProfileId);
      if (selectedProfile) {
        onStartTranscription(selectedProfile.parameters, selectedProfile.id);
      }
    }
  };

  const handleProfileChange = (value: string) => {
    setSelectedProfileId(value);
  };

  const getSelectedProfileName = () => {
    if (selectedProfileId === "default") {
      const defaultProfile = profiles.find(p => p.is_default);
      return defaultProfile ? `${defaultProfile.name} (Default)` : "Default";
    }
    const profile = profiles.find(p => p.id === selectedProfileId);
    return profile?.name || "";
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700">
        <DialogHeader>
          <DialogTitle className="text-gray-900 dark:text-gray-100">
            Transcribe with Profile
          </DialogTitle>
          <DialogDescription className="text-gray-600 dark:text-gray-400">
            Choose a saved profile to start transcription with your preferred settings.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="profile" className="text-gray-700 dark:text-gray-300 font-medium">
              Select Profile
            </Label>
            
            {profilesLoading ? (
              <div className="flex items-center space-x-2 p-3 bg-gray-50 dark:bg-gray-800 rounded-md border border-gray-200 dark:border-gray-700">
                <Loader2 className="h-4 w-4 animate-spin text-gray-500 dark:text-gray-400" />
                <span className="text-sm text-gray-600 dark:text-gray-400">Loading profiles...</span>
              </div>
            ) : profiles.length === 0 ? (
              <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded-md border border-gray-200 dark:border-gray-700">
                <span className="text-sm text-gray-600 dark:text-gray-400">No profiles available</span>
              </div>
            ) : (
              <Select
                value={selectedProfileId}
                onValueChange={handleProfileChange}
              >
                <SelectTrigger className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-900 dark:text-gray-100 focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400">
                  <SelectValue placeholder="Choose a profile..." />
                </SelectTrigger>
                <SelectContent className="bg-white dark:bg-gray-800 border-gray-200 dark:border-gray-700 max-h-60">
                  {/* Default profile option */}
                  {profiles.find(p => p.is_default) && (
                    <SelectItem 
                      value="default" 
                      className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700"
                    >
                      <div className="flex items-center space-x-2">
                        <span>Default</span>
                        <span className="text-xs text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/30 px-1.5 py-0.5 rounded">
                          {profiles.find(p => p.is_default)?.name}
                        </span>
                      </div>
                    </SelectItem>
                  )}
                  
                  {/* All profiles */}
                  {profiles.map((profile) => (
                    <SelectItem 
                      key={profile.id} 
                      value={profile.id}
                      className="text-gray-900 dark:text-gray-100 focus:bg-gray-100 dark:focus:bg-gray-700"
                    >
                      <div className="flex flex-col space-y-1">
                        <div className="flex items-center space-x-2">
                          <span>{profile.name}</span>
                          {profile.is_default && (
                            <span className="text-xs text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30 px-1.5 py-0.5 rounded">
                              Default
                            </span>
                          )}
                        </div>
                        {profile.description && (
                          <span className="text-xs text-gray-500 dark:text-gray-400 truncate">
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
            <div className="p-3 bg-gray-50 dark:bg-gray-800 rounded-md border border-gray-200 dark:border-gray-700">
              <div className="text-sm">
                <span className="font-medium text-gray-700 dark:text-gray-300">Selected: </span>
                <span className="text-gray-600 dark:text-gray-400">{getSelectedProfileName()}</span>
              </div>
              {(() => {
                const profile = selectedProfileId === "default" 
                  ? profiles.find(p => p.is_default)
                  : profiles.find(p => p.id === selectedProfileId);
                return profile?.description ? (
                  <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
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
            className="bg-white dark:bg-gray-800 border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
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
