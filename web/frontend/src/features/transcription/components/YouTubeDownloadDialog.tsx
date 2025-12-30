import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Youtube, Download, AlertCircle, CheckCircle } from "lucide-react";
import { useYouTubeDownload } from "@/features/transcription/hooks/useAudioFiles";

interface YouTubeDownloadDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onDownloadComplete?: () => void;
}

export function YouTubeDownloadDialog({
  isOpen,
  onClose,
  onDownloadComplete
}: YouTubeDownloadDialogProps) {
  const { mutateAsync: downloadYouTube, isPending: isDownloading } = useYouTubeDownload();
  const [url, setUrl] = useState("");
  const [title, setTitle] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const validateYouTubeUrl = (url: string): boolean => {
    return url.includes('youtube.com') || url.includes('youtu.be');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!url.trim()) {
      setError("Please enter a YouTube URL");
      return;
    }

    if (!validateYouTubeUrl(url)) {
      setError("Please enter a valid YouTube URL");
      return;
    }

    setError(null);

    try {
      await downloadYouTube({ url, title });
      setSuccess(true);
      setTimeout(() => {
        handleClose();
        onDownloadComplete?.();
      }, 2000);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Network error occurred. Please try again.");
    }
  };

  const handleClose = () => {
    setTitle("");
    setError(null);
    setSuccess(false);
    onClose();
  };

  const getYouTubeVideoId = (url: string): string | null => {
    const regex = /(?:youtube\.com\/(?:[^/]+\/.+\/|(?:v|e(?:mbed)?)\/|.*[?&]v=)|youtu\.be\/)([^"&?/\s]{11})/;
    const match = url.match(regex);
    return match ? match[1] : null;
  };

  const videoId = getYouTubeVideoId(url);

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Youtube className="h-5 w-5 text-[var(--error)]" />
            Download from YouTube
          </DialogTitle>
          <DialogDescription>
            Enter a YouTube video URL to download its audio for transcription
          </DialogDescription>
        </DialogHeader>

        {success ? (
          <div className="space-y-4 py-4">
            <div className="flex items-center justify-center">
              <CheckCircle className="h-12 w-12 text-[var(--success)]" />
            </div>
            <div className="text-center">
              <h3 className="font-medium text-[var(--text-primary)] mb-2">
                Download Complete!
              </h3>
              <p className="text-sm text-[var(--text-secondary)]">
                The audio has been downloaded and added to your audio files.
              </p>
            </div>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="youtube-url">YouTube URL</Label>
              <Input
                id="youtube-url"
                type="url"
                placeholder="https://www.youtube.com/watch?v=..."
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                disabled={isDownloading}
              />

              {/* YouTube thumbnail preview */}
              {videoId && (
                <div className="mt-2">
                  <img
                    src={`https://img.youtube.com/vi/${videoId}/mqdefault.jpg`}
                    alt="YouTube thumbnail"
                    className="w-full h-32 object-cover rounded-md border"
                    onError={(e) => {
                      (e.target as HTMLImageElement).style.display = 'none';
                    }}
                  />
                </div>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="custom-title">Custom Title (Optional)</Label>
              <Input
                id="custom-title"
                type="text"
                placeholder="Leave empty to use video title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                disabled={isDownloading}
              />
              <p className="text-xs text-[var(--text-tertiary)]">
                If left empty, the video's title will be used automatically
              </p>
            </div>

            {error && (
              <div className="flex items-center gap-2 p-3 bg-[var(--error)]/10 border border-[var(--error)]/20 rounded-[var(--radius-input)]">
                <AlertCircle className="h-4 w-4 text-[var(--error)]" />
                <p className="text-sm text-[var(--error)]">{error}</p>
              </div>
            )}

            <div className="flex gap-2 justify-end pt-4">
              <Button
                type="button"
                variant="outline"
                onClick={handleClose}
                disabled={isDownloading}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={isDownloading || !url.trim()}
                className="min-w-24"
              >
                {isDownloading ? (
                  <>
                    <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin mr-2" />
                    Downloading...
                  </>
                ) : (
                  <>
                    <Download className="h-4 w-4 mr-2" />
                    Download
                  </>
                )}
              </Button>
            </div>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}