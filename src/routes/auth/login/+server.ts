// src/routes/auth/login/+server.ts
import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import * as auth from '$lib/server/auth';
import { user } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { db } from '$lib/server/db';
import { sha256 } from '@oslojs/crypto/sha2';
import { encodeHexLowerCase } from '@oslojs/encoding';

export const POST: RequestHandler = async ({ request, cookies }) => {
   const { username, password } = await request.json();
   const hashedPassword = encodeHexLowerCase(sha256(new TextEncoder().encode(password)));
   
   const [foundUser] = await db
       .select()
       .from(user)
       .where(
           eq(user.username, username) && 
           eq(user.passwordHash, hashedPassword)
       );

   if (!foundUser) {
       return json({ error: 'Invalid credentials' }, { status: 401 });
   }

   const sessionToken = auth.generateSessionToken();
   await auth.createSession(sessionToken, foundUser.id);

   cookies.set(auth.sessionCookieName, sessionToken, {
       path: '/',
       secure: true,
       httpOnly: true,
       sameSite: 'lax'
   });

   return json({ success: true });
};
