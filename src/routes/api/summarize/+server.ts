import OpenAI from 'openai';
import type { RequestHandler } from '@sveltejs/kit';
import { json } from '@sveltejs/kit';
import { OPENAI_API_KEY } from '$env/static/private';
import { OPENAI_ENDPOINT } from '$env/static/private';
import { OPENAI_MODEL } from '$env/static/private';
import { OPENAI_ROLE } from '$env/static/private';

const openai = new OpenAI({
	baseURL: OPENAI_ENDPOINT,
	apiKey: OPENAI_API_KEY
});

export const POST: RequestHandler = async ({ request, locals, fetch }) => {
	// Expect JSON body instead of form data
	const { templateId, transcript, id } = await request.json();

	const resp = await fetch(`/api/templates/${templateId}`);
	const templateRecord = await resp.json();
	const prompt = templateRecord.prompt;
	const fullPrompt = `${transcript}\n${prompt}`;
	const completion = await openai.chat.completions.create({
		messages: [{ role: OPENAI_ROLE, content: fullPrompt }],
		model: OPENAI_MODEL
	});

	console.log(completion)

	locals.pb.collection('scribo').update(id, {
		summary: completion.choices[0].message.content
	});

	console.log(completion.choices[0]);

	return json(completion.choices[0]);
};
