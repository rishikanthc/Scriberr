// lib/server/worker.ts
import { initQueue, QUEUE_NAMES } from './queue';
import { transcribeAudio } from './transcription';
import type { TranscriptionJob } from '$lib/types';
import { TranscribeStream } from './transcribeStream';
import { jobQueue } from './jobQueue';

export async function startWorker() {
  console.log('Starting worker...');
  const boss = await initQueue();
  
  if (!boss) {
    console.log('Queue not initialized, worker cannot start');
    return;
  }
  
  // Subscribe to the 'error' event on the boss instance
  boss.on('error', error => {
    console.error('pg-boss worker error:', error);
  });
  
  // Use the correct handler signature for pg-boss
  await boss.work<TranscriptionJob>(
    QUEUE_NAMES.TRANSCRIPTION,
    async (job) => {  // pg-boss passes a single job, not an array
      console.log('Received job:', job);
      
      if (!job?.data?.audioId) {
        console.error('Invalid job data:', job);
        return { success: false, error: 'Invalid job data' };
      }

      try {
        const audioId = job.data.audioId;
        console.log(`Processing transcription for audioId: ${audioId}`);

        // Create a transcribe stream for this job
        const transcribeStream = new TranscribeStream();
        
        // Add job to the job queue and mark as running
        const queueJob = jobQueue.addJob(audioId);
        queueJob.isRunning = true;
        jobQueue.addStream(audioId, transcribeStream);
        
        // Send initial status
        await transcribeStream.sendProgress({
          status: 'processing',
          progress: 0,
          transcript: []
        });

        // Process the transcription
        const result = await transcribeAudio(audioId, transcribeStream);
        console.log('Transcription completed:', result ? result.length + ' segments' : 'no result');
        
        return { success: true, audioId };
      } catch (error) {
        console.error(`Failed to process job ${job.id}:`, error);
        return { success: false, error: error.message || 'Unknown error' };
      } finally {
        // Ensure job is marked as not running after completion
        if (job?.data?.audioId) {
          jobQueue.setJobRunning(job.data.audioId, false);
        }
      }
    }
  );

  console.log('Worker started successfully and listening for transcription jobs');
  
  // Fetch and process any pending jobs that might have been missed
  try {
    // Use object parameter form with limit property instead of direct numeric parameter
    const pendingJobs = await boss.fetch(QUEUE_NAMES.TRANSCRIPTION, { limit: 10 });
    console.log(`Found ${pendingJobs?.length || 0} pending jobs to process`);
  } catch (error) {
    console.error('Error fetching pending jobs:', error);
  }
}