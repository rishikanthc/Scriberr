import { writable } from 'svelte/store';

// Create an auth token store. Initially null means the user is not logged in.
export const authToken = writable<string | null>(null);
