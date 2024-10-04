import { RequestHandler } from '@sveltejs/kit';
import { ensureCollectionExists } from '$lib/fileFuncs';
import { transcriptionQueue } from '$lib/queue';

export const POST: RequestHandler = async ({ request, locals }) => {
	try {
		// Get the form data (including file) from the request
		const formData = await request.formData();
		const file = formData.get('audio') as File | null;
		console.log('from upload file', file);

		if (!file) {
			return new Response(JSON.stringify({ message: 'No file uploaded' }), { status: 400 });
		}

		const audioBlob = new Blob([file], { type: file.type });

		const data = {
			audio: audioBlob,
			processed: false,
			title: file.name.split('.')[0],
			date: new Date().toISOString()
		};

		const record = await locals.pb.collection('scribo').create(data);

		const job = await transcriptionQueue.add('processAudio', { recordId: record.id });
		console.log('Created job:', job.id);

		return new Response(JSON.stringify({ message: 'File uploaded successfully', record }), {
			status: 200
		});
	} catch (error: any) {
		console.log(error.message);
		return new Response(JSON.stringify({ message: 'File upload failed', error: error.message }), {
			status: 500
		});
	}
};
