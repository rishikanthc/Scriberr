// lib/server/worker.ts
import { getBoss, QUEUE_NAMES } from './queue';
import { transcribeAudio } from './transcription';
import type { TranscriptionJob } from '$lib/types';
import { TranscribeStream } from './transcribeStream';
import { jobQueue } from './jobQueue';

export async function startWorker() {
  console.log('Starting worker...');
  const boss = await getBoss();
  
  await boss.work<TranscriptionJob>(
    QUEUE_NAMES.TRANSCRIPTION,
    async (jobs) => {  // Note: pg-boss passes an array of jobs
      console.log('Received jobs:', jobs);
      
      // Process each job in the array
      for (const job of jobs) {
        if (!job?.data?.audioId) {
          console.error('Invalid job data:', job);
          continue;
        }

        try {
          console.log(`Processing transcription for audioId: ${job.data.audioId}`);

          const stream = new TransformStream();
          const writer = stream.writable.getWriter();
          const transcribeStream = new TranscribeStream(writer);

          // Add job to the job queue
          const queueJob = jobQueue.addJob(job.data.audioId);
          queueJob.isRunning = true;
          jobQueue.addStream(job.data.audioId, transcribeStream);

          // Process the transcription
          const result = await transcribeAudio(job.data.audioId, transcribeStream);
          console.log('Transcription completed:', result);
        } catch (error) {
          console.error(`Failed to process job ${job.id}:`, error);
          throw error;
        }
      }
    }
  );

  console.log('Worker started successfully');
}
