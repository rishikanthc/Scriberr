// src/lib/stores/config.ts
import { writable } from 'svelte/store';
import { Preferences } from '@capacitor/preferences';

const KEYS = {
    SERVER_URL: 'server_url',
    USERNAME: 'username',
    PASSWORD: 'password',
    AUTH_TOKEN: 'auth_token'
} as const;

export const serverUrl = writable<string>('');
export const isAuthenticated = writable<boolean>(false);
export const authToken = writable<string>('');

interface ServerConfig {
    url: string;
    username: string;
    password: string;
}

interface AuthResponse {
    accessToken: string;
    user: {
        username: string;
        isAdmin: boolean;
    };
}

async function authenticate(config: ServerConfig): Promise<AuthResponse> {
    const response = await fetch(`${config.url}/api/auth`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            username: config.username,
            password: config.password
        })
    });

    if (!response.ok) {
        throw new Error('Authentication failed');
    }

    return response.json();
}

export async function initializeServerConfig(): Promise<Partial<ServerConfig> | null> {
    try {
        const [urlResult, usernameResult, passwordResult, tokenResult] = await Promise.all([
            Preferences.get({ key: KEYS.SERVER_URL }),
            Preferences.get({ key: KEYS.USERNAME }),
            Preferences.get({ key: KEYS.PASSWORD }),
            Preferences.get({ key: KEYS.AUTH_TOKEN })
        ]);

        const config: Partial<ServerConfig> = {};
        
        if (urlResult.value) {
            config.url = urlResult.value;
            serverUrl.set(urlResult.value);
        }
        if (usernameResult.value) config.username = usernameResult.value;
        if (passwordResult.value) config.password = passwordResult.value;
        if (tokenResult.value) {
            authToken.set(tokenResult.value);
            isAuthenticated.set(true);
        }

        return config.url && config.username && config.password ? config : null;
    } catch (error) {
        console.error('Failed to get server config:', error);
        return null;
    }
}

export async function setServerConfig(config: ServerConfig): Promise<void> {
    try {
        const authResponse = await authenticate(config);
        
        await Promise.all([
            Preferences.set({ key: KEYS.SERVER_URL, value: config.url }),
            Preferences.set({ key: KEYS.USERNAME, value: config.username }),
            Preferences.set({ key: KEYS.PASSWORD, value: config.password }),
            Preferences.set({ key: KEYS.AUTH_TOKEN, value: authResponse.accessToken })
        ]);
        
        serverUrl.set(config.url);
        authToken.set(authResponse.accessToken);
        isAuthenticated.set(true);
    } catch (error) {
        console.error('Failed to save server config:', error);
        throw error;
    }
}

export async function clearServerConfig(): Promise<void> {
    try {
        await Promise.all([
            Preferences.remove({ key: KEYS.SERVER_URL }),
            Preferences.remove({ key: KEYS.USERNAME }),
            Preferences.remove({ key: KEYS.PASSWORD }),
            Preferences.remove({ key: KEYS.AUTH_TOKEN })
        ]);
        
        serverUrl.set('');
        authToken.set('');
        isAuthenticated.set(false);
    } catch (error) {
        console.error('Failed to clear server config:', error);
        throw error;
    }
}

export async function validateServerConfig(config: ServerConfig): Promise<boolean> {
    try {
        const authResponse = await authenticate(config);
        return !!authResponse.accessToken;
    } catch (error) {
        console.error('Failed to validate server config:', error);
        return false;
    }
}
