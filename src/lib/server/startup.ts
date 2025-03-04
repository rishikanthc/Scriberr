import { db } from '$lib/server/db';  // Your Drizzle instance
import { systemSettings } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';

export async function checkFirstStartup() {
  // Try to get settings
  const settings = await db.select().from(systemSettings).limit(1);
  
  if (settings.length === 0) {
    // First time ever - create settings record
    await db.insert(systemSettings).values({
      isInitialized: false,
      firstStartupDate: new Date(),
      lastStartupDate: new Date(),
      whisperModelSizes: [],
     });
    return false;
  }
  
  if (!settings[0].isInitialized) {
    // Update last startup date
    await db.update(systemSettings)
      .set({ lastStartupDate: new Date() })
      .where(eq(systemSettings.isInitialized, false));
    return false;
  }
  
  return true;
}
