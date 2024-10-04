import { json } from '@sveltejs/kit';

export async function load({ params, fetch, locals }) {
	const pb = locals.pb;
	const response = await fetch('/api/records');
	const records = await response.json();

	const response2 = await fetch('/api/templates');
	const templates = await response2.json();
	console.log('templates ===== ', templates);

	// Use map to create an array of promises
	const fileUrlPromises = records.map(async (value) => {
		const record = await pb.collection('scribo').getOne(value.id);
		const selected_file = pb.getFileUrl(record, record.audio);

		return { selected_file, id: record.id };
	});

	// Wait for all promises to resolve
	const fileUrls = await Promise.all(fileUrlPromises);

	console.log('Home page - fetched', records.length);

	// Return both records and fileUrls
	return { records, fileUrls, templates };
}
