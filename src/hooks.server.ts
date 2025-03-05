import type { Handle } from '@sveltejs/kit';
import { redirect } from '@sveltejs/kit';
import * as auth from '$lib/server/auth.js';
import { building } from '$app/environment';
import { startWorker } from '$lib/server/worker';
import { setupTranscriptionWorker } from '$lib/server/queue';

const PUBLIC_PATHS = ['/login', '/api/setup', '/api/complete-setup', '/api/auth', '/api/verify', '/api/check-config'];
const ALLOWED_ORIGINS = [
    'capacitor://localhost', 
    'http://localhost:5173',
    'http://localhost',
    'http://localhost:*',
    'http://127.0.0.1:5173',
    'http://127.0.0.1',
    'capacitor://127.0.0.1',
    'http://your-frontend-domain.com'
];

// Static paths that should always show a 401 error for API, not redirect
const API_PATHS = ['/api/'];

if (!building) {
    // startWorker().catch(console.error);
    await setupTranscriptionWorker().catch(console.error);
    console.log("STARTED WORKER -->")
}

const handleAuth: Handle = async ({ event, resolve }) => {
    console.log('[Hooks] Incoming request:', {
        path: event.url.pathname,
        method: event.request.method,
        origin: event.request.headers.get('origin'),
        host: event.request.headers.get('host'),
        referer: event.request.headers.get('referer')
    });

    // Skip auth check during build time
    if (building) {
        console.log('[Hooks] Skipping auth check during build time');
        return resolve(event);
    }

    const requestOrigin = event.request.headers.get('origin');
    console.log('[Hooks] Request origin:', requestOrigin);
    
    // Handle CORS
    let corsHeaders = {};
    if (requestOrigin) {
        const isAllowed = ALLOWED_ORIGINS.some(allowed => {
            if (allowed.includes('*')) {
                const pattern = new RegExp('^' + allowed.replace('*', '.*') + '$');
                return pattern.test(requestOrigin);
            }
            return allowed === requestOrigin;
        });

        if (isAllowed) {
            corsHeaders = {
                'Access-Control-Allow-Origin': requestOrigin,
                'Access-Control-Allow-Methods': 'GET, PATCH, POST, PUT, DELETE, OPTIONS',
                'Access-Control-Allow-Headers': 'Content-Type, Authorization, X-Requested-With',
                'Access-Control-Allow-Credentials': 'true',
                'Vary': 'Origin'
            };
        }
    }

    if (event.request.method === 'OPTIONS') {
        return new Response(null, {
            status: 204,
            headers: {
                ...corsHeaders,
                'Access-Control-Max-Age': '3600',
                'Access-Control-Allow-Headers': 'Content-Type, Authorization, X-Requested-With'
            }
        });
    }

    const path = event.url.pathname;
    const isPublicPath = PUBLIC_PATHS.some((p) => path.startsWith(p));
    const isApiPath = API_PATHS.some((p) => path.startsWith(p));
    
    // Check authentication in this order: Bearer token, URL token (for EventSource), Cookie
    let authenticated = false;

    // 1. Check Bearer token
    const authHeader = event.request.headers.get('authorization');
    if (authHeader?.startsWith('Bearer ')) {
        const token = authHeader.slice(7);
        try {
            const { session, user } = await auth.validateSessionToken(token);
            if (session && user) {
                event.locals.user = user;
                event.locals.session = session;
                authenticated = true;
            }
        } catch (error) {
            console.error('[Hooks] Bearer token validation error:', error);
        }
    }

    // 2. Check URL token (for EventSource)
    if (!authenticated) {
        const urlToken = event.url.searchParams.get('token');
        if (urlToken) {
            console.log('[Hooks] Found URL token, validating...');
            try {
                const { session, user } = await auth.validateSessionToken(urlToken);
                if (session && user) {
                    event.locals.user = user;
                    event.locals.session = session;
                    authenticated = true;
                    console.log('[Hooks] URL token validated successfully');
                }
            } catch (error) {
                console.error('[Hooks] URL token validation error:', error);
            }
        }
    }

    // 3. Check cookie
    if (!authenticated) {
        const sessionToken = event.cookies.get(auth.sessionCookieName);
        if (sessionToken) {
            try {
                const { session, user } = await auth.validateSessionToken(sessionToken);
                if (session && user) {
                    event.locals.user = user;
                    event.locals.session = session;
                    auth.setSessionTokenCookie(event, sessionToken, session.expiresAt);
                    authenticated = true;
                } else if (!isPublicPath) {
                    auth.deleteSessionTokenCookie(event);
                }
            } catch (error) {
                console.error('[Hooks] Cookie validation error:', error);
                // Explicitly clear cookie on validation error
                auth.deleteSessionTokenCookie(event);
            }
        }
    }

    // Handle unauthenticated requests to protected paths
    if (!authenticated && !isPublicPath) {
        console.log('[Hooks] Unauthenticated request to protected path:', path);
        
        // API requests return 401 Unauthorized
        if (isApiPath || path.startsWith('/api/')) {
            return new Response(JSON.stringify({ 
                error: 'Unauthorized',
                code: 'AUTH_REQUIRED',
                message: 'Authentication required'
            }), {
                status: 401,
                headers: {
                    ...corsHeaders,
                    'Content-Type': 'application/json'
                }
            });
        }
        
        // Non-API requests redirect to login
        throw redirect(303, '/login');
    }

    // Continue processing the request
    const response = await resolve(event);
    
    // Add CORS headers to all responses
    Object.entries(corsHeaders).forEach(([key, value]) => {
        response.headers.set(key, value);
    });
    
    return response;
};

export const handle: Handle = handleAuth;