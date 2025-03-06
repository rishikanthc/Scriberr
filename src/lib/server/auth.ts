import type { RequestEvent } from '@sveltejs/kit';
import { eq } from 'drizzle-orm';
import { sha256 } from '@oslojs/crypto/sha2';
import { encodeBase64url, encodeHexLowerCase } from '@oslojs/encoding';
import { db } from '$lib/server/db';
import * as table from '$lib/server/db/schema';

// Use process.env directly instead of importing from $env modules
// This allows the build to succeed without environment variables
const ADMIN_USERNAME = process.env.ADMIN_USERNAME;
const ADMIN_PASSWORD = process.env.ADMIN_PASSWORD;

const DAY_IN_MS = 1000 * 60 * 60 * 24;
export const sessionCookieName = 'auth-session';

async function setupAdminAccount() {
    const adminUsername = ADMIN_USERNAME;
    const adminPassword = ADMIN_PASSWORD;

    // Only run this in production or if credentials are set
    if (process.env.NODE_ENV !== 'production' && (!adminUsername || !adminPassword)) {
        console.log('Admin credentials not set, skipping admin account setup');
        return;
    }

    if (process.env.NODE_ENV === 'production' && (!adminUsername || !adminPassword)) {
        console.error('Admin credentials required in production but not set');
        return;
    }

    const [existingAdmin] = await db
        .select()
        .from(table.user)
        .where(eq(table.user.username, adminUsername));

    if (!existingAdmin) {
        const passwordHash = encodeHexLowerCase(sha256(new TextEncoder().encode(adminPassword)));
        const [user] = await db.insert(table.user).values({
            username: adminUsername,
            passwordHash,
            isAdmin: true,
        }).returning();
        return user;
    }
}

if (process.env.DATABASE_URL && process.env.NODE_ENV !== 'build') {
    setupAdminAccount().catch(console.error);
} else {
    console.log('Skipping admin account setup because DATABASE_URL is not set or build is in progress.');
}

export function generateSessionToken() {
    const bytes = crypto.getRandomValues(new Uint8Array(18));
    return encodeBase64url(bytes);
}

export async function login(username: string, password: string) {
    console.log('Login attempt:', { username });
    
    const passwordHash = encodeHexLowerCase(sha256(new TextEncoder().encode(password)));
    
    const [user] = await db
        .select()
        .from(table.user)
        .where(eq(table.user.username, username));
    
    if (!user || user.passwordHash !== passwordHash) {
        throw new Error('Invalid credentials');
    }

    try {
        const token = generateSessionToken();
        console.log('Generated token:', token);

        const session = await createSession(token, user.id);
        console.log('Created session:', session);

        return { user, session, token };
    } catch (error) {
        console.error('Session creation error:', error);
        throw error;
    }
}

export async function createSession(token: string, userId: string) {
    const sessionId = encodeHexLowerCase(sha256(new TextEncoder().encode(token)));
    console.log('Creating session:', { sessionId, userId });

    const session = {
        id: sessionId,
        userId,
        expiresAt: new Date(Date.now() + DAY_IN_MS * 30)
    };

    const [createdSession] = await db.insert(table.session)
        .values(session)
        .returning();

    return createdSession;
}

export async function validateSessionToken(token: string) {
    const sessionId = encodeHexLowerCase(sha256(new TextEncoder().encode(token)));
    const [result] = await db
        .select({
            user: { id: table.user.id, username: table.user.username, isAdmin: table.user.isAdmin },
            session: table.session
        })
        .from(table.session)
        .innerJoin(table.user, eq(table.session.userId, table.user.id))
        .where(eq(table.session.id, sessionId));

    if (!result) {
        return { session: null, user: null };
    }

    const { session, user } = result;
    const sessionExpired = Date.now() >= session.expiresAt.getTime();

    if (sessionExpired) {
        await invalidateSession(session.id);
        return { session: null, user: null };
    }

    const renewSession = Date.now() >= session.expiresAt.getTime() - DAY_IN_MS * 15;
    if (renewSession) {
        session.expiresAt = new Date(Date.now() + DAY_IN_MS * 30);
        await db
            .update(table.session)
            .set({ expiresAt: session.expiresAt })
            .where(eq(table.session.id, session.id));
    }

    return { session, user };
}

export async function invalidateSession(sessionId: string) {
    await db.delete(table.session).where(eq(table.session.id, sessionId));
}

export function setSessionTokenCookie(event: RequestEvent, token: string, expiresAt: Date) {
    event.cookies.set(sessionCookieName, token, {
        expires: expiresAt,
        path: '/',
        httpOnly: true,
        secure: true,
        sameSite: 'lax'
    });
}

export function deleteSessionTokenCookie(event: RequestEvent) {
    event.cookies.delete(sessionCookieName, { path: '/' });
}

// Check if setup is completed
export async function checkSetupStatus() {
    try {
        const settings = await db.select().from(table.systemSettings).limit(1);
        return settings.length > 0 && settings[0].isInitialized === true;
    } catch (error) {
        console.error('Error checking setup status:', error);
        return false;
    }
}

export async function requireAuth(locals: any) {
    // Check setup status first before auth
    const isSetupComplete = await checkSetupStatus();
    if (!isSetupComplete) {
        // Allow access to setup page without auth
        if (locals.url?.pathname === '/setup') {
            return { user: null, session: null };
        }
        throw Error('Setup required');
    }

    const user = locals.user;
    const session = locals.session;
    const urlToken = locals.url?.searchParams?.get('token');
    
    // If we have a valid session already populated (from hooks), use that
    if (session && user) {
        return { user, session };
    }
    
    // Handle URL token from EventSource
    if (urlToken) {
        try {
            const { user: verifiedUser, session: verifiedSession } = await validateSessionToken(urlToken);
            
            if (verifiedSession && verifiedUser) {
                return {
                    user: verifiedUser,
                    session: verifiedSession
                };
            }
        } catch (error) {
            console.error('Token validation failed:', error);
        }
    }
    
    throw Error('Unauthorized');
}

export async function requireAdmin(locals) {
    const { user } = await requireAuth(locals);
    
    if (!user.isAdmin) {
        throw Error('Forbidden');
    }
    
    return user;
}

export async function logout(event: RequestEvent) {
    try {
        // Get the session token from cookies
        const token = event.cookies.get(sessionCookieName);
        
        if (token) {
            // Get session ID from token
            const sessionId = encodeHexLowerCase(sha256(new TextEncoder().encode(token)));

            // Delete session from database
            await invalidateSession(sessionId);

            // Clear the session cookie
            event.cookies.delete(sessionCookieName, {
                path: '/',
                httpOnly: true,
                secure: true,
                sameSite: 'lax'
            });
        }
        
        return { success: true };
    } catch (error) {
        console.error('Logout error:', error);
        throw error;
    }
}