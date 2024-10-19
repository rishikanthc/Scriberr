import { RequestHandler } from '@sveltejs/kit';
import { ensureCollectionExists } from '$lib/fileFuncs';
import { wizardQueue } from '$lib/wizardQueue'; 

export const GET: RequestHandler = async ({ params, locals }) => {
  const {id} = params

try {

		if (!id) {
		return new Response(JSON.stringify({ message: 'Job ID required', error: error.message }), {
			status: 500
		});
		}

    const job = await wizardQueue.getJob(id);

		if (!job) {
			return new Response(JSON.stringify({ message: 'Job not found' }), { status: 404 });
		}

		// Get the job progress
		const progress = job.progress;
		const {logs} = await wizardQueue.getJobLogs(id);

		return new Response(JSON.stringify({ jobId: job.id, progress, logs }), {
			status: 200
		});
	} catch (error: any) {
		console.log(error.message);
		return new Response(JSON.stringify({ message: 'Configuration wizard failed', error: error.message }), {
			status: 500
		});
	}};
