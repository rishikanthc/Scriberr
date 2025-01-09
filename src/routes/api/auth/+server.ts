// src/routes/api/auth/+server.ts
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import * as auth from '$lib/server/auth';

export const OPTIONS: RequestHandler = async () => {
    return new Response(null, {
        headers: {
            'Access-Control-Allow-Origin': '*',
            'Access-Control-Allow-Methods': 'POST, OPTIONS',
            'Access-Control-Allow-Headers': 'Content-Type'
        }
    });
};

export const POST: RequestHandler = async ({ request }) => {
    // Add CORS headers to POST response
    const headers = {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Methods': 'POST, OPTIONS',
        'Access-Control-Allow-Headers': 'Content-Type'  
    };
    
    try {
        const body = await request.json();
        const { username, password } = body;

        if (!username || !password) {
            return json({ error: 'Username and password are required' }, { status: 400 });
        }

        try {
            // Use existing login function
            const { user, session, token } = await auth.login(username, password);

            console.log("Logged in successfully")

            // Return the access token for mobile app
            return json({ 
                accessToken: token,
                user: {
                    username: user.username,
                    isAdmin: user.isAdmin
                }
            });
        } catch (error) {
            console.error('Login error:', error);
            return json({ error: 'Invalid credentials' }, { status: 401 });
        }
    } catch (error) {
        console.error('Auth error:', error);
        return json({ error: 'Internal server error' }, { status: 500 });
    }
};
