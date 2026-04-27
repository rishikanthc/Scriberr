import { useCallback, useMemo, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/features/auth/hooks/useAuth";
import { uploadFile, type ScriberrFile } from "@/features/files/api/filesApi";
import { filesQueryKey } from "@/features/files/hooks/useFiles";
import type { FileEvent } from "@/features/files/hooks/useFileEvents";

export type UploadItemStatus = "uploading" | "processing" | "ready" | "failed";

export type UploadItem = {
  id: string;
  fileId?: string;
  fileName: string;
  progress: number;
  status: UploadItemStatus;
  error?: string;
};

const audioExtensions = [".mp3", ".wav", ".flac", ".m4a", ".aac", ".ogg", ".opus", ".webm"];
const videoExtensions = [".mp4", ".mov", ".mkv", ".avi", ".webm", ".wmv", ".flv"];

export const importAccept = [
  "audio/*",
  "video/*",
  ...audioExtensions,
  ...videoExtensions,
].join(",");

export function useFileImport() {
  const { getAuthHeaders } = useAuth();
  const queryClient = useQueryClient();
  const [items, setItems] = useState<UploadItem[]>([]);

  const activeCount = useMemo(
    () => items.filter((item) => item.status === "uploading" || item.status === "processing").length,
    [items]
  );

  const importFiles = useCallback(async (fileList: FileList | File[]) => {
    const selected = Array.from(fileList).filter(isSupportedMediaFile);
    if (selected.length === 0) return;

    const createdItems: UploadItem[] = selected.map((file) => ({
      id: crypto.randomUUID(),
      fileName: file.name,
      progress: 0,
      status: "uploading",
    }));

    setItems((current) => [...createdItems, ...current]);

    await Promise.allSettled(
      selected.map(async (file, index) => {
        const uploadId = createdItems[index].id;

        try {
          const uploaded = await uploadFile(file, getAuthHeaders(), (progress) => {
            setItems((current) =>
              current.map((item) => item.id === uploadId ? { ...item, progress } : item)
            );
          });

          const status = uploadStatusFromFile(uploaded);
          setItems((current) =>
            current.map((item) =>
              item.id === uploadId
                ? {
                  ...item,
                  fileId: uploaded.id,
                  progress: 100,
                  status,
                }
                : item
            )
          );

          queryClient.invalidateQueries({ queryKey: filesQueryKey });
          queryClient.invalidateQueries({ queryKey: ["audioFiles"] });
        } catch (error) {
          setItems((current) =>
            current.map((item) =>
              item.id === uploadId
                ? {
                  ...item,
                  status: "failed",
                  error: error instanceof Error ? error.message : "Upload failed",
                }
                : item
            )
          );
        }
      })
    );
  }, [getAuthHeaders, queryClient]);

  const handleFileEvent = useCallback((event: FileEvent) => {
    const fileId = event.data.id;
    if (!fileId) return;

    if (event.name === "file.ready") {
      setItems((current) =>
        current.map((item) => item.fileId === fileId ? { ...item, status: "ready", progress: 100 } : item)
      );
      window.setTimeout(() => {
        setItems((current) => current.filter((item) => item.fileId !== fileId || item.status !== "ready"));
      }, 2600);
    }

    if (event.name === "file.failed") {
      setItems((current) =>
        current.map((item) => item.fileId === fileId ? { ...item, status: "failed", error: "Import failed" } : item)
      );
    }
  }, []);

  const dismissItem = useCallback((id: string) => {
    setItems((current) => current.filter((item) => item.id !== id));
  }, []);

  return {
    activeCount,
    importFiles,
    uploadItems: items,
    dismissItem,
    handleFileEvent,
  };
}

function uploadStatusFromFile(file: ScriberrFile): UploadItemStatus {
  if (file.status === "processing") return "processing";
  if (file.status === "failed") return "failed";
  return "ready";
}

function isSupportedMediaFile(file: File) {
  if (file.type.startsWith("audio/") || file.type.startsWith("video/")) return true;
  const name = file.name.toLowerCase();
  return [...audioExtensions, ...videoExtensions].some((extension) => name.endsWith(extension));
}
