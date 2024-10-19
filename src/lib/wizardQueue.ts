import { Queue, Worker } from 'bullmq';
import { execSync, spawn } from 'child_process';
import { exec } from 'child_process';
import fs from 'fs';
import path from 'path';
import PocketBase from 'pocketbase';
import { env } from '$env/dynamic/private';

// Create the queue
export const wizardQueue = new Queue('wizardQueue', {
	connection: { host: env.REDIS_HOST, port: env.REDIS_PORT }
});
const pb = new PocketBase('http://localhost:8080');
pb.autoCancellation(false);
await pb.admins.authWithPassword(env.POCKETBASE_ADMIN_EMAIL, env.POCKETBASE_ADMIN_PASSWORD);


// Remove all jobs from the queue
async function clearQueue() {
	await wizardQueue.drain();
	console.log('Queue cleared!');
}

async function cleanAllJobs() {
	// Remove all completed jobs
	await wizardQueue.clean(0, 0, 'completed');
	console.log('All completed jobs have been removed');

	// Remove all failed jobs
	await wizardQueue.clean(0, 0, 'failed');
	console.log('All failed jobs have been removed');

	// Remove all waiting jobs
	await wizardQueue.clean(0, 0, 'waiting');
	console.log('All waiting jobs have been removed');

	// Remove all active jobs (this may kill jobs that are actively being processed)
	await wizardQueue.clean(0, 0, 'active');
	console.log('All active jobs have been removed');

	// Remove all delayed jobs
	await wizardQueue.clean(0, 0, 'delayed');
	console.log('All delayed jobs have been removed');
}

async function pauseAndCleanQueue() {
	// Pause the queue to prevent new jobs from being processed
	await wizardQueue.pause();
	console.log('Queue paused');

	// Clean all jobs as described above
	await cleanAllJobs();

	// Optionally resume the queue
	await wizardQueue.resume();
	console.log('Queue resumed');
}

// pauseAndCleanQueue();

// clearQueue();

// Helper function to execute shell commands and log output
const execCommandWithLogging = (cmd, job) => {
	return new Promise((resolve, reject) => {
		const process = exec(cmd);

		// Capture stdout
		process.stdout.on('data', async (data) => {
			console.log(`stdout: ${data}`);
			await job.log(`stdout: ${data}`);
		});

		// Capture stderr (in case you want to log errors)
		process.stderr.on('data', async (data) => {
			console.error(`stderr: ${data}`);
			await job.log(`stderr: ${data}`);
		});

		// Handle process close event (when the command finishes)
		process.on('close', (code) => {
			if (code === 0) {
				resolve(true); // Command succeeded
			} else {
				reject(new Error(`Command failed with exit code ${code}`)); // Command failed
			}
		});

		// Handle possible errors during execution
		process.on('error', (err) => {
			reject(new Error(`Failed to start process: ${err.message}`));
		});
	});
};
;

export const execCommandWithLoggingSync = (cmd: string, job: any): Promise<boolean> => {
    return new Promise((resolve, reject) => {
        const process = exec(cmd, { shell: true, maxBuffer: 1024 * 1024 * 10 }); // Max buffer for larger outputs

        process.stdout.on('data', async (data) => {
            console.log(`stdout: ${data}`);
            await job.log(`stdout: ${data}`);
        });

        process.stderr.on('data', async (data) => {
            console.error(`stderr: ${data}`);
            await job.log(`stderr: ${data}`);
        });

        process.on('close', (code) => {
            if (code === 0) {
                resolve(true);
            } else {
                reject(new Error(`Command failed with exit code ${code !== null ? code : 'unknown'}`));
            }
        });

        process.on('error', (err) => {
            reject(new Error(`Failed to start process: ${err.message}`));
        });
    });
};

// Set up the worker to process jobs automatically
const worker = new Worker(
	'wizardQueue',
	async (job) => {
		console.log("hello world from wizard")
		let modelPath;
		let cmd;
		try {
			const {settings }= job.data;
			if (env.DEV_MODE) {
				modelPath = path.resolve(env.SCRIBO_FILES, 'models/whisper.cpp')
			} else {
				modelPath = path.resolve('/models/whisper.cpp')
			}
			await job.log('starting job')
			cmd = `make clean -C ${modelPath}`
			await execCommandWithLogging(cmd, job);

			cmd = `make -C ${modelPath}`;
			await execCommandWithLogging(cmd, job);
			await job.log('finished making whisper')
			job.updateProgress(50)

			const modToDownload = modelsToDownload(settings);
			console.log(modToDownload)
			await job.log(modToDownload)

			modToDownload.forEach(async (m, idx) => {
			  const cmd2 = `sh ${modelPath}/models/download-ggml-model.sh ${m}.en`;
			  await job.log(`Executing command: ${cmd2}`);
			  execCommandWithLoggingSync(cmd2, job);
			  const prg = 50 + (50 * (idx + 1) / modelsToDownload.length); // idx + 1 ensures progress increments
			  await job.updateProgress(prg);
			});

			await job.log('finished job')
		} catch(error) {
			console.log(error);
			job.log(error)
		}
		
		const settt = await pb.collection('settings').getList(1,1);

		if (settt && settt.items.length > 0) {
	    const record = settt.items[0]; // Get the first record (assuming one record is returned)

	    // Update the 'wizard' field to true
	    const updatedRecord = await pb.collection('settings').update(record.id, {
	        wizard: true
	    });

	    console.log('Updated record:', updatedRecord);
		} else {
		    console.log('No records found in settings collection');
		}

		job.updateProgress(100)
	},
	{
		connection: { host: env.REDIS_HOST, port: env.REDIS_PORT }, // Redis connection
		concurrency: 1, // Allows multiple jobs to run concurrently
		lockDuration: 500000, // Lock duration (in milliseconds), e.g., 5 minutes
    lockRenewTime: 500000
	}
);

function modelsToDownload(settings) {
		let m = [];

		const set = JSON.parse(settings)
		console.log('Settings:', settings);
	console.log('Models:', set.models);


		if (set.models.small) {
			m.push('small');
		}
		if (set.models.tiny) {
			m.push('tiny');
		}
		if (set.models.medium) {
			m.push('medium');
		}
		if (set.models.largev1) {
			m.push('large-v1');
		}

		return m;
	}
