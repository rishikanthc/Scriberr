// lib/server/queue.ts
import PgBoss from 'pg-boss';
import { DATABASE_URL } from '$env/static/private';
import type { TranscriptionJob } from '$lib/types';
import { TranscribeStream } from './transcribeStream';
import { transcribeAudio } from './transcription';
import { jobQueue } from './jobQueue';

export const QUEUE_NAMES = {
  TRANSCRIPTION: 'transcription'
} as const;

let bossInstance: PgBoss | null = null;

export async function getBoss() {
  if (!bossInstance) {
    console.log('Initializing PgBoss...');
    bossInstance = new PgBoss({
      connectionString: DATABASE_URL,
      schema: 'pgboss',
    });

    bossInstance.on('error', error => {
      console.error('PgBoss error:', error);
    });

    await bossInstance.start();
    console.log('PgBoss started successfully');

    // await bossInstance.createQueue(QUEUE_NAMES.TRANSCRIPTION)
    // console.log("QUEUE CREATED")
  }
  return bossInstance;
}

export async function setupTranscriptionWorker() {
    const boss = await getBoss();
    
    await boss.work<TranscriptionJob>(
        QUEUE_NAMES.TRANSCRIPTION,
        async (jobs) => {
      for (const job of jobs) {
        if (!job?.data?.audioId) {
            console.error('Invalid job data:', job);
            return;
        }
        try {
            console.log(`Processing transcription for audioId: ${job.data.audioId}`);
            const stream = new TransformStream();
            const writer = stream.writable.getWriter();
            const transcribeStream = new TranscribeStream(writer);
            
            const queueJob = jobQueue.addJob(job.data.audioId);
            queueJob.isRunning = true;
            jobQueue.addStream(job.data.audioId, transcribeStream);
            
            const result = await transcribeAudio(job.data.audioId, transcribeStream);
            console.log('Transcription completed:', result);
        } catch (error) {
            console.error(`Failed to process job ${job.id}:`, error);
            throw error;
        }
      }
    }
    );
    return boss
}


export async function queueTranscriptionJob(audioId: number, options): Promise<string> {
  const boss = await getBoss();
    
  try {
    console.log('Queuing transcription job for audioId:', audioId);
    const jobId = await boss?.send(QUEUE_NAMES.TRANSCRIPTION, { audioId, options });
    console.log('Job queued successfully:', { jobId, audioId });
  } catch (error) {
    console.error('Failed to queue job:', error);
    throw error;
  }
}
