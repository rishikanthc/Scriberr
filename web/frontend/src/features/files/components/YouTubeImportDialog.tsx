import { useEffect, useId, useState, type FormEvent } from "react";
import { Loader2 } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";

type YouTubeImportDialogProps = {
  open: boolean;
  importing: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (url: string) => Promise<void>;
};

export function YouTubeImportDialog({ open, importing, onOpenChange, onSubmit }: YouTubeImportDialogProps) {
  const inputId = useId();
  const [url, setUrl] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setUrl("");
      setError(null);
    }
  }, [open]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmedURL = url.trim();
    if (!trimmedURL) {
      setError("Paste a YouTube URL.");
      return;
    }
    setError(null);
    try {
      await onSubmit(trimmedURL);
      onOpenChange(false);
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "YouTube import failed.");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="scr-youtube-dialog">
        <DialogHeader>
          <DialogTitle>Import from YouTube</DialogTitle>
          <DialogDescription>Paste a YouTube video URL. Scriberr will save the audio as a recording.</DialogDescription>
        </DialogHeader>
        <form className="scr-youtube-form" onSubmit={handleSubmit}>
          <div className="scr-youtube-field">
            <Label htmlFor={inputId}>YouTube URL</Label>
            <Input
              id={inputId}
              type="url"
              value={url}
              placeholder="https://www.youtube.com/watch?v=..."
              disabled={importing}
              onChange={(event) => setUrl(event.target.value)}
            />
            {error && <p className="scr-youtube-error">{error}</p>}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" disabled={importing} onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={importing || !url.trim()}>
              {importing && <Loader2 size={15} className="scr-spin" aria-hidden="true" />}
              Import
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
