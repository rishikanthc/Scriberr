import { json, RequestHandler } from '@sveltejs/kit';
import PocketBase from 'pocketbase';
import { ensureCollectionExists } from '$lib/fileFuncs';

export const GET: RequestHandler = async ({ request, locals }) => {
	ensureCollectionExists(locals.pb);
	const records = await locals.pb.collection('settings').getList(1, 1);

	if (records.items.length > 1) {
		throw new Error('Duplicate settings items found');
	}
	const settings = records.items[0];

	return new Response(JSON.stringify(settings), {
		status: 200,
		headers: { 'Content-Type': 'application/json' }
	});
};

export const POST: RequestHandler = async ({ request, locals }) => {
	try {
		// Ensure the settings collection exists
		ensureCollectionExists(locals.pb);

		// Parse the incoming request body
		const requestBody = await request.json();

		// Fetch the existing settings record (assuming there's one record in the 'settings' collection)
		const settingsList = await locals.pb.collection('settings').getList(1, 1);
		if (settingsList.items.length === 0) {
			return new Response(JSON.stringify({ error: 'Settings record not found' }), {
				status: 404,
				headers: { 'Content-Type': 'application/json' }
			});
		}

		const settingsId = settingsList.items[0].id; // Get the ID of the first (and likely only) settings record

		// Update the settings record with the new values from the request body
		const updatedSettings = await locals.pb.collection('settings').update(settingsId, requestBody);

		// Return a success response
		return new Response(
			JSON.stringify({ message: 'Settings updated successfully', updatedSettings }),
			{
				status: 200,
				headers: { 'Content-Type': 'application/json' }
			}
		);
	} catch (error) {
		// Handle any errors that occur during the process
		console.error('Error updating settings:', error);
		return new Response(JSON.stringify({ error: 'Failed to update settings' }), {
			status: 500,
			headers: { 'Content-Type': 'application/json' }
		});
	}
};
