import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { desc } from 'drizzle-orm';
import { requireAuth } from '$lib/server/auth';

export const GET: RequestHandler = async ({ locals }) => {
    console.log("API AUDIOFILEA ---->")
    await requireAuth(locals);
    console.log("API AUDIOFILEA ----> finishing auth")

    try {
        const files = await db
            .select({
                id: audioFiles.id,
                fileName: audioFiles.fileName,
                duration: audioFiles.duration,
                title: audioFiles.title,
                transcriptionStatus: audioFiles.transcriptionStatus,
                summary: audioFiles.summary,
                language: audioFiles.language,
                uploadedAt: audioFiles.uploadedAt,
                transcribedAt: audioFiles.transcribedAt,
                diarization: audioFiles.diarization,
            })
            .from(audioFiles)
            .orderBy(desc(audioFiles.uploadedAt));
        return json(files);
    } catch (error) {
        console.error('Error fetching audio files:', error);
        return new Response('Failed to fetch audio files', { status: 500 });
    }
};

export const POST: RequestHandler = async ({ request, locals }) => {
    await requireAuth(locals);

    try {
        const { id, status } = await request.json();
        
        const [updatedFile] = await db
            .update(audioFiles)
            .set({
                transcriptionStatus: status,
                updatedAt: new Date(),
                ...(status === 'completed' ? { transcribedAt: new Date() } : {})
            })
            .where(audioFiles.id === id)
            .returning();
        return json(updatedFile);
    } catch (error) {
        console.error('Error updating audio file:', error);
        return new Response('Failed to update audio file', { status: 500 });
    }
};
