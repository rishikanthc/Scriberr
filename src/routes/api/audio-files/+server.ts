import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { desc } from 'drizzle-orm';
import { requireAuth, checkSetupStatus } from '$lib/server/auth';

export const GET: RequestHandler = async ({ locals }) => {
    console.log("API AUDIO FILES ---->")
    
    try {
        // Skip auth check if system is not yet set up
        const isSetupComplete = await checkSetupStatus().catch(() => false);
        
        if (!isSetupComplete) {
            console.log("System not initialized yet");
            return json([]);
        }
        
        // Proceed with auth check
        try {
            await requireAuth(locals);
        } catch (error) {
            console.log("Auth check failed, returning empty list:", error);
            return json([]);
        }
        
        console.log("API AUDIO FILES ----> finishing auth")

        const files = await db
            .select({
                id: audioFiles.id,
                fileName: audioFiles.fileName,
                originalFileName: audioFiles.originalFileName,
                originalFileType: audioFiles.originalFileType,
                duration: audioFiles.duration,
                title: audioFiles.title,
                transcriptionStatus: audioFiles.transcriptionStatus,
                summary: audioFiles.summary,
                language: audioFiles.language,
                uploadedAt: audioFiles.uploadedAt,
                transcribedAt: audioFiles.transcribedAt,
                diarization: audioFiles.diarization,
                peaks: audioFiles.peaks,
            })
            .from(audioFiles)
            .orderBy(desc(audioFiles.uploadedAt));
            
        return json(files || []);
    } catch (error) {
        console.error('Error fetching audio files:', error);
        // Return empty array instead of error to prevent client-side errors
        return json([]);
    }
};

export const POST: RequestHandler = async ({ request, locals }) => {
    // Skip auth check during setup
    const isSetupComplete = await checkSetupStatus().catch(() => false);
    if (isSetupComplete) {
        try {
            await requireAuth(locals);
        } catch (error) {
            return new Response('Unauthorized', { status: 401 });
        }
    }

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