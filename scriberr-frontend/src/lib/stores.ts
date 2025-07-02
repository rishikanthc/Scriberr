import { writable } from 'svelte/store';
import { browser } from '$app/environment';

// Create a writable store with an initial value of false.
// This store will hold the authentication status of the user.
export const isAuthenticated = writable<boolean>(false);

// Function to initialize the authentication status.
// It checks with the backend API to see if the user has a valid session.
export async function checkAuthStatus(): Promise<void> {
	// This should only run in the browser environment.
	if (!browser) {
		return;
	}

	try {
		const controller = new AbortController();
		const timeoutId = setTimeout(() => controller.abort(), 5000);

		const response = await fetch('/api/auth/status', { 
			credentials: 'include',
			signal: controller.signal
		});
		
		clearTimeout(timeoutId);
		
		if (response.ok) {
			const data = await response.json();
			isAuthenticated.set(data.authenticated === true);
		} else {
			// Any non-OK response (e.g., 401 Unauthorized) means not authenticated.
			isAuthenticated.set(false);
		}
	} catch (error) {
		console.error('Failed to check authentication status:', error);
		// If there's a network error or the server is down, assume not authenticated.
		isAuthenticated.set(false);
	}
}

// Function to check authentication with redirect handling.
// This is more reliable for route protection.
export async function checkAuthWithRedirect(): Promise<boolean> {
	// This should only run in the browser environment.
	if (!browser) {
		return false;
	}

	try {
		const controller = new AbortController();
		const timeoutId = setTimeout(() => controller.abort(), 5000);

		const response = await fetch('/api/auth/check', { 
			credentials: 'include',
			signal: controller.signal
		});
		
		clearTimeout(timeoutId);
		
		if (response.ok) {
			const data = await response.json();
			const authenticated = data.authenticated === true;
			isAuthenticated.set(authenticated);
			return authenticated;
		} else {
			// Check if there's a redirect response
			try {
				const data = await response.json();
				if (data.redirect) {
					// Redirect to login
					window.location.href = data.redirect;
					return false;
				}
			} catch (e) {
				// Ignore JSON parsing errors
			}
			
			isAuthenticated.set(false);
			return false;
		}
	} catch (error) {
		console.error('Failed to check authentication status:', error);
		isAuthenticated.set(false);
		return false;
	}
}
