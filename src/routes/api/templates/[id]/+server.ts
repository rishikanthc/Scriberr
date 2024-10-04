import { json, RequestHandler } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { ensureCollectionExists } from '$lib/fileFuncs';

export const GET: RequestHandler = async ({ request, locals, params }) => {
	const pb = locals.pb;
	ensureCollectionExists(pb);

	const { id } = params;

	try {
		if (id) {
			// Fetch specific template by id
			const record = await pb.collection('templates').getOne(id);
			return new Response(JSON.stringify(record), {
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			});
		} else {
			// Fetch all templates if no id is provided
			console.log('Missing parameter ID');
			return new Response(JSON.stringify({ error: 'Error fetching record. Missing ID' }), {
				status: 500,
				headers: { 'Content-Type': 'application/json' }
			});
		}
	} catch (err) {
		console.log('API records | Error fetching records', err);
		return new Response(JSON.stringify({ error: 'Error fetching record' }), {
			status: 500,
			headers: { 'Content-Type': 'application/json' }
		});
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
		const record = await pb.collection('templates').getOne(id);
		if (!record) {
			return new Response(JSON.stringify({ error: 'Record not found' }), {
				status: 404,
				headers: { 'Content-Type': 'application/json' }
			});
		}

		// Update the 'prompt' field of the record
		const updatedRecord = await pb.collection('templates').update(id, {
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
