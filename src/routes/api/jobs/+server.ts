import { transcriptionQueue } from '$lib/queue'; // Adjust the path as per your project structure
import { json } from '@sveltejs/kit';

export async function GET() {
	try {
		// Fetch pending jobs
		const waitingJobs = await transcriptionQueue.getWaiting();
		const activeJobs = await transcriptionQueue.getActive();
		const delayedJobs = await transcriptionQueue.getDelayed();
		const failedJobs = await transcriptionQueue.getFailed();

		// Prepare the response data
		const formatJobs = (jobs) =>
			jobs.map((job) => ({
				id: job.id,
				name: job.name,
				data: job.data,
				progress: job.progress,
				timestamp: job.timestamp,
				attemptsMade: job.attemptsMade
			}));

		const jobs = {
			waiting: formatJobs(waitingJobs),
			active: formatJobs(activeJobs),
			delayed: formatJobs(delayedJobs),
			failed: formatJobs(failedJobs)
		};

		return json(jobs);
	} catch (err) {
		console.error('Error fetching jobs:', err);
		return json({ error: 'Failed to fetch jobs' }, { status: 500 });
	}
}
