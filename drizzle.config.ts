import { defineConfig } from 'drizzle-kit';

// Check if we're in production mode
const isProduction = process.env.NODE_ENV === 'production';

// In production, DATABASE_URL is required
if (isProduction && !process.env.DATABASE_URL) {
  throw new Error('DATABASE_URL is required in production mode');
}

// Default to a placeholder URL during development/build
// At runtime in production, the real DATABASE_URL will be used
const dbUrl = process.env.DATABASE_URL || 'postgresql://placeholder:placeholder@localhost:5432/placeholder';

export default defineConfig({
	schema: './src/lib/server/db/schema.ts',

	dbCredentials: {
		url: dbUrl
	},

	verbose: true,
	strict: false,
	dialect: 'postgresql'
});