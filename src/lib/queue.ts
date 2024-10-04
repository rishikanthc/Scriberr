import { Queue, Worker } from 'bullmq';
import { exec } from 'child_process';
import fs from 'fs';
import path from 'path';
import PocketBase from 'pocketbase';
import { env } from '$env/dynamic/private';

import { ExpressAdapter } from '@bull-board/express';
import { createBullBoard } from '@bull-board/api';
import { BullMQAdapter } from '@bull-board/api/bullMQAdapter';
import express from 'express';

// Create the queue
export const transcriptionQueue = new Queue('transcriptionQueue', {
	connection: { host: 'localhost', port: 6379 } // Adjust as needed
});
const pb = new PocketBase('http://localhost:8080');
pb.autoCancellation(false);
await pb.admins.authWithPassword(env.POCKETBASE_ADMIN_EMAIL, env.POCKETBASE_ADMIN_PASSWORD);

// Create an express app
const app = express();

// Create Bull Board UI
const serverAdapter = new ExpressAdapter();
createBullBoard({
	queues: [new BullMQAdapter(transcriptionQueue)],
	serverAdapter: serverAdapter
});

// Set the base path for the Bull Board UI
serverAdapter.setBasePath('/admin/queues');

// Expose the Bull Board UI via Express
app.use('/admin/queues', serverAdapter.getRouter());

// Start the express server (if not already running)
const PORT = 9243;
app.listen(PORT, () => {
	console.log(`Bull Board running at http://localhost:${PORT}/admin/queues`);
});

// Helper function to execute shell commands and log output
const execCommandWithLogging = (cmd: string, job: Job) => {
	return new Promise((resolve, reject) => {
		const process = exec(cmd);

		// Capture stdout
		process.stdout.on('data', async (data) => {
			console.log(`stdout: ${data}`);
			await job.log(`stdout: ${data}`);
		});

		// Capture stderr and update progress
		process.stderr.on('data', async (data) => {
			console.error(`stderr: ${data}`);
			await job.log(`stderr: ${data}`);

			// Check if stderr contains a progress update from Whisper
			const progressMatch = data.toString().match(/progress\s*=\s*(\d+)%/);
			if (progressMatch) {
				const progress = parseInt(progressMatch[1], 10);
				if (progress == 100) {
					return;
				}
				await job.updateProgress(progress);
			}
		});

		// Handle process close event
		process.on('close', (code) => {
			if (code === 0) {
				resolve(true);
			} else {
				reject(new Error(`Command failed with exit code ${code}`));
			}
		});
	});
};

// Set up the worker to process jobs automatically
const worker = new Worker(
	'transcriptionQueue',
	async (job) => {
		const { recordId } = job.data;
		await job.log(`Starting job ${job.id} for record ${recordId}`);

		// Authenticate with PocketBase
		const record = await pb.collection('scribo').getOne(recordId);
		await job.log(`Fetched record for ${recordId}`);

		const audioFilename = record.audio;
		const fileExtension = path.extname(audioFilename);

		// Download the audio file
		const audioUrl = pb.files.getUrl(record, audioFilename);
		const audioPath = path.resolve(env.SCRIBO_FILES, 'audio', `${recordId}${fileExtension}`);
		const ffmpegPath = path.resolve(env.SCRIBO_FILES, 'audio', `${recordId}-ffmpeg.wav`);
		const res = await fetch(audioUrl);

		const buffer = await res.arrayBuffer();
		fs.writeFileSync(audioPath, Buffer.from(buffer));
		await job.log(`Downloaded and saved audio file for record ${recordId}`);

		const audiowaveformCmd = `audiowaveform -i ${audioPath} -o ${audioPath}.json`;
		await execCommandWithLogging(audiowaveformCmd, job);
		await job.log(`Audiowaveform for ${recordId} generated`);

		// Execute the ffmpeg command and log output
		const ffmpegCmd = `ffmpeg -i ${audioPath} -ar 16000 -ac 1 -c:a pcm_s16le ${ffmpegPath}`;
		await execCommandWithLogging(ffmpegCmd, job);
		await job.log(`Audio file for ${recordId} converted successfully`);

		const settingsRecords = await pb.collection('settings').getList(1, 1);
		const settings = settingsRecords.items[0];

		// Execute whisper.cpp command and log output
		const transcriptPath = path.resolve(env.SCRIBO_FILES, 'transcripts', `${recordId}`);
		const whisperCmd = `./whisper.cpp/main -m ./whisper.cpp/models/ggml-${settings.model}.en.bin -f ${ffmpegPath} -oj -of ${transcriptPath} -t ${settings.threads} -p ${settings.processors} -pp`;
		await execCommandWithLogging(whisperCmd, job);
		await job.log(`Whisper transcription for ${recordId} completed`);

		// Audiowaveform generation

		// Read and update transcript
		const transcript = fs.readFileSync(`${transcriptPath}.json`, 'utf-8');
		const audioPeaks = fs.readFileSync(`${audioPath}.json`, 'utf-8');

		await pb.collection('scribo').update(recordId, {
			transcript,
			processed: true,
			peaks: JSON.parse(audioPeaks)
		});
		await job.log(`Updated PocketBase record for ${recordId}`);

		// Clean up
		fs.unlinkSync(audioPath);
		fs.unlinkSync(`${audioPath}.json`);
		fs.unlinkSync(ffmpegPath);
		fs.unlinkSync(`${transcriptPath}.json`);
		await job.log(`Cleaned up temporary files for ${recordId}`);

		console.log(`Job ${job.id} for record ${recordId} completed successfully`);
		job.updateProgress(100); // Mark job progress as complete
	},
	{
		connection: { host: 'localhost', port: 6379 }, // Redis connection
		concurrency: 5 // Allows multiple jobs to run concurrently
	}
);
