export type FileStatus = "ready" | "uploaded" | "processing" | "failed" | "stopped" | "canceled";
export type FileKind = "audio" | "video" | "youtube" | "";

export type ScriberrFile = {
  id: string;
  title: string;
  kind: FileKind;
  status: FileStatus;
  mime_type: string;
  size_bytes: number;
  duration_seconds: number | null;
  created_at: string;
  updated_at: string;
};

export type FilesResponse = {
  items: ScriberrFile[];
  next_cursor: string | null;
};

export type UploadProgressHandler = (progress: number) => void;

export type UpdateFilePayload = {
  title: string;
};

export type ImportYouTubePayload = {
  url: string;
  title?: string;
};

async function readErrorMessage(response: Response, fallback: string) {
  try {
    const body = await response.json() as { error?: { message?: string } };
    return body.error?.message || fallback;
  } catch {
    return fallback;
  }
}

export function listFiles(headers: Record<string, string>): Promise<FilesResponse> {
  return fetch("/api/v1/files?limit=100&sort=-created_at", {
    headers,
  }).then(async (response) => {
    if (!response.ok) {
      throw new Error("Failed to load files");
    }
    return response.json() as Promise<FilesResponse>;
  });
}

export function getFile(fileId: string, headers: Record<string, string>): Promise<ScriberrFile> {
  return fetch(`/api/v1/files/${fileId}`, {
    headers,
  }).then(async (response) => {
    if (!response.ok) {
      throw new Error("Failed to load file");
    }
    return response.json() as Promise<ScriberrFile>;
  });
}

export async function updateFile(
  fileId: string,
  payload: UpdateFilePayload,
  headers: Record<string, string>
): Promise<ScriberrFile> {
  const response = await fetch(`/api/v1/files/${fileId}`, {
    method: "PATCH",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ title: payload.title }),
  });
  if (!response.ok) {
    throw new Error("Failed to update file");
  }
  return response.json() as Promise<ScriberrFile>;
}

export async function importYouTube(payload: ImportYouTubePayload, headers: Record<string, string>): Promise<ScriberrFile> {
  const response = await fetch("/api/v1/files:import-youtube", {
    method: "POST",
    headers: {
      ...headers,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      url: payload.url.trim(),
      title: payload.title?.trim() || undefined,
    }),
  });
  if (!response.ok) {
    throw new Error(await readErrorMessage(response, "Failed to import YouTube audio"));
  }
  return response.json() as Promise<ScriberrFile>;
}

export function uploadFile(
  file: File,
  headers: Record<string, string>,
  onProgress?: UploadProgressHandler
): Promise<ScriberrFile> {
  return new Promise((resolve, reject) => {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("title", file.name.replace(/\.[^/.]+$/, ""));

    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/v1/files");

    Object.entries(headers).forEach(([key, value]) => {
      xhr.setRequestHeader(key, value);
    });

    xhr.upload.onprogress = (event) => {
      if (!event.lengthComputable || !onProgress) return;
      onProgress(Math.round((event.loaded / event.total) * 100));
    };

    xhr.onload = () => {
      if (xhr.status < 200 || xhr.status >= 300) {
        reject(new Error("Upload failed"));
        return;
      }

      try {
        resolve(JSON.parse(xhr.responseText) as ScriberrFile);
      } catch {
        reject(new Error("Upload response was invalid"));
      }
    };

    xhr.onerror = () => reject(new Error("Upload failed"));
    xhr.onabort = () => reject(new Error("Upload canceled"));
    xhr.send(formData);
  });
}
