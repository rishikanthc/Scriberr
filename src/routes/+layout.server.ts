// routes/+layout.server.ts
import { redirect } from '@sveltejs/kit';
import { checkFirstStartup } from '$lib/server/startup';
export const prerender = false;

export async function load({ url, fetch }) {
    // If DATABASE_URL is not set, skip DB-dependent startup checks.
    if (!process.env.DATABASE_URL) {
        console.log('DATABASE_URL not set. Skipping startup configuration check.');
        return { needsConfiguration: false };
    }

    try {
        // Use the same API endpoint as the login page to check configuration
        const response = await fetch('/api/check-config');
        const { isConfigured } = await response.json();
        const needsConfiguration = !isConfigured;
        
        // If we're on the debug page, don't redirect
        if (url.pathname === '/DebugConfigWizard') {
            return { needsConfiguration };
        }
        
        // Check if we need to show the setup wizard
        if (needsConfiguration && url.pathname !== '/') {
            // Redirect to root where the ConfigWizard is shown 
            throw redirect(303, '/');
        }
        
        return { needsConfiguration };
    } catch (error) {
        console.error('Error checking system configuration:', error);
        return { needsConfiguration: false, error: 'Failed to check configuration status' };
    }
}