// lib/server/queue.ts
import PgBoss from 'pg-boss';
import type { TranscriptionJob } from '$lib/types';
import { TranscribeStream } from './transcribeStream';
import { transcribeAudio } from './transcription';
import { jobQueue } from './jobQueue';

// Use process.env directly instead of importing from $env modules
const DATABASE_URL = process.env.DATABASE_URL;
const USE_WORKER = process.env.USE_WORKER === 'true';

export const QUEUE_NAMES = {
  TRANSCRIPTION: 'transcription'
};

let boss: PgBoss | null = null;

export async function initQueue() {
  try {
    // Skip queue initialization if DATABASE_URL is not set (development/build mode)
    if (!DATABASE_URL) {
      console.log('DATABASE_URL not set, skipping queue initialization');
      return null;
    }

    if (boss) {
      console.log('Queue already initialized, reusing existing instance');
      return boss;
    }

    console.log('Initializing queue system with URL:', DATABASE_URL ? '[configured]' : '[missing]');
    boss = new PgBoss(DATABASE_URL);
    
    boss.on('error', error => console.error('pg-boss error:', error));
    
    await boss.start();
    console.log('Queue system initialized successfully');
    
    return boss;
  } catch (error) {
    console.error('Failed to initialize queue:', error);
    return null;
  }
}

export async function queueTranscriptionJob(audioId: number, options = {}) {
  try {
    // Log the environment settings
    console.log('Queue settings:', { DATABASE_URL: !!DATABASE_URL, USE_WORKER });
    
    if (!boss) {
      // In development without a queue, process directly
      if (!DATABASE_URL || !USE_WORKER) {
        console.log('No queue available or worker disabled, processing transcription directly');
        jobQueue.setJobRunning(audioId, true);
        const stream = new TranscribeStream();
        jobQueue.broadcastToJob(audioId, { status: 'starting' });
        
        try {
          await transcribeAudio(audioId, stream);
          return { success: true, audioId };
        } catch (error) {
          console.error('Direct processing error:', error);
          jobQueue.broadcastToJob(audioId, { status: 'failed', error: error instanceof Error ? error.message : String(error) });
          return { success: false, error: error instanceof Error ? error.message : String(error) };
        } finally {
          jobQueue.setJobRunning(audioId, false);
        }
      }
      
      // Try to initialize the queue if it's not initialized yet
      console.log('Initializing queue for job...');
      await initQueue();
      
      if (!boss) {
        console.error('Queue system could not be initialized');
        throw new Error('Queue system not initialized');
      }
    }
    
    // Queue the job
    console.log('Queueing job for audioId:', audioId);
    const job = await boss.send(QUEUE_NAMES.TRANSCRIPTION, { audioId, options });
    console.log('Job queued successfully:', job);
    
    return { success: true, jobId: job };
  } catch (error) {
    console.error('Failed to queue job:', error);
    throw error;
  }
}

// Initialize the queue when this module is imported
initQueue().catch(console.error);