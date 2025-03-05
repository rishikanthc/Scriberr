// routes/api/templates/[id]/+server.ts
import { json } from '@sveltejs/kit';
import { db } from '$lib/server/db';
import { summarizationTemplates } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';

export async function PATCH({ params, request }) {
  const data = await request.json();
  const [updated] = await db.update(summarizationTemplates)
    .set({
      ...data,
      updatedAt: new Date()
    })
    .where(eq(summarizationTemplates.id, params.id))
    .returning();
  return json(updated);
}

export async function DELETE({ params }) {
  await db.delete(summarizationTemplates)
    .where(eq(summarizationTemplates.id, params.id));
  return new Response(null, { status: 204 });
}
