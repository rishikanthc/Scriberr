import { error } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { transcribeAudio } from '$lib/server/transcription';
import { TranscribeStream } from '$lib/server/transcribeStream';
import { requireAuth } from '$lib/server/auth';
import { jobQueue } from '$lib/server/jobQueue';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';

export const GET: RequestHandler = async ({ params, locals }) => {
    await requireAuth(locals);
    
    const fileId = parseInt(params.id);
    if (isNaN(fileId)) throw error(400, 'Invalid file ID');

    // Get file status from database
    const file = await db.select().from(audioFiles).where(eq(audioFiles.id, fileId)).then(rows => rows[0]);
    if (!file) throw error(404, 'File not found');

    const stream = new TransformStream();
    const writer = stream.writable.getWriter();
    const transcribeStream = new TranscribeStream(writer);

    // Add stream to job queue
    const job = jobQueue.addJob(fileId);
    jobQueue.addStream(fileId, transcribeStream);

    // If job isn't running and file is still pending, start transcription
    if (!job.isRunning && file.transcriptionStatus === 'pending') {
        job.isRunning = true;
        transcribeAudio(fileId, transcribeStream).catch(console.error);
    } else {
        // Send current status immediately
        transcribeStream.sendProgress({
            status: file.transcriptionStatus,
            progress: file.transcriptionProgress || 0,
            transcript: file.transcript
        }).catch(console.error);
    }

    return new Response(stream.readable, {
        headers: {
            'Content-Type': 'text/event-stream',
            'Cache-Control': 'no-cache',
            'Connection': 'keep-alive'
        }
    });
};
