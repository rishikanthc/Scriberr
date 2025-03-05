import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import * as auth from '$lib/server/auth';

export const POST: RequestHandler = async ({ request, locals }) => {
  try {
    // Get bearer token from the request
    const authHeader = request.headers.get('authorization');
    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      return new Response(JSON.stringify({ error: 'Invalid authentication header' }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' }
      });
    }

    const token = authHeader.slice(7);
    
    // Validate the token and get user information
    const { session, user } = await auth.validateSessionToken(token);
    
    if (!session || !user) {
      return new Response(JSON.stringify({ error: 'Invalid token' }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' }
      });
    }
    
    // Generate a new token
    const newToken = auth.generateSessionToken();
    
    // Create a new session with this token
    const newSession = await auth.createSession(newToken, user.id);
    
    // Return the new token and expiration time
    return json({
      token: newToken,
      expiresAt: newSession.expiresAt.toISOString(),
      user: {
        username: user.username,
        isAdmin: user.isAdmin
      }
    });
  } catch (error) {
    console.error('Token refresh error:', error);
    return new Response(JSON.stringify({ error: 'Internal server error' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' }
    });
  }
};