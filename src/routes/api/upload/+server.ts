import { error } from '@sveltejs/kit';
import { requireAuth } from '$lib/server/auth';
import type { RequestHandler } from './$types';
import { mkdir, writeFile, readFile, unlink } from 'fs/promises';
import { join } from 'path';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { queueTranscriptionJob } from '$lib/server/queue';
import { promisify } from 'util';
import { exec } from 'child_process';
import { AUDIO_DIR, WORK_DIR } from '$env/static/private'; 

const execAsync = promisify(exec);

let UPLOAD_DIR;
let TEMP_DIR;

if (AUDIO_DIR !== '') {
    UPLOAD_DIR = AUDIO_DIR;
} else {
    UPLOAD_DIR = join(process.cwd(), 'uploads')
}

if (WORK_DIR !== '') {
    TEMP_DIR = WORK_DIR;

} else {
    TEMP_DIR = join(process.cwd(), 'temp');
}

async function convertToWav(inputPath: string): Promise<string> {
    await mkdir(TEMP_DIR, { recursive: true });
    const outputPath = join(TEMP_DIR, `${Date.now()}-converted.wav`);
    
    try {
        await execAsync(
            `ffmpeg -i "${inputPath}" -ar 16000 -ac 1 -c:a pcm_s16le "${outputPath}"`
        );
        return outputPath;
    } catch (err) {
        console.error('Failed to convert audio:', err);
        throw new Error('Audio conversion failed');
    }
}

async function extractPeaks(audioPath: string): Promise<number[]> {
    try {
        await mkdir(TEMP_DIR, { recursive: true });
        const jsonPath = join(TEMP_DIR, `${Date.now()}.json`);
        
        await execAsync(`audiowaveform -i "${audioPath}" -o "${jsonPath}"`);
        
        const waveformData = JSON.parse(await readFile(jsonPath, 'utf-8'));
        
        await unlink(jsonPath);
        
        return waveformData.data || [];
    } catch (err) {
        console.error('Failed to extract peaks:', err);
        return [];
    }
}

export const POST: RequestHandler = async ({ request, locals}) => {
    await requireAuth(locals);
    
    try {
        await mkdir(UPLOAD_DIR, { recursive: true });
        const formData = await request.formData();
        const file = formData.get('file') as File;
        const options = JSON.parse(formData.get('options') as string);
        
        if (!file) {
            throw error(400, 'No file uploaded');
        }

        // Save original file to temp directory first
        await mkdir(TEMP_DIR, { recursive: true });
        const tempOriginalPath = join(TEMP_DIR, `${Date.now()}-original-${file.name}`);
        await writeFile(tempOriginalPath, Buffer.from(await file.arrayBuffer()));
        console.log("SAVE ORIG --->")

        try {
            // Convert to WAV
            const convertedPath = await convertToWav(tempOriginalPath);
            
            // Generate final filename and move to uploads directory
            const finalFileName = `${Date.now()}.wav`;
            const finalPath = join(UPLOAD_DIR, finalFileName);
            
            // Move converted file to uploads directory
            await execAsync(`mv "${convertedPath}" "${finalPath}"`);
            
            // Extract peaks from the converted WAV file
            const peaks = await extractPeaks(finalPath);
            
            // Create database entry
            const [audioFile] = await db.insert(audioFiles).values({
                fileName: finalFileName,
                transcriptionStatus: 'pending',
                language: options.language,
                uploadedAt: new Date(),
                title: finalFileName,
                peaks,
                modelSize: options.modelSize,
                diarization: options.diarization,
               threads: options.threads,
               processors: options.processors,
            }).returning();

            // Queue transcription job
            await queueTranscriptionJob(audioFile.id, options);
            console.log('Queued job:', { audioFile });

            // Clean up temp files
            await unlink(tempOriginalPath).catch(console.error);

            return new Response(JSON.stringify({ 
                id: audioFile.id,
                fileName: finalFileName,
                peaks,
            }), {
                headers: {
                    'Content-Type': 'application/json'
                }
            });
        } finally {
            // Clean up temp original file if it exists
            await unlink(tempOriginalPath).catch(() => {});
        }
    } catch (err) {
        console.error('Upload error:', err);
        throw error(500, 'Failed to upload file');
    }
};
