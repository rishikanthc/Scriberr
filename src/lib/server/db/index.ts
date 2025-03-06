import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';

// Use process.env directly instead of importing from $env
// This allows the build to succeed without environment variables
const dbUrl = process.env.DATABASE_URL || 'postgresql://placeholder:placeholder@localhost:5432/placeholder';

// Only check for required env variables at runtime, not during build
if (process.env.NODE_ENV === 'production' && !process.env.DATABASE_URL) {
  console.error('ERROR: DATABASE_URL is required in production mode');
}

const client = postgres(dbUrl);
export const db = drizzle(client);