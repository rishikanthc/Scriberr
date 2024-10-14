import { json, RequestHandler } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { ensureCollectionExists } from '$lib/fileFuncs';

// const pb = new PocketBase('http://localhost:8090');
export const GET: RequestHandler = async ({ request, locals }) => {
	const pb = locals.pb;
	ensureCollectionExists(pb);

	try {
		const records = await pb.collection('templates').getFullList();
		return new Response(JSON.stringify(records), {
			status: 200,
			headers: { 'Content-Type': 'application/json' }
		});
	} catch (err) {
		console.log('API records | Error fetching records', err);
		return new Response(JSON.stringify(err), {
			status: 200,
			headers: { 'Content-Type': 'application/json' }
		});
	}

	return new Response(JSON.stringify('Success'), {
		status: 200,
		headers: { 'Content-Type': 'application/json' }
	});
};

export const POST: RequestHandler = async ({ request, locals }) => {
	const pb = locals.pb;
	await ensureCollectionExists(pb);

	try {
		// Get the JSON data from the request
		const { title, prompt } = await request.json();

		// Validate the input data (optional but recommended)
		if (!title || !prompt) {
			return new Response(JSON.stringify({ error: 'Title and prompt are required' }), {
				status: 400,
				headers: { 'Content-Type': 'application/json' }
			});
		}

		// Create the new record in the PocketBase collection 'templates'
		const newRecord = await pb.collection('templates').create({
			title,
			prompt
		});

		// Respond with the created record
		return new Response(JSON.stringify(newRecord), {
			status: 201,
			headers: { 'Content-Type': 'application/json' }
		});
	} catch (err) {
		console.log('API records | Error creating record', err);
		return new Response(JSON.stringify({ error: err.message }), {
			status: 500,
			headers: { 'Content-Type': 'application/json' }
		});
	}
};
