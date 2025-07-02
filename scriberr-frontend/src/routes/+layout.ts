// We need to disable server-side rendering for the root layout to ensure
// our client-side authentication checks work correctly. The check depends
// on browser APIs (fetch with cookies) to determine the user's session status.
export const ssr = false;

// The root layout itself is static and can be pre-rendered.
// Child pages will not be prerendered unless they specifically opt in.
export const prerender = true;
