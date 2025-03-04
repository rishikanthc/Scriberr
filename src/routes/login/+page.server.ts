// routes/login/+page.server.ts
import type { Actions } from './$types';
import * as auth from '$lib/server/auth';
import { fail, redirect } from '@sveltejs/kit';

export const prerender = false;

export const actions: Actions = {
    default: async (event) => {
        const data = await event.request.formData();
        const username = data.get('username');
        const password = data.get('password');

        if (!username || !password) {
            return fail(400, { success: false });
        }

        try {
            const { token, session, user } = await auth.login(username.toString(), password.toString());
            auth.setSessionTokenCookie(event, token, session.expiresAt);
            return { success: true, token, expiresAt: session.expiresAt };
        } catch (error) {
            console.error('Login error:', error);
            return fail(400, { success: false });
        }
    }
};
