// routes/api/templates/+server.ts
import { json } from '@sveltejs/kit';
import { db } from '$lib/server/db';
import { summarizationTemplates } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';

export async function GET() {
  const templates = await db.select().from(summarizationTemplates);
  return json(templates);
}

export async function POST({ request }) {
  const data = await request.json();
  const [template] = await db.insert(summarizationTemplates)
    .values({
      title: data.title,
      prompt: data.prompt
    })
    .returning();
  return json(template);
}
