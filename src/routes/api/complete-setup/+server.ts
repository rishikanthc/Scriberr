// routes/api/complete-setup/+server.ts
import { db } from '$lib/server/db';
import { systemSettings } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { json } from '@sveltejs/kit';

export async function POST({ request }) {
  const config = await request.json();
  
  try {
    console.log("Completing setup and marking system as initialized");
    
    // First check if settings exist, if not create one
    const existingSettings = await db.select().from(systemSettings).limit(1);
    
    if (existingSettings.length === 0) {
      console.log("No settings found, creating new settings record");
      // Create default settings
      await db.insert(systemSettings).values({
        isInitialized: true,
        firstStartupDate: new Date(),
        lastStartupDate: new Date(),
        whisperModelSizes: [],
      });
      console.log("Created new settings record");
    } else {
      console.log("Settings found, updating existing record");
      // Update existing settings
      await db.update(systemSettings)
        .set({ 
          isInitialized: true,
          lastStartupDate: new Date()
        })
        .where(eq(systemSettings.id, existingSettings[0].id));
      console.log("Updated settings record");
    }
    
    return json({ success: true });
  } catch (error) {
    console.error("Error completing setup:", error);
    return json({ 
      success: false, 
      error: error instanceof Error ? error.message : 'Unknown error' 
    }, { status: 500 });
  }
}