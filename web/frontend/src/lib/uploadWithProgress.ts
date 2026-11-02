import { useAuthStore } from '../features/auth/store/authStore';
import { refreshToken, navigateToHome } from './authHelpers';

export type UploadProgressHandler = (percent: number) => void;

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
        xhr.upload.onprogress = (event) => {
            if (event.lengthComputable) onProgress?.(Math.round((event.loaded / event.total) * 100));
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
