// routes/api/check-configuration/+server.ts
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { checkFirstStartup } from '$lib/server/startup';
// import { requireAuth } from '$lib/server/auth';

export const GET: RequestHandler = async ({request, locals}) => {
  // await requireAuth(locals);
  try {
   const isConfigured = await checkFirstStartup();
   return json({ isConfigured });
  } catch (error) {
   return json(
     { error: error instanceof Error ? error.message : 'Unknown error' }, 
     { status: 500 }
   );
  }
};
