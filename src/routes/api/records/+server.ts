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
