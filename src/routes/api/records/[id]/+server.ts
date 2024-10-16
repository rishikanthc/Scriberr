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
		// Get the body from the request (containing the new 'title' and/or 'prompt' values)
		const data = await request.json();
		const { title, transcript, diarizedtranscript } = data;

		// Ensure that at least one of 'title' or 'prompt' is provided
		if (!title && !transcript && !diarizedtranscript) {
			return new Response(JSON.stringify({ error: 'Title must be provided' }), {
				status: 400,
				headers: { 'Content-Type': 'application/json' }
			});
		}

		
		// Find the record by its id
		const record = await pb.collection('scribo').getOne(id);
		if (!record) {
			return new Response(JSON.stringify({ error: 'Record not found' }), {
				status: 404,
				headers: { 'Content-Type': 'application/json' }
			});
		}

		// Prepare the update object with only the fields that are provided
		const updateData: { title?: string; transcript?: object, diarizedtranscript?: object} = {};
		if (title) updateData.title = title;
		if (transcript) updateData.transcript = transcript;
		if (diarizedtranscript) updateData.diarizedtranscript = diarizedtranscript;

		// Update the record with the provided data
		const updatedRecord = await pb.collection('scribo').update(id, updateData);

		return new Response(JSON.stringify(updatedRecord), {
			status: 200,
			headers: { 'Content-Type': 'application/json' }
		});
	} catch (err) {
		console.log('API records | Error updating record', err);
		return new Response(JSON.stringify({ error: err.message }), {
			status: 500,
			headers: { 'Content-Type': 'application/json' }
		});
	}
};
