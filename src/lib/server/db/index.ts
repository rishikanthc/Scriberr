import { drizzle } from 'drizzle-orm/postgres-js';
import postgres from 'postgres';

// For build time, provide a dummy placeholder so the build doesn't fail
// At runtime, this will be replaced with the actual value
let DATABASE_URL = process.env.DATABASE_URL || 'postgres://placeholder:placeholder@db:5432/placeholder';

// If we have the component parts but not the URL, construct it (for Docker)
if (!process.env.DATABASE_URL && process.env.POSTGRES_USER && process.env.POSTGRES_PASSWORD && process.env.POSTGRES_DB) {
  DATABASE_URL = `postgres://${process.env.POSTGRES_USER}:${process.env.POSTGRES_PASSWORD}@db:5432/${process.env.POSTGRES_DB}`;
  console.log("Generated DATABASE_URL from components");
}

// Only throw the error at runtime (not during build)
if (process.env.NODE_ENV !== 'development' && !process.env.DATABASE_URL && process.env.RUNTIME_CHECK === 'true') {
  console.error("DATABASE_URL environment variable is not set!");
  console.error("Please make sure POSTGRES_USER, POSTGRES_PASSWORD, and POSTGRES_DB are set");
}

console.log(`Using database connection: ${DATABASE_URL.replace(/:[^:@]+@/, ':***@')}`);

// Initialize database connection
const client = postgres(DATABASE_URL);
export const db = drizzle(client);