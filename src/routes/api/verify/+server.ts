// src/routes/api/verify/+server.ts
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import * as auth from '$lib/server/auth';

export const GET: RequestHandler = async ({ request }) => {
    try {
        const authHeader = request.headers.get('authorization');
        
        if (!authHeader?.startsWith('Bearer ')) {
            return json({ error: 'Missing or invalid authorization header' }, { status: 401 });
        }

        const token = authHeader.slice(7);

        try {
            // Use existing token validation
            const { session, user } = await auth.validateSessionToken(token);

            if (!session || !user) {
                return json({ error: 'Invalid or expired token' }, { status: 401 });
            }

            return json({ 
                valid: true,
                user: {
                    username: user.username,
                    isAdmin: user.isAdmin
                }
            });
        } catch (error) {
            return json({ error: 'Invalid token' }, { status: 401 });
        }
    } catch (error) {
        console.error('Verify error:', error);
        return json({ error: 'Internal server error' }, { status: 500 });
    }
};
