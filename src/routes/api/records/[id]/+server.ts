import { json, RequestHandler } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { ensureCollectionExists } from '$lib/fileFuncs';

export const GET: RequestHandler = async ({ params, locals }) => {
	const { id } = params;

	try {
		// Fetch record by id from PocketBase collection 'scribo'
		const record = await locals.pb.collection('scribo').getOne(id);

		return json({
			record
		});
	} catch (error) {
		console.error('Failed to fetch record details:', error);
		return json({ message: 'Failed to fetch record details' }, { status: 500 });
	}
};

export const POST: RequestHandler = async ({ request, params, locals }) => {
	const pb = locals.pb;
	const { id } = params;
	ensureCollectionExists(pb);

	try {
		// Get the body from the request (containing the new 'prompt' value)
		const data = await request.json();
		const prompt = data.prompt;

		// Find the record by its id
		const record = await pb.collection('scribo').getOne(id);
		if (!record) {
			return new Response(JSON.stringify({ error: 'Record not found' }), {
				status: 404,
				headers: { 'Content-Type': 'application/json' }
			});
		}

		// Update the 'prompt' field of the record
		const updatedRecord = await pb.collection('scribo').update(id, {
			prompt: prompt
		});

		return new Response(JSON.stringify(updatedRecord), {
			status: 200,
			headers: { 'Content-Type': 'application/json' }
		});
	} catch (err) {
		console.log('API records | Error updating record', err);
		return new Response(JSON.stringify(err), {
			status: 500,
			headers: { 'Content-Type': 'application/json' }
		});
	}
};
