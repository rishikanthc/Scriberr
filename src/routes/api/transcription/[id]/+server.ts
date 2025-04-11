import { error } from '@sveltejs/kit';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { requireAuth } from '$lib/server/auth';

export const GET: RequestHandler = async ({ params, locals }) => {
  await requireAuth(locals);

  if (!params.id) {
    throw error(400, 'Missing ID parameter');
  }

  const audioFile = await db
    .select()
    .from(audioFiles)
    .where(eq(audioFiles.id, parseInt(params.id)))
    .then(rows => rows[0]);

  if (!audioFile) {
    throw error(404, 'File not found');
  }

  // Parse transcript if it's a string
  let transcript = audioFile.transcript;
  try {
    if (typeof transcript === 'string') {
      transcript = JSON.parse(transcript);
    }
  } catch (e) {
    console.error('Failed to parse transcript:', e);
    transcript = [];
  }

  return new Response(JSON.stringify({
    id: audioFile.id,
    fileName: audioFile.fileName,
    originalFileName: audioFile.originalFileName,
    originalFileType: audioFile.originalFileType,
    status: audioFile.transcriptionStatus,
    transcript: transcript,
    error: audioFile.lastError,
    updatedAt: audioFile.updatedAt
  }), {
    headers: {
      'Content-Type': 'application/json'
    }
  });
};