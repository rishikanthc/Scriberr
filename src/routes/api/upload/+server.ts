import { error } from '@sveltejs/kit';
import { requireAuth, checkSetupStatus } from '$lib/server/auth';
import type { RequestHandler } from './$types';
import { mkdir, writeFile, readFile, unlink } from 'fs/promises';
import { join } from 'path';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { queueTranscriptionJob } from '$lib/server/queue';
import { transcribeAudio } from '$lib/server/transcription';
import { TranscribeStream } from '$lib/server/transcribeStream';
import { jobQueue } from '$lib/server/jobQueue';
import { promisify } from 'util';
import { exec } from 'child_process';

// Use process.env directly instead of importing from $env modules
const AUDIO_DIR = process.env.AUDIO_DIR;
const WORK_DIR = process.env.WORK_DIR;
const WHISPER_BATCH_SIZE = process.env.WHISPER_BATCH_SIZE || '16';

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
    // Skip auth check during setup
    const isSetupComplete = await checkSetupStatus().catch(() => false);
    
    if (isSetupComplete) {
        try {
            await requireAuth(locals);
        } catch (error) {
            console.error("Auth failed for upload:", error);
            return new Response('Unauthorized', { status: 401 });
        }
    } else {
        console.log("Skipping auth check for upload as system may not be initialized");
    }
    
    try {
        await mkdir(UPLOAD_DIR, { recursive: true });
        const formData = await request.formData();
        const file = formData.get('file') as File;
        const optionsStr = formData.get('options') as string;
        const options = optionsStr ? JSON.parse(optionsStr) : {
            language: 'en',
            modelSize: 'base',
            diarization: false,
            threads: 4,
            processors: 1
        };

        // Convert diarization to a Boolean
        options.diarization = Boolean(options.diarization);

        console.log("Diarization setting:", options.diarization);

        if (!file) {
            throw error(400, 'No file uploaded');
        }

        // Create directories if needed
        await mkdir(TEMP_DIR, { recursive: true });
        await mkdir(UPLOAD_DIR, { recursive: true });
        
        // Generate a timestamp for consistent naming
        const timestamp = Date.now();
        const fileExt = file.name.split('.').pop()?.toLowerCase() || '';
        
        // Create temporary path for uploaded file
        const tempOriginalPath = join(TEMP_DIR, `${timestamp}-original-${file.name}`);
        await writeFile(tempOriginalPath, Buffer.from(await file.arrayBuffer()));
        console.log("SAVE ORIG --->")

        try {
            // Save a copy of the original file in its native format for high-quality playback
            const originalFileName = `${timestamp}-original.${fileExt}`;
            const originalFilePath = join(UPLOAD_DIR, originalFileName);
            await execAsync(`cp "${tempOriginalPath}" "${originalFilePath}"`);
            console.log("Saved original file for playback:", originalFilePath);
            
            // Convert to WAV for transcription (optimized for speech recognition)
            const convertedPath = await convertToWav(tempOriginalPath);
            
            // Generate WAV filename and move to uploads directory
            const finalFileName = `${timestamp}.wav`;
            const finalPath = join(UPLOAD_DIR, finalFileName);
            
            // Move converted file to uploads directory
            await execAsync(`mv "${convertedPath}" "${finalPath}"`);
            
            // Extract peaks from the converted WAV file for visualization
            const peaks = await extractPeaks(finalPath);
            
            // Create database entry with both original and WAV file info
            console.log("Transcription options being used:", options);
            console.log("Diarization setting:", options.diarization);
            const [audioFile] = await db.insert(audioFiles).values({
                fileName: finalFileName, // WAV file for transcription
                originalFileName: originalFileName, // Original file preserved
                originalFileType: fileExt, // Store the file type
                transcriptionStatus: 'pending',
                language: options.language,
                uploadedAt: new Date(),
                title: file.name.replace(/\.[^/.]+$/, ""), // Use original filename without extension as title
                peaks,
                modelSize: options.modelSize,
                diarization: options.diarization,
                threads: options.threads,
                processors: options.processors,
            }).returning();

            // Queue transcription job for worker
            await queueTranscriptionJob(audioFile.id, options);
            console.log('Queued job:', { audioFile });
            
            // Fire and forget - start transcription in the background
            setTimeout(async () => {
                try {
                    // Create stream for this transcription
                    const transcribeStream = new TranscribeStream();
                    
                    // Mark job as running to prevent duplicate processing
                    jobQueue.setJobRunning(audioFile.id, true);
                    jobQueue.addStream(audioFile.id, transcribeStream);
                    
                    // Process transcription directly
                    await transcribeAudio(audioFile.id, transcribeStream);
                    console.log(`Direct transcription completed for file ID: ${audioFile.id}`);
                } catch (err) {
                    console.error('Direct transcription error:', err);
                    jobQueue.setJobRunning(audioFile.id, false);
                }
            }, 100);

            // Clean up temp file
            await unlink(tempOriginalPath).catch(console.error);

            return new Response(JSON.stringify({ 
                id: audioFile.id,
                fileName: finalFileName,
                originalFileName: originalFileName,
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