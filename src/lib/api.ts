// src/lib/api.ts
import { Preferences } from '@capacitor/preferences';
import { authToken, serverUrl } from '$lib/stores/config';
import { get } from 'svelte/store';

export async function apiFetch(endpoint: string, options: RequestInit = {}) {
  const baseUrl = get(serverUrl);
  const token = get(authToken);
  
  const url = baseUrl ? `${baseUrl}${endpoint}` : endpoint;
  const headers = {
    ...options.headers,
    'Authorization': token ? `Bearer ${token}` : '',
  };
  return fetch(url, {
    ...options,
    headers
  });
}

// Get the stored server URL
export async function getServerUrl() {
    const { value } = await Preferences.get({ key: 'server_url' });
    if (!value) {
      return '';
    } else {
      return value;
    }
}

export async function createEventSource(path: string): Promise<EventSource> {
    const baseUrl = await getServerUrl();
    const token = get(authToken);
    
    // Ensure path starts with '/'
    const cleanPath = path.startsWith('/') ? path : '/' + path;
    
    // Create URL with auth token
    const url = `${baseUrl}${cleanPath}?token=${token}`;
    
    return new EventSource(url);
}
