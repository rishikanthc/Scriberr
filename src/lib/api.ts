// src/lib/api.ts
import { Preferences } from '@capacitor/preferences';
import { authToken, serverUrl, isAuthenticated } from '$lib/stores/config';
import { get } from 'svelte/store';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';

const AUTH_ENDPOINTS = ['/api/auth', '/login'];

// Error indicating authorization failure that needs handling
export class AuthError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'AuthError';
  }
}

export async function apiFetch(endpoint: string, options: RequestInit = {}) {
  const baseUrl = get(serverUrl);
  const token = get(authToken);
  
  // Do not add auth headers for auth-related endpoints
  const isAuthEndpoint = AUTH_ENDPOINTS.some(authPath => endpoint.includes(authPath));
  
  const url = baseUrl ? `${baseUrl}${endpoint}` : endpoint;
  const headers = {
    ...options.headers,
    'Authorization': !isAuthEndpoint && token ? `Bearer ${token}` : '',
  };
  
  try {
    const response = await fetch(url, {
      ...options,
      headers
    });
    
    // Handle authentication errors
    if (response.status === 401 && !isAuthEndpoint) {
      // Clear authentication state
      if (browser) {
        // Clear in-memory token
        authToken.set('');
        isAuthenticated.set(false);
        
        // Clear from local storage
        localStorage.removeItem('sessionToken');
        localStorage.removeItem('sessionExpires');
        
        // Navigate to login
        goto('/login');
      }
      
      throw new AuthError('Authentication failed. Please log in again.');
    }
    
    return response;
  } catch (error) {
    // Rethrow AuthError instances
    if (error instanceof AuthError) {
      throw error;
    }
    
    // Handle network errors
    if (error instanceof Error && error.message.includes('fetch')) {
      console.error('Network error:', error);
      // Only redirect on client
      if (browser && !isAuthEndpoint) {
        goto('/login');
      }
      throw new AuthError('Network error. Please check your connection and try again.');
    }
    
    // Rethrow other errors
    throw error;
  }
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

// Check if the token is about to expire and refresh it if needed
export async function checkAndRefreshToken(): Promise<boolean> {
  if (!browser) return false;
  
  const expiresAtStr = localStorage.getItem('sessionExpires');
  if (!expiresAtStr) return false;
  
  const expiresAt = new Date(expiresAtStr).getTime();
  const now = Date.now();
  
  // If token expires in less than 1 day (86400000 ms), refresh it
  const shouldRefresh = expiresAt - now < 86400000 && expiresAt > now;
  
  if (shouldRefresh) {
    try {
      const response = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${get(authToken)}`
        }
      });
      
      if (response.ok) {
        const data = await response.json();
        if (data.token) {
          // Update token in localStorage and stores
          localStorage.setItem('sessionToken', data.token);
          localStorage.setItem('sessionExpires', data.expiresAt);
          authToken.set(data.token);
          return true;
        }
      }
    } catch (error) {
      console.error('Failed to refresh token:', error);
    }
  }
  
  return false;
}