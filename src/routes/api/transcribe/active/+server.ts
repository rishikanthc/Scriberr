import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { eq, or } from 'drizzle-orm';
import { requireAuth } from '$lib/server/auth';

export const GET: RequestHandler = async ({ locals }) => {
    await requireAuth(locals);

    try {
        // Fetch all files that are either pending or processing
        const files = await db
            .select({
                id: audioFiles.id,
                fileName: audioFiles.fileName,
                status: audioFiles.transcriptionStatus,
                progress: audioFiles.transcriptionProgress,
                error: audioFiles.lastError
            })
            .from(audioFiles)
            .where(
                or(
                    eq(audioFiles.transcriptionStatus, 'pending'),
                    eq(audioFiles.transcriptionStatus, 'processing')
                )
            );

        return new Response(JSON.stringify(files), {
            headers: {
                'Content-Type': 'application/json'
            }
        });
    } catch (err) {
        console.error('Error fetching active jobs:', err);
        throw error(500, 'Failed to fetch active jobs');
    }
};
