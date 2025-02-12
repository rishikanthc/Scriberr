// lib/server/transcription.ts

import { spawn } from 'child_process';
import { readFile } from 'fs/promises';
import { join } from 'path';
import { db } from './db';
import { audioFiles } from './db/schema';
import { jobQueue } from './jobQueue';
import type { TranscriptSegment } from '$lib/types';
import { eq } from 'drizzle-orm';
import { TranscribeStream } from './transcribeStream';
import { AUDIO_DIR } from '$env/static/private';

let UPLOAD_DIR = AUDIO_DIR
  ? AUDIO_DIR
  : join(process.cwd(), 'uploads');

export async function transcribeAudio(audioId: number, stream: TranscribeStream) {
  // 1. Grab the file record
  const file = await db
    .select()
    .from(audioFiles)
    .where(eq(audioFiles.id, audioId))
    .then(rows => rows[0]);

  if (!file) throw new Error('File not found');

  // 2. Prepare paths
  const inputPath = join(UPLOAD_DIR, file.fileName);
  // We'll write output to e.g. "someFileName.json"
  const outputPath = join(UPLOAD_DIR, `${file.fileName}.json`);

  // 3. Build arguments for the Python script
  const pythonArgs: string[] = [
    'transcribe.py',
    '--audio-file', inputPath,
    '--model-size', 'base',
    '--device', 'cpu',
    '--compute-type', 'int8',
    '--output-file', outputPath,
    '--HF_TOKEN', process.env.HF_API_KEY,
    '--diarization-model', process.env.DIARIZATION_MODEL,
  ];

  // If user selected an explicit language
  if (file.language) {
    pythonArgs.push('--language', file.language);
  }

  // If alignment was requested
  if (file.align) {
    pythonArgs.push('--align');
  }

  // If diarization was requested
  if (file.diarization) {
    pythonArgs.push('--diarize');

    // Check if HF_API_KEY is defined
    if (process.env.HF_API_KEY) {
      pythonArgs.push('--HF_TOKEN', process.env.HF_API_KEY);
    } else {
      console.warn('HF_API_KEY is not set. Diarization may not be performed.');
    }

    // Check if DIARIZATION_MODEL is defined
    if (process.env.DIARIZATION_MODEL) {
      pythonArgs.push('--diarization-model', process.env.DIARIZATION_MODEL);
    }
  }

  console.log("Launching transcribe.py with arguments:", pythonArgs);

  // 4. Spawn the Python process
  const pythonProcess = spawn('python3', pythonArgs, {
    cwd: process.cwd(),  // ensure we're in project root
    shell: true,
    env: { ...process.env, PYTHONUNBUFFERED: '1' }
  });

  let progressStream = stream;
  let lastProgress = 0;

  // 5. Parse progress from the Python script's stdout
  pythonProcess.stdout.on('data', async (data) => {
    const output = data.toString();

    // If your transcribe.py prints progress lines like: "Progress: 33.33%"
    // we can detect and parse them:
    const match = output.match(/Progress:\s*([\d.]+)%/i);
    if (match) {
      const currentProgress = parseFloat(match[1]);
      if (!isNaN(currentProgress) && currentProgress !== lastProgress) {
        lastProgress = currentProgress;
        await db.update(audioFiles)
          .set({
            transcriptionProgress: Math.floor(currentProgress),
            transcriptionStatus: 'processing',
            updatedAt: new Date()
          })
          .where(eq(audioFiles.id, audioId));
        jobQueue.broadcastToJob(audioId, {
          status: 'processing',
          progress: currentProgress
        });
      }
    }

    // Log other outputs for debugging purposes
    console.log(output);
  });

  pythonProcess.stderr.on('data', (data) => {
    // The script might print errors or logs to stderr
    console.error(`transcribe.py stderr:`, data.toString());
  });

  // 6. Wait for the Python script to finish
  try {
    await new Promise((resolve, reject) => {
      pythonProcess.on('close', async (code) => {
        if (code === 0) {
          try {
            // 7. The script wrote "outputPath" as JSON
            const rawJson = await readFile(outputPath, 'utf-8');
            const jsonData = JSON.parse(rawJson);
            let transcriptSegments: TranscriptSegment[] = [];

            // Extract the segments consistently, regardless of diarization
            const segments = jsonData.segments || [];
            transcriptSegments = segments.map((seg: any) => ({
              start: seg.start,
              end: seg.end,
              text: seg.text,
              speaker: seg.speaker ?? 'unknown'  // Use 'unknown' if speaker is not set
            }));

            // 8. Save transcript to DB
            await db.update(audioFiles)
              .set({
                transcript: JSON.stringify(transcriptSegments),
                transcriptionStatus: 'completed',
                transcriptionProgress: 100,
                transcribedAt: new Date(),
                updatedAt: new Date()
              })
              .where(eq(audioFiles.id, audioId));

            // 9. Broadcast final completion
            if (progressStream) {
              jobQueue.broadcastToJob(audioId, {
                status: 'completed',
                progress: 100,
                transcript: transcriptSegments
              });
            }

            resolve(null);
          } catch (err) {
            reject(err);
          }
        } else {
          reject(new Error(`transcribe.py exited with code ${code}`));
        }
      });

      pythonProcess.on('error', reject);
    });
  } catch (error: any) {
    console.error('Transcription error:', error);

    // Update DB with failure status
    await db.update(audioFiles)
      .set({
        transcriptionStatus: 'failed',
        lastError: error.message,
        updatedAt: new Date()
      })
      .where(eq(audioFiles.id, audioId));

    // Broadcast the failure
    jobQueue.broadcastToJob(audioId, { status: 'failed', error: error.message });

    throw error;
  } finally {
      // 10. Cleanup
      pythonProcess.kill();
      if (progressStream) {
        jobQueue.setJobRunning(audioId, false);
        await progressStream.close();
        progressStream = null;
      }
  }
}