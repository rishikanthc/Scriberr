import { error, json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { createReadStream, statSync } from 'fs';
import { join } from 'path';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { requireAuth } from '$lib/server/auth';

// Use process.env directly instead of importing from $env modules
const AUDIO_DIR = process.env.AUDIO_DIR;
const WORK_DIR = process.env.WORK_DIR;

export const GET: RequestHandler = async ({ params, locals, request, url }) => {
    console.log("AUDIO REQ --->")
    await requireAuth(locals);
    console.log("AUDIO REQ ---> AUTH DONE")
    try {
        const id = parseInt(params.id);
        if (isNaN(id)) throw error(400, 'Invalid file ID');

        // Get the file info from the database
        const file = await db
            .select()
            .from(audioFiles)
            .where(eq(audioFiles.id, id))
            .then(rows => rows[0]);

        if (!file) throw error(404, 'File not found');

        // Determine which file to serve (original or transcoded)
        const useOriginal = url.searchParams.has('original');
        const fileName = useOriginal ? file.originalFileName : file.fileName;

        // Determine audio directory
        let audioDirectory = AUDIO_DIR || join(process.cwd(), 'uploads');

        // Build the full path to the audio file
        const filePath = join(audioDirectory, fileName);

        // Get file stats
        const stats = statSync(filePath);

        // Determine content type based on file extension
        const fileExt = fileName.split('.').pop()?.toLowerCase();
        let contentType = 'audio/mpeg'; // Default
        
        if (fileExt === 'wav') contentType = 'audio/wav';
        else if (fileExt === 'ogg') contentType = 'audio/ogg';
        else if (fileExt === 'flac') contentType = 'audio/flac';
        else if (fileExt === 'm4a') contentType = 'audio/mp4';
        else if (fileExt === 'mp3') contentType = 'audio/mpeg';
        
        // Support range requests for seeking
        const range = request.headers.get('range');
        
        if (range) {
            const parts = range.replace(/bytes=/, '').split('-');
            const start = parseInt(parts[0], 10);
            let end = parts[1] ? parseInt(parts[1], 10) : stats.size - 1;
            
            // Prevent oversized chunks
            const CHUNK_SIZE = 1024 * 1024; // 1MB
            if (end - start >= CHUNK_SIZE) {
                end = start + CHUNK_SIZE - 1;
            }
            
            const contentLength = end - start + 1;
            console.log("Serving audio range:", { start, end, contentLength });
            
            // Create read stream for the specific range
            const stream = createReadStream(filePath, { start, end });
            
            const headers = {
                'Content-Type': contentType,
                'Content-Length': contentLength.toString(),
                'Content-Range': `bytes ${start}-${end}/${stats.size}`,
                'Accept-Ranges': 'bytes',
                'Cache-Control': 'public, max-age=3600',
            };
            
            return new Response(stream, { 
                status: 206, 
                headers
            });
        }
        
        // Serve the entire file
        console.log("Serving full audio file:", filePath);
        const stream = createReadStream(filePath);
        
        const headers = {
            'Content-Type': contentType,
            'Content-Length': stats.size.toString(),
            'Accept-Ranges': 'bytes',
            'Cache-Control': 'public, max-age=3600',
        };

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