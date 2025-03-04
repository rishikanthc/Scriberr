// src/routes/api/auth/logout/+server.ts
import { json } from '@sveltejs/kit';
import { logout } from '$lib/server/auth';
import type { RequestEvent } from '@sveltejs/kit';

export async function POST(event: RequestEvent) {
    try {
        await logout(event);
        return json({ success: true });
    } catch (error) {
        console.error('Logout endpoint error:', error);
        return json({ error: 'Logout failed' }, { status: 500 });
    }
}
