import { json, RequestHandler } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { ensureCollectionExists } from '$lib/fileFuncs';

// const pb = new PocketBase('http://localhost:8090');
export const GET: RequestHandler = async ({ request, locals }) => {
	const pb = locals.pb;
	ensureCollectionExists(pb);

	try {
		const records = await pb.collection('scribo').getFullList();
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

// New DELETE endpoint to delete a template by ID
export const DELETE: RequestHandler = async ({ url, locals }) => {
	const pb = locals.pb;
	await ensureCollectionExists(pb);

	try {
		// Extract the 'id' query parameter from the request URL
		const id = url.searchParams.get('id');

		if (!id) {
			return new Response(JSON.stringify({ error: 'Record ID is required' }), {
				status: 400,
				headers: { 'Content-Type': 'application/json' }
			});
		}

		// Delete the record from the PocketBase collection 'templates'
		await pb.collection('scribo').delete(id);

		// Respond with a success message
		return new Response(JSON.stringify({ message: 'Record deleted successfully' }), {
			status: 200,
			headers: { 'Content-Type': 'application/json' }
		});
	} catch (err) {
		console.log('API records | Error deleting record', err);
		return new Response(JSON.stringify({ error: err.message }), {
			status: 500,
			headers: { 'Content-Type': 'application/json' }
		});
	}
};

export const POST: RequestHandler = async ({ request, locals }) => {
	const pb = locals.pb;
	await ensureCollectionExists(pb);

	try {
		// Get the JSON data from the request
		const data = await request.json();

		// Check if there is at least one valid field to update
		if (!data.title) {
			return new Response(JSON.stringify({ error: 'At least one field (title) is required' }), {
				status: 400,
				headers: { 'Content-Type': 'application/json' }
			});
		}

		// Prepare the object with the provided fields only
		const updateData: { title?: string } = {};

		if (data.title) {
			updateData.title = data.title;
		}

		// Create or update the record in the PocketBase collection 'templates'
		const newRecord = await pb.collection('scribo').create(updateData);

		// Respond with the created or updated record
		return new Response(JSON.stringify(newRecord), {
			status: 201,
			headers: { 'Content-Type': 'application/json' }
		});
	} catch (err) {
		console.log('API records | Error creating/updating record', err);
		return new Response(JSON.stringify({ error: err.message }), {
			status: 500,
			headers: { 'Content-Type': 'application/json' }
		});
	}
};
