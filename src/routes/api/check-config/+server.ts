import { json } from '@sveltejs/kit';
import { db } from '$lib/server/db';
import { systemSettings } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async () => {
  console.log("API: /api/check-config endpoint called");
  
  try {
    // Check if database is working
    const settings = await db.select().from(systemSettings).limit(1);
    console.log("API: Database connection successful");
    
    const isConfigured = settings.length > 0 && settings[0].isInitialized === true;
    console.log("API: System configuration check result:", { isConfigured });
    
    // Add debug information to help diagnose issues
    return json({
      isConfigured,
      debug: {
        timestamp: new Date().toISOString(),
        connectionWorking: true,
        settingsFound: settings.length > 0,
        settingsValue: settings.length > 0 ? { 
          isInitialized: settings[0].isInitialized,
          id: settings[0].id
        } : null
      }
    });
  } catch (error) {
    console.error("API: Database error during config check:", error);
    return json({
      isConfigured: false,
      error: error instanceof Error ? error.message : "Unknown error",
      debug: {
        timestamp: new Date().toISOString(),
        connectionWorking: false
      }
    }, { status: 500 });
  }
};