import { RequestHandler } from '@sveltejs/kit';
import { ensureCollectionExists } from '$lib/fileFuncs';
import { wizardQueue } from '$lib/wizardQueue'; 

export const POST: RequestHandler = async ({ request, locals }) => {
	try {
		// Get the form data (including file) from the request

		const formData = await request.formData();
		const settings = formData.get('settings');

		
		const job = await wizardQueue.add('configWizard', {
		  settings: settings
		});
		console.log('Created job:', job.id);

		return new Response(JSON.stringify({ jobId: job.id}), {
			status: 200
		});
	} catch (error: any) {
		console.log(error.message);
		return new Response(JSON.stringify({ message: 'Configuration wizard failed', error: error.message }), {
			status: 500
		});
	}
};

