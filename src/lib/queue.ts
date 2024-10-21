import { Queue, Worker } from 'bullmq';
import { exec } from 'child_process';
import { wizardQueue } from './wizardQueue';
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
	connection: { host: env.REDIS_HOST, port: env.REDIS_PORT }
});
const pb = new PocketBase(env.POCKETBASE_URL);
pb.autoCancellation(false);
await pb.admins.authWithPassword(env.POCKETBASE_ADMIN_EMAIL, env.POCKETBASE_ADMIN_PASSWORD);
const concur = Number(env.CONCURRENCY);

// Create an express app
const app = express();

// Create Bull Board UI
const serverAdapter = new ExpressAdapter();
createBullBoard({
	queues: [new BullMQAdapter(transcriptionQueue), new BullMQAdapter(wizardQueue)],
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

// Remove all jobs from the queue
async function clearQueue() {
	await transcriptionQueue.drain();
	console.log('Queue cleared!');
}

async function cleanAllJobs() {
	// Remove all completed jobs
	await transcriptionQueue.clean(0, 0, 'completed');
	console.log('All completed jobs have been removed');

	// Remove all failed jobs
	await transcriptionQueue.clean(0, 0, 'failed');
	console.log('All failed jobs have been removed');

	// Remove all waiting jobs
	await transcriptionQueue.clean(0, 0, 'waiting');
	console.log('All waiting jobs have been removed');

	// Remove all active jobs (this may kill jobs that are actively being processed)
	await transcriptionQueue.clean(0, 0, 'active');
	console.log('All active jobs have been removed');

	// Remove all delayed jobs
	await transcriptionQueue.clean(0, 0, 'delayed');
	console.log('All delayed jobs have been removed');
}

async function pauseAndCleanQueue() {
	// Pause the queue to prevent new jobs from being processed
	await transcriptionQueue.pause();
	console.log('Queue paused');

	// Clean all jobs as described above
	await cleanAllJobs();

	// Optionally resume the queue
	await transcriptionQueue.resume();
	console.log('Queue resumed');
}

pauseAndCleanQueue();

clearQueue();

// Helper function to execute shell commands and log output
const execCommandWithLogging = (cmd: string, job: Job, progress: number) => {
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
				const tprogress = parseInt(progressMatch[1], 10);

				// if (tprogress == 100) {
				// 	return;
				// }

				const _remaining = 95 - progress;
				const _prog = (_remaining * tprogress) / 100;

				await job.updateProgress(_prog);
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

		process.on('error', (err) => {
			reject(new Error(`Failed to start process: ${err.message}`));
		});
	});
};

// Set up the worker to process jobs automatically
const worker = new Worker(
	'transcriptionQueue',
	async (job) => {
		try {
			const { recordId, title } = job.data;
			await job.log(`Starting job ${job.id} for record ${recordId}`);

			// Authenticate with PocketBase
			const record = await pb.collection('scribo').getOne(recordId);
			await job.log(`Fetched record for ${recordId}`);

			const audioFilename = record.audio;
			const fileExtension = path.extname(audioFilename);

			// Download the audio file
			const baseUrl = path.resolve(env.SCRIBO_FILES, 'audio', `${recordId}`);
			fs.mkdir(baseUrl, { recursive: true }, (err) => {
				if (err) throw err;
			});
			const audioUrl = pb.files.getUrl(record, audioFilename);
			const audioPath = path.resolve(baseUrl, `${recordId}${fileExtension}`);
			const ffmpegPath = path.resolve(baseUrl, `${recordId}-ffmpeg.wav`);
			const res = await fetch(audioUrl);

			const buffer = await res.arrayBuffer();
			fs.writeFileSync(audioPath, Buffer.from(buffer));
			await job.log(`Downloaded and saved audio file for record ${recordId}`);

			job.updateProgress(1.5);
			// Execute the ffmpeg command and log output
			const ffmpegCmd = `ffmpeg -i ${audioPath} -ar 16000 -ac 1 -c:a pcm_s16le ${ffmpegPath}`;
			await execCommandWithLogging(ffmpegCmd, job);
			await job.log(`Audio file for ${recordId} converted successfully`);

			job.updateProgress(7.5);
			const audiowaveformCmd = `audiowaveform -i ${ffmpegPath} -o ${audioPath}.json`;
			await execCommandWithLogging(audiowaveformCmd, job);
			await job.log(`Audiowaveform for ${recordId} generated`);

			const settingsRecords = await pb.collection('settings').getList(1, 1);
			const settings = settingsRecords.items[0];

			// Execute whisper.cpp command and log output
			const transcriptdir = path.resolve(env.SCRIBO_FILES, 'transcripts', `${recordId}`);
			const transcriptPath = path.resolve(
				env.SCRIBO_FILES,
				'transcripts',
				`${recordId}`,
				`${recordId}`
			);
			fs.mkdir(transcriptdir, { recursive: true }, (err) => {
				if (err) throw err;
			});

			let whisperCmd;
			console.log(env.DEV_MODE);
			job.log(env.DEV_MODE);

			const isDevMode = env.DEV_MODE === 'true' || env.DEV_MODE === true;

			console.log('DEV MODE ----->', isDevMode);
			job.log(`DEV MODE -----> ${isDevMode}`);

			if (isDevMode) {
				whisperCmd = `./whisper.cpp/main -m ./whisper.cpp/models/ggml-${settings.model}.en.bin -f ${ffmpegPath} -oj -of ${transcriptPath} -t ${settings.threads} -p ${settings.processors} -pp`;
			} else {
				whisperCmd = `whisper -m /models/ggml-${settings.model}.en.bin -f ${ffmpegPath} -oj -of ${transcriptPath} -t ${settings.threads} -p ${settings.processors} -pp`;
			}

			let rttmContent;
			let segments;

			if (settings.diarize) {
				job.updateProgress(12);
				const rttmPath = path.resolve(baseUrl, `${recordId}.rttm`);
				const diarizeCmd = `python3 ./diarize/local.py ${ffmpegPath} ${rttmPath}`;
				await execCommandWithLogging(diarizeCmd, job);
				await job.log(`Diarization completed successfully`);
				// Read and parse the RTTM file
				rttmContent = fs.readFileSync(rttmPath, 'utf-8');
				segments = parseRttm(rttmContent);
				await job.log(`Parsed RTTM file for record ${recordId}`);

				if (isDevMode) {
					whisperCmd = `./whisper.cpp/main -m ./whisper.cpp/models/ggml-${settings.model}.en.bin -f ${ffmpegPath} -oj -of ${transcriptPath} -t ${settings.threads} -p ${settings.processors} -pp -ml 1`;
				} else {
					whisperCmd = `whisper -m /models/ggml-${settings.model}.en.bin -f ${ffmpegPath} -oj -of ${transcriptPath} -t ${settings.threads} -p ${settings.processors} -pp -ml 1`;
				}
			}

			job.updateProgress(35);

			await execCommandWithLogging(whisperCmd, job, 35);
			await job.log(`Whisper transcription for ${recordId} completed`);

			// Read and update transcript
			const transcript = fs.readFileSync(`${transcriptPath}.json`, 'utf-8');
			let transcriptJson = JSON.parse(transcript);
			console.log(transcriptJson);

			const audioPeaks = fs.readFileSync(`${audioPath}.json`, 'utf-8');
			let upd;

			if (settings.diarize) {
				const diarizedTranscript = generateTranscript(transcriptJson.transcription, rttmContent);
				const diarizedJson = { transcription: diarizedTranscript };

				upd = await pb.collection('scribo').update(recordId, {
					// transcript: '{ "test": "hi" }',
					transcript: transcriptJson,
					diarizedtranscript: diarizedJson,
					rttm: rttmContent,
					processed: true,
					diarized: true,
					peaks: JSON.parse(audioPeaks)
				});
			} else {
				upd = await pb.collection('scribo').update(recordId, {
					// transcript: '{ "test": "hi" }',
					transcript: transcriptJson,
					processed: true,
					diarized: false,
					peaks: JSON.parse(audioPeaks)
				});
			}

			await job.log(`Updated PocketBase record for ${recordId}`);
			console.log('UPDATED +++++ ', upd);

			// Clean up
			// fs.unlinkSync(audioPath);
			// fs.unlinkSync(`${audioPath}.json`);
			// fs.unlinkSync(ffmpegPath);
			// fs.unlinkSync(`${transcriptPath}.json`);
			fs.rm(baseUrl, { recursive: true, force: true }, (err) => {
				if (err) throw err;
			});
			await job.log(`Cleaned up temporary files for ${recordId}`);

			console.log(`Job ${job.id} for record ${recordId} completed successfully`);
			job.updateProgress(100); // Mark job progress as complete
		} catch (error) {
			await job.log(`Error: ${error.message}`);
			console.error(error);
		}
	},
	{
		connection: { host: env.REDIS_HOST, port: env.REDIS_PORT }, // Redis connection
		concurrency: concur || 1 // Allows multiple jobs to run concurrently
	}
);

// Helper function to format time from RTTM file into HH:mm:ss,SSS
function formatRttmTimestamp(seconds) {
	const hours = Math.floor(seconds / 3600)
		.toString()
		.padStart(2, '0');
	const minutes = Math.floor((seconds % 3600) / 60)
		.toString()
		.padStart(2, '0');
	const secs = (seconds % 60).toFixed(3).padStart(6, '0');
	return `${hours}:${minutes}:${secs}`;
}

function parseRttm(text) {
	const lines = text.split('\n');
	return lines
		.map((line) => {
			const parts = line.split(' ');
			if (parts.length >= 5) {
				const startTime = parseFloat(parts[3]);
				const duration = parseFloat(parts[4]);
				const speaker = parts[7]; // Assuming speaker info is in column 8
				return { startTime, duration, speaker };
			}
		})
		.filter(Boolean);
}

async function splitAudioIntoSegments(audioPath, segments, outputDir, job) {
	const segmentPaths = [];
	for (let i = 0; i < segments.length; i++) {
		const { startTime, duration } = segments[i];
		const outputFileName = path.resolve(outputDir, `segment_${i}.wav`);
		segmentPaths.push(outputFileName);

		const ffmpegCmd = `ffmpeg -i ${audioPath} -ss ${startTime} -t ${duration} -c copy ${outputFileName}`;
		await execCommandWithLogging(ffmpegCmd, job);
		await job.log(`Segment ${i} saved as ${outputFileName}`);
	}
	return segmentPaths;
}

function preprocessWordTimestamps(wordTimestamps) {
	const cleanedTimestamps = [];
	let previousWord = null;

	wordTimestamps.forEach((word, index) => {
		const text = word.text.trim();

		// Handle periods and other punctuation
		if (text === '.') {
			if (previousWord) {
				// Append the period to the previous word
				previousWord.text += text;
				previousWord.timestamps.to = word.timestamps.to;
			}
		} else if (text.startsWith("'")) {
			// Append apostrophe-starting words to the previous word
			if (previousWord) {
				previousWord.text += text;
				previousWord.timestamps.to = word.timestamps.to;
			}
		} else if (text.length === 1 && text !== 'a' && text !== 'i' && text !== 'I') {
			// Handle single character words (except "a")
			// if (previousWord) {
			//     // Append single character to the previous word
			//     previousWord.text += ` ${text}`;
			//     previousWord.timestamps.to = word.timestamps.to;
			// } else if (index + 1 < wordTimestamps.length) {
			//     // If no previous word, prepend to the next word
			//     const nextWord = wordTimestamps[index + 1];
			//     nextWord.text = `${text} ${nextWord.text}`;
			//     nextWord.timestamps.from = word.timestamps.from;
			// }
			console.log('deleting char');
		} else if (text.length === 1 && (text === 'a' || text === 'I' || text === 'i')) {
			// Keep "a" as a separate word
			cleanedTimestamps.push(word);
			previousWord = word;
		} else {
			// Remove other single-character symbols (e.g., parentheses, commas)
			if (!/^[\.,!?;:()\[\]]$/.test(text)) {
				cleanedTimestamps.push(word);
				previousWord = word;
			}
		}
	});

	return cleanedTimestamps;
}

function generateTranscript(wordys, rttmString) {
	const speakerSegments = parseRttm(rttmString);
	const wordTimestamps = preprocessWordTimestamps(wordys);

	const finalTranscript = [];
	let currentSegment = {
		text: '',
		timestamps: { from: null, to: null },
		speaker: null
	};

	wordTimestamps.forEach((word) => {
		const wordStart = word.offsets.from;
		const wordEnd = word.offsets.to;

		const matchingSpeakerSegment = speakerSegments.find((speakerSegment) => {
			const speakerStart = speakerSegment.startTime * 1000;
			const speakerEnd = speakerStart + speakerSegment.duration * 1000;
			return wordEnd >= speakerStart && wordEnd <= speakerEnd;
		});

		const assignedSpeaker = matchingSpeakerSegment
			? matchingSpeakerSegment.speaker
			: currentSegment.speaker;

		if (!matchingSpeakerSegment) {
			console.log('---------> Speaker unknown');
		}

		// If the current segment is for the same speaker, append the word
		if (currentSegment.speaker === assignedSpeaker) {
			currentSegment.text += word.text;
			currentSegment.timestamps.to = word.timestamps.to; // Update end time
		} else if (currentSegment === null) {
			currentSegment.speaker = assignedSpeaker;
			currentSegment.text += word.text;
			currentSegment.timestamps.to = word.timestamps.to; // Update end time
		} else {
			// Push the current segment if it has text
			if (currentSegment.text.length > 0) {
				finalTranscript.push({ ...currentSegment });
			}

			// Start a new segment for the new speaker
			currentSegment = {
				text: word.text,
				timestamps: { from: word.timestamps.from, to: word.timestamps.to },
				speaker: assignedSpeaker
			};
		}
	});

	// Push the last segment if any
	if (currentSegment.text.length > 0) {
		finalTranscript.push(currentSegment);
	}

	return finalTranscript;
}

function timestampToSeconds(timestamp) {
	const [hours, minutes, seconds] = timestamp.split(':');
	const [sec, ms] = seconds.split(',');
	return (
		parseFloat(hours) * 3600 + parseFloat(minutes) * 60 + parseFloat(sec) + parseFloat(ms) / 1000
	);
}
