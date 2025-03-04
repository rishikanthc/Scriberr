// routes/api/complete-setup/+server.ts
import { db } from '$lib/server/db';
import { systemSettings } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { json } from '@sveltejs/kit';

export async function POST({ request }) {
  const config = await request.json();
  
  // Save configuration settings
  // ... your config saving logic here ...
  
  // Mark system as initialized
  await db.update(systemSettings)
    .set({ 
      isInitialized: true,
      lastStartupDate: new Date()
    })
    .where(eq(systemSettings.isInitialized, false));
  
  return json({ success: true });
}
