import React, { useState, useRef, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardContent } from "@/components/ui/card";
import { Upload, Clock, CheckCircle, XCircle, FileAudio, Zap } from "lucide-react";
import { useAuth } from "../contexts/AuthContext";

interface Profile {
  id: string;
  name: string;
  description?: string;
  is_default: boolean;
}

interface QuickTranscriptionJob {
  id: string;
  status: "processing" | "completed" | "failed";
  transcript?: string;
  error_message?: string;
  created_at: string;
  expires_at: string;
}

interface TranscriptSegment {
  start: number;
  end: number;
  text: string;
  words?: Array<{
    word: string;
    start: number;
    end: number;
    score: number;
  }>;
}

interface TranscriptData {
  segments: TranscriptSegment[];
  language: string;
}

interface QuickTranscriptionDialogProps {
  isOpen: boolean;
  onClose: () => void;
}

export function QuickTranscriptionDialog({ isOpen, onClose }: QuickTranscriptionDialogProps) {
  const { getAuthHeaders } = useAuth();
  const [step, setStep] = useState<"upload" | "profile" | "processing" | "result">("upload");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [selectedProfile, setSelectedProfile] = useState<string>("");
  const [job, setJob] = useState<QuickTranscriptionJob | null>(null);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const pollIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // Load profiles when dialog opens
  useEffect(() => {
    if (isOpen && profiles.length === 0) {
      loadProfiles();
    }
  }, [isOpen]);

  // Cleanup polling on unmount
  useEffect(() => {
    return () => {
      if (pollIntervalRef.current) {
        clearInterval(pollIntervalRef.current);
      }
    };
  }, []);

  const loadProfiles = async () => {
    try {
      const response = await fetch("/api/v1/profiles/", {
        headers: getAuthHeaders(),
      });
      if (response.ok) {
        const data = await response.json();
        setProfiles(data);
        // Set default profile if available
        const defaultProfile = data.find((p: Profile) => p.is_default);
        if (defaultProfile) {
          setSelectedProfile(defaultProfile.name);
        }
      }
    } catch (err) {
      console.error("Failed to load profiles:", err);
    }
  };

  const handleFileSelect = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file && file.type.startsWith("audio/")) {
      setSelectedFile(file);
      setStep("profile");
    }
  };

  const handleSubmit = async () => {
    if (!selectedFile) return;

    setStep("processing");
    setError(null);

    try {
      const formData = new FormData();
      formData.append("audio", selectedFile);
      
      if (selectedProfile) {
        formData.append("profile_name", selectedProfile);
      }

      const response = await fetch("/api/v1/transcription/quick", {
        method: "POST",
        headers: getAuthHeaders(),
        body: formData,
      });

      if (response.ok) {
        const jobData = await response.json();
        setJob(jobData);
        startPolling(jobData.id);
      } else {
        const errorData = await response.json();
        setError(errorData.error || "Failed to submit transcription");
        setStep("profile");
      }
    } catch (err) {
      setError("Network error occurred");
      setStep("profile");
    }
  };

  const startPolling = (jobId: string) => {
    pollIntervalRef.current = setInterval(async () => {
      try {
        const response = await fetch(`/api/v1/transcription/quick/${jobId}`, {
          headers: getAuthHeaders(),
        });

        if (response.ok) {
          const jobData = await response.json();
          setJob(jobData);

          if (jobData.status === "completed" || jobData.status === "failed") {
            if (pollIntervalRef.current) {
              clearInterval(pollIntervalRef.current);
              pollIntervalRef.current = null;
            }
            setStep("result");
          }
        }
      } catch (err) {
        console.error("Polling error:", err);
      }
    }, 2000); // Poll every 2 seconds
  };

  const handleClose = () => {
    if (pollIntervalRef.current) {
      clearInterval(pollIntervalRef.current);
      pollIntervalRef.current = null;
    }
    setStep("upload");
    setSelectedFile(null);
    setSelectedProfile("");
    setJob(null);
    setError(null);
    onClose();
  };

  const formatTime = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
  };

  const formatTranscript = (transcript: string): React.ReactElement[] => {
    try {
      const data: TranscriptData = JSON.parse(transcript);
      
      return data.segments.map((segment, index) => (
        <div key={index} className="mb-4 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <div className="flex items-center gap-2 mb-2">
            <span className="text-xs font-mono text-gray-500 bg-gray-200 dark:bg-gray-700 px-2 py-1 rounded">
              {formatTime(segment.start)} - {formatTime(segment.end)}
            </span>
          </div>
          <p className="text-gray-900 dark:text-gray-100 leading-relaxed">
            {segment.text.trim()}
          </p>
        </div>
      ));
    } catch (err) {
      // Fallback for plain text
      return [
        <div key="fallback" className="p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
          <p className="text-gray-900 dark:text-gray-100 leading-relaxed whitespace-pre-wrap">
            {transcript}
          </p>
        </div>
      ];
    }
  };

  const getExpiryInfo = (): string => {
    if (!job?.expires_at) return "";
    
    const expiryTime = new Date(job.expires_at);
    const now = new Date();
    const hoursRemaining = Math.ceil((expiryTime.getTime() - now.getTime()) / (1000 * 60 * 60));
    
    return `Expires in ${hoursRemaining} hours`;
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Zap className="h-5 w-5 text-yellow-500" />
            Quick Transcription
          </DialogTitle>
          <DialogDescription>
            Fast transcription without saving to your library - files auto-delete after 6 hours
          </DialogDescription>
        </DialogHeader>

        {step === "upload" && (
          <div className="space-y-4">
            <Card 
              className="border-2 border-dashed border-gray-300 dark:border-gray-600 hover:border-blue-400 dark:hover:border-blue-500 cursor-pointer transition-colors"
              onClick={handleFileSelect}
            >
              <CardContent className="flex flex-col items-center justify-center py-12">
                <Upload className="h-12 w-12 text-gray-400 mb-4" />
                <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
                  Select Audio File
                </h3>
                <p className="text-gray-500 dark:text-gray-400 text-center">
                  Click to choose an audio file from your device
                </p>
              </CardContent>
            </Card>
            
            <input
              ref={fileInputRef}
              type="file"
              accept="audio/*"
              onChange={handleFileChange}
              className="hidden"
            />
          </div>
        )}

        {step === "profile" && (
          <div className="space-y-4">
            <div className="flex items-center gap-3 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
              <FileAudio className="h-8 w-8 text-blue-500" />
              <div>
                <h3 className="font-medium text-gray-900 dark:text-gray-100">
                  {selectedFile?.name}
                </h3>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  {selectedFile && (selectedFile.size / 1024 / 1024).toFixed(2)} MB
                </p>
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300">
                Transcription Profile
              </label>
              <Select value={selectedProfile} onValueChange={setSelectedProfile}>
                <SelectTrigger>
                  <SelectValue placeholder="Select a profile or use default settings" />
                </SelectTrigger>
                <SelectContent>
                  {profiles.map((profile) => (
                    <SelectItem key={profile.id} value={profile.name}>
                      <div className="flex items-center gap-2">
                        <span>{profile.name}</span>
                        {profile.is_default && (
                          <span className="text-xs bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 px-2 py-0.5 rounded">
                            Default
                          </span>
                        )}
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                Leave empty to use default settings
              </p>
            </div>

            {error && (
              <div className="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
              </div>
            )}

            <div className="flex gap-2 justify-end">
              <Button variant="outline" onClick={handleClose}>
                Cancel
              </Button>
              <Button onClick={handleSubmit}>
                Start Transcription
              </Button>
            </div>
          </div>
        )}

        {step === "processing" && job && (
          <div className="space-y-4 text-center">
            <div className="flex flex-col items-center">
              <Clock className="h-12 w-12 text-blue-500 animate-spin mb-4" />
              <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-2">
                Transcribing Audio...
              </h3>
              <p className="text-gray-500 dark:text-gray-400">
                This may take a few minutes depending on the audio length
              </p>
              <p className="text-xs text-gray-400 dark:text-gray-500 mt-2">
                {getExpiryInfo()}
              </p>
            </div>
            
            <div className="text-left bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
              <h4 className="font-medium mb-2">Job Details:</h4>
              <p className="text-sm text-gray-600 dark:text-gray-400">ID: {job.id}</p>
              <p className="text-sm text-gray-600 dark:text-gray-400">Status: {job.status}</p>
            </div>

            <Button variant="outline" onClick={handleClose}>
              Cancel
            </Button>
          </div>
        )}

        {step === "result" && job && (
          <div className="space-y-4">
            {job.status === "completed" && job.transcript ? (
              <>
                <div className="flex items-center gap-2 p-3 bg-green-50 dark:bg-green-900/20 rounded-lg">
                  <CheckCircle className="h-5 w-5 text-green-500" />
                  <div>
                    <h3 className="font-medium text-green-900 dark:text-green-100">
                      Transcription Complete
                    </h3>
                    <p className="text-sm text-green-600 dark:text-green-400">
                      {getExpiryInfo()}
                    </p>
                  </div>
                </div>

                <div className="max-h-96 overflow-y-auto space-y-2">
                  {formatTranscript(job.transcript)}
                </div>

                <div className="flex gap-2 justify-end">
                  <Button 
                    variant="outline" 
                    onClick={() => {
                      if (job.transcript) {
                        navigator.clipboard.writeText(JSON.parse(job.transcript).segments.map((s: TranscriptSegment) => s.text.trim()).join('\n'));
                      }
                    }}
                  >
                    Copy Text
                  </Button>
                  <Button onClick={handleClose}>
                    Close
                  </Button>
                </div>
              </>
            ) : (
              <>
                <div className="flex items-center gap-2 p-3 bg-red-50 dark:bg-red-900/20 rounded-lg">
                  <XCircle className="h-5 w-5 text-red-500" />
                  <div>
                    <h3 className="font-medium text-red-900 dark:text-red-100">
                      Transcription Failed
                    </h3>
                    <p className="text-sm text-red-600 dark:text-red-400">
                      {job.error_message || "An unknown error occurred"}
                    </p>
                  </div>
                </div>

                <div className="flex gap-2 justify-end">
                  <Button variant="outline" onClick={() => setStep("upload")}>
                    Try Again
                  </Button>
                  <Button onClick={handleClose}>
                    Close
                  </Button>
                </div>
              </>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}