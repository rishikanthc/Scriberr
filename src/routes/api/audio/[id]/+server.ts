import { error, json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { createReadStream, statSync } from 'fs';
import { join } from 'path';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { requireAuth } from '$lib/server/auth';
import { AUDIO_DIR, WORK_DIR } from '$env/static/private'; 

export const GET: RequestHandler = async ({ params, locals, request }) => {
    console.log("AUDIO REQ --->")
    await requireAuth(locals);
    console.log("AUDIO REQ ---> AUTH DONE")
    try {
        const id = parseInt(params.id);
        if (isNaN(id)) {
            throw error(400, 'Invalid file ID');
        }

        const [file] = await db
            .select()
            .from(audioFiles)
            .where(eq(audioFiles.id, id));

        if (!file) {
            throw error(404, 'File not found');
        }

        const filePath = join(AUDIO_DIR, file.fileName);
        const stats = statSync(filePath);
        
        // Handle range requests for better streaming
        const range = request.headers.get('range');
        if (range) {
            const parts = range.replace(/bytes=/, '').split('-');
            const start = parseInt(parts[0], 10);
            const end = parts[1] ? parseInt(parts[1], 10) : stats.size - 1;
            const chunkSize = (end - start) + 1;
            const stream = createReadStream(filePath, { start, end });
            
            const headers = new Headers({
                'Content-Type': 'audio/mpeg', // or the correct mime type for your audio
                'Content-Length': chunkSize.toString(),
                'Content-Range': `bytes ${start}-${end}/${stats.size}`,
                'Accept-Ranges': 'bytes',
                'Cache-Control': 'public, max-age=3600',
            });
            
            return new Response(stream, { 
                status: 206,
                headers 
            });
        }

        // Non-range request - send entire file
        const stream = createReadStream(filePath);
        const headers = new Headers({
            'Content-Type': 'audio/mpeg', // or the correct mime type for your audio
            'Content-Length': stats.size.toString(),
            'Accept-Ranges': 'bytes',
            'Cache-Control': 'public, max-age=3600',
        });

        return new Response(stream, { headers });
    } catch (err) {
        console.error('Audio file serve error:', err);
        if (err.status) throw err;
        throw error(500, 'Internal server error');
    }
};

export const PATCH: RequestHandler = async ({ params, request, locals }) => {
    await requireAuth(locals);
    try {
        const id = parseInt(params.id);
        if (isNaN(id)) throw error(400, 'Invalid file ID');
        
        const body = await request.json();
        
        // Remove any undefined or null values from the update
        const updateData = Object.fromEntries(
            Object.entries(body).filter(([_, value]) => value !== undefined && value !== null)
        );
        
        // Validate the fields being updated
        const allowedFields = new Set([
            'title',
            'duration',
            'peaks',
            'transcriptionStatus',
            'language',
            'transcribedAt',
            'transcript',
            'diarization',
            'lastError'
        ]);

        const invalidFields = Object.keys(updateData).filter(field => !allowedFields.has(field));
        if (invalidFields.length > 0) {
            throw error(400, `Invalid fields: ${invalidFields.join(', ')}`);
        }

        const [updated] = await db
            .update(audioFiles)
            .set(updateData)
            .where(eq(audioFiles.id, id))
            .returning();
            
        return json(updated);
    } catch (err) {
        console.error('Update error:', err);
        throw error(500, String(err));
    }
};

export async function DELETE({ params }) {
  try {
    const fileId = parseInt(params.id);
    if (isNaN(fileId)) {
      return new Response('Invalid file ID', { status: 400 });
    }

    // Delete the record from the database
    await db.delete(audioFiles)
      .where(eq(audioFiles.id, fileId));

    // You might also want to delete the actual file from storage here
    // This depends on your storage implementation
    // For example:
    // await deleteFileFromStorage(fileId);

    return json({ success: true });
  } catch (error) {
    console.error('Error deleting file:', error);
    return new Response('Failed to delete file', { 
      status: 500 
    });
  }
}
