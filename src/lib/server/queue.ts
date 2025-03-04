// lib/server/queue.ts
import PgBoss from 'pg-boss';
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
    // Get DATABASE_URL from environment variables with fallback for build time
    let DATABASE_URL = process.env.DATABASE_URL || 'postgres://placeholder:placeholder@db:5432/placeholder';
    
    // If we have the component parts but not the URL, construct it (for Docker)
    if (!process.env.DATABASE_URL && process.env.POSTGRES_USER && process.env.POSTGRES_PASSWORD && process.env.POSTGRES_DB) {
      DATABASE_URL = `postgres://${process.env.POSTGRES_USER}:${process.env.POSTGRES_PASSWORD}@db:5432/${process.env.POSTGRES_DB}`;
      console.log("Generated DATABASE_URL from components for queue");
    }
    
    // Only show warning during runtime, not during build
    if (!process.env.DATABASE_URL && process.env.RUNTIME_CHECK === 'true') {
      console.warn('DATABASE_URL not provided, using default or generated value');
    }
    
    console.log('Initializing PgBoss...');
    console.log(`Using database connection: ${DATABASE_URL.replace(/:[^:@]+@/, ':***@')}`);
    
    try {
      bossInstance = new PgBoss({
        connectionString: DATABASE_URL,
        schema: 'pgboss',
        noScheduling: true, // Disable scheduling to simplify operation
        monitorStateIntervalSeconds: 10, // Reduce polling
      });

      bossInstance.on('error', error => {
        console.error('PgBoss error:', error);
      });

      await bossInstance.start();
      console.log('PgBoss started successfully');
    } catch (error) {
      console.error('Failed to initialize PgBoss:', error);
      throw error;
    }
  }
  return bossInstance;
}

export async function setupTranscriptionWorker() {
  try {
    // Skip worker setup during build
    if (process.env.NODE_ENV === 'build') {
      console.log("Skipping transcription worker setup during build");
      return null;
    }
    
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
    return boss;
  } catch (error) {
    console.error("Failed to setup transcription worker:", error);
    return null;
  }
}

export async function queueTranscriptionJob(audioId: number, options): Promise<string> {
  const boss = await getBoss();
    
  try {
    console.log('Queuing transcription job for audioId:', audioId);
    const jobId = await boss?.send(QUEUE_NAMES.TRANSCRIPTION, { audioId, options });
    console.log('Job queued successfully:', { jobId, audioId });
    return jobId;
  } catch (error) {
    console.error('Failed to queue job:', error);
    throw error;
  }
}