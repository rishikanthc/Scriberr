// routes/api/summarize/+server.ts
import { json } from '@sveltejs/kit';
import { OLLAMA_BASE_URL, AI_MODEL, OPENAI_API_KEY } from '$env/static/private';
import OpenAI from 'openai';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { processThinkingSections } from '$lib/utils';

let openai = null;
if (OLLAMA_BASE_URL === "") {
  openai = new OpenAI({
    apiKey: `${OPENAI_API_KEY}`
  })
} else {
  openai = new OpenAI({
    baseURL: `${OLLAMA_BASE_URL}`,
    apiKey: OPENAI_API_KEY !== "" ? `${OPENAI_API_KEY}` : "ollama"
  });
}

export async function POST({ request }) {
  try {
    const { fileId, prompt, transcript, processThinking = false } = await request.json();
    
    if (!fileId || !prompt || !transcript) {
      return new Response('Missing fileId, prompt, or transcript', { status: 400 });
    }

    if (!openai) {
      return new Response('OpenAI not initialized', { status: 400 });
    }

    // Update status to processing
    await db.update(audioFiles)
      .set({ 
        summaryStatus: 'processing',
        summaryPrompt: prompt
      })
      .where(eq(audioFiles.id, fileId));

    const chatCompletion = await openai.chat.completions.create({
      model: AI_MODEL,
      messages: [
        { 
          role: 'user', 
          content: `${prompt}\n\nTranscript:\n${transcript}`
        }
      ]
    });

    // Get the raw summary from the model
    const rawSummary = chatCompletion.choices[0].message.content;

    // Process the summary - keep the thinking sections in the database
    // but allow clients to request it without thinking sections
    const summary = rawSummary;

    // Update the database with the summary
    await db.update(audioFiles)
      .set({ 
        summary,
        summaryStatus: 'completed',
        summarizedAt: new Date()
      })
      .where(eq(audioFiles.id, fileId));

    // For the response, process the thinking sections if requested
    const processedSummary = processThinking 
      ? processThinkingSections(summary, 'remove').processedText
      : summary;

    return json({ 
      summary: processedSummary,
      hasThinking: processThinkingSections(summary).hasThinkingSections
    });

  } catch (error) {
    console.error('Summarization error:', error);
    
    // Update status to failed and store error
    if (fileId) {
      await db.update(audioFiles)
        .set({ 
          summaryStatus: 'failed',
          lastError: error.message
        })
        .where(eq(audioFiles.id, fileId));
    }

    return new Response('Failed to generate summary', { 
      status: 500 
    });
  }
}