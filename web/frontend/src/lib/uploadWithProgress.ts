import { useAuthStore } from '../features/auth/store/authStore';
import { refreshToken, navigateToHome } from './authHelpers';

export interface UploadProgressEvent {
    percent: number;
    loadedBytes: number;
    totalBytes: number;
    bytesPerSecond: number;
}

export type UploadProgressHandler = (progress: UploadProgressEvent) => void;

interface XhrResult {
    status: number;
    body: unknown;
}

function send(url: string, body: FormData, token: string | null, onProgress?: UploadProgressHandler): Promise<XhrResult> {
    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', url);
        if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`);
        xhr.responseType = 'json';
        const startedAt = performance.now();
        xhr.upload.onprogress = (event) => {
            if (!event.lengthComputable) return;
            // Average since start: per-event deltas jump around too much to read.
            const elapsedSeconds = (performance.now() - startedAt) / 1000;
            onProgress?.({
                percent: Math.round((event.loaded / event.total) * 100),
                loadedBytes: event.loaded,
                totalBytes: event.total,
                bytesPerSecond: elapsedSeconds > 0 ? event.loaded / elapsedSeconds : 0,
            });
        };
        xhr.onload = () => resolve({ status: xhr.status, body: xhr.response });
        xhr.onerror = () => reject(new Error('Network error during upload'));
        xhr.onabort = () => reject(new Error('Upload aborted'));
        xhr.send(body);
    });
}

function errorMessage(body: unknown): string {
    if (typeof body === 'object' && body !== null && 'error' in body && typeof body.error === 'string') {
        return body.error;
    }
    return 'Upload failed';
}

/** POSTs a form with upload progress. fetch cannot report upload progress, so this mirrors the auth interceptor over XHR. */
export async function uploadWithProgress(url: string, body: FormData, onProgress?: UploadProgressHandler): Promise<unknown> {
    const state = useAuthStore.getState();
    let result = await send(url, body, state.token, onProgress);

    if (result.status === 401) {
        const newToken = await refreshToken();
        if (newToken) result = await send(url, body, newToken, onProgress);
        if (result.status === 401) {
            state.logout();
            navigateToHome();
        }
    }

    if (result.status < 200 || result.status >= 300) throw new Error(errorMessage(result.body));
    return result.body;
}
