// The server injects <base href> into index.html; the Vite dev server serves at the root.
export const BASE_PATH = import.meta.env.DEV ? '' : new URL(document.baseURI).pathname.replace(/\/$/, '');

/** Prefixes a root-relative path ("/api/...") with BASE_PATH. Safe to apply twice. */
export function withBasePath(path: string): string {
    if (!path.startsWith('/') || path.startsWith('//')) return path;
    if (BASE_PATH && (path === BASE_PATH || path.startsWith(BASE_PATH + '/'))) return path;
    return BASE_PATH + path;
}
