import PocketBase from 'pocketbase';
import { env } from '$env/dynamic/private';
import type { Handle } from '@sveltejs/kit';
import { sequence } from '@sveltejs/kit/hooks';
import { redirect } from '@sveltejs/kit';
import '$lib/queue';

export const authentication: Handle = async ({ event, resolve }) => {
	event.locals.pb = new PocketBase('http://localhost:8080');
	event.locals.pb.autoCancellation(false);
	// Load the store data from the request cookie string
	const cookieHeader = event.request.headers.get('cookie') || '';
	event.locals.pb.authStore.loadFromCookie(cookieHeader, 'pb_auth');
	console.log('Auth state after loading cookie:', event.locals.pb.authStore.isValid);
	console.log('Auth token:', event.locals.pb.authStore.token);

	try {
		// Get an up-to-date auth store state by verifying and refreshing the loaded auth model (if any)
		if (!event.locals.pb.authStore.isValid) {
			await event.locals.pb.collection('users').authRefresh();
			console.log('Auth state after refresh:', event.locals.pb.authStore.isValid);
		}
	} catch (err) {
		// Clear the auth store on failed refresh
		console.error('Auth refresh failed trying to login');
		console.log('admin email: ', env.POCKETBASE_ADMIN_EMAIL);
		console.log('admin password: ', env.POCKETBASE_ADMIN_PASSWORD);

		try {
			await event.locals.pb.admins.authWithPassword(
				env.POCKETBASE_ADMIN_EMAIL,
				env.POCKETBASE_ADMIN_PASSWORD
			);
		} catch (err) {
			console.error('login failed:', err);
			event.locals.pb.authStore.clear();
		}
	}

	const response = await resolve(event);

	// Send back the auth cookie to the client with the latest store state
	const cookie = event.locals.pb.authStore.exportToCookie({
		httpOnly: true,
		secure: process.env.NODE_ENV === 'production',
		sameSite: 'lax',
		path: '/'
	});
	response.headers.append('Set-Cookie', cookie);

	return response;
};

export const handle = authentication;
