import { json } from '@sveltejs/kit';
import OpenAI from 'openai';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { processThinkingSections } from '$lib/utils';

function getOpenAI() {
  try {
    const runtimeOpenaiBaseUrl = process.env.OPENAI_BASE_URL || "";
    const runtimeOllamaBaseUrl = process.env.OLLAMA_BASE_URL || "";
    const runtimeOpenaiApiKey = process.env.OPENAI_API_KEY || "";

    // First try custom OpenAI-compatible server if configured
    if (runtimeOpenaiBaseUrl && runtimeOpenaiApiKey) {
      return new OpenAI({
        baseURL: runtimeOpenaiBaseUrl,
        apiKey: runtimeOpenaiApiKey,
        dangerouslyAllowBrowser: true
      });
    }

    // Then try Ollama if configured
    if (runtimeOllamaBaseUrl) {
      return new OpenAI({
        baseURL: runtimeOllamaBaseUrl,
        apiKey: "ollama",
        dangerouslyAllowBrowser: true
      });
    }

    // Finally try official OpenAI if API key is provided
    if (runtimeOpenaiApiKey) {
      return new OpenAI({
        apiKey: runtimeOpenaiApiKey,
        dangerouslyAllowBrowser: true
      });
    }

    return null;
  } catch (error) {
    return null;
  }
}

export async function POST({ request }) {
  try {
    const { fileId, prompt, system_prompt, transcript, processThinking = false } = await request.json();

    if (!fileId || !prompt || !transcript) {
      return new Response('Missing fileId, prompt, or transcript', { status: 400 });
    }

    const openai = getOpenAI();
    if (!openai) {
      return new Response('AI client not initialized. Check your OPENAI_BASE_URL, OLLAMA_BASE_URL or OPENAI_API_KEY configuration.', { status: 500 });
    }

    const numericFileId = parseInt(fileId);

    await db.update(audioFiles)
      .set({
        summaryStatus: 'processing',
        summaryPrompt: prompt
      })
      .where(eq(audioFiles.id, numericFileId));

    try {
      const runtimeAiModel = process.env.AI_MODEL || "gpt-3.5-turbo";
      const messages = [
        {
          role: 'user',
          content: `${prompt}\n\nTranscript:\n${transcript}`
        }
      ];

      // Add system prompt if provided
      if (system_prompt) {
        messages.unshift({
          role: 'system',
          content: system_prompt
        });
      }

      const chatCompletion = await openai.chat.completions.create({
        model: runtimeAiModel,
        messages
      });

      const rawSummary = chatCompletion.choices[0].message.content;
      const summary = rawSummary;

      await db.update(audioFiles)
        .set({
          summary,
          summaryStatus: 'completed',
          summarizedAt: new Date()
        })
        .where(eq(audioFiles.id, numericFileId));

      const processedSummary = processThinking
        ? processThinkingSections(summary, 'remove').processedText
        : summary;

      return json({
        summary: processedSummary,
        hasThinking: processThinkingSections(summary).hasThinkingSections
      });
    } catch (apiError) {
      try {
        const runtimeOllamaBaseUrl = process.env.OLLAMA_BASE_URL || "";
        if (!runtimeOllamaBaseUrl) {
          throw new Error("No OLLAMA_BASE_URL configured for fallback");
        }

        const baseUrlWithoutPath = runtimeOllamaBaseUrl.replace(/\/api.*$/, '');
        const chatUrl = `${baseUrlWithoutPath}/api/chat`;
        const runtimeAiModel = process.env.AI_MODEL || "gpt-3.5-turbo";

        const directOllamaResponse = await fetch(chatUrl, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            model: runtimeAiModel,
            messages: [
              {
                role: "user",
                content: `${prompt}\n\nTranscript:\n${transcript}`
              }
            ]
          })
        });

        if (!directOllamaResponse.ok) {
          throw new Error(`Direct Ollama call failed: ${await directOllamaResponse.text()}`);
        }

        const ollamaData = await directOllamaResponse.json();
        const directSummary = ollamaData.message?.content || "Summary could not be generated.";

        await db.update(audioFiles)
          .set({
            summary: directSummary,
            summaryStatus: 'completed',
            summarizedAt: new Date()
          })
          .where(eq(audioFiles.id, numericFileId));

        const processedSummary = processThinking
          ? processThinkingSections(directSummary, 'remove').processedText
          : directSummary;

        return json({
          summary: processedSummary,
          hasThinking: processThinkingSections(directSummary).hasThinkingSections
        });

      } catch (fallbackError) {
        await db.update(audioFiles)
          .set({
            summaryStatus: 'failed',
            lastError: `${apiError.message} | Fallback error: ${fallbackError.message}`
          })
          .where(eq(audioFiles.id, numericFileId));

        return new Response(`API error: ${apiError.message}`, { status: 500 });
      }
    }
  } catch (error) {
    try {
      const { fileId } = await request.json();
      if (fileId) {
        const numericFileId = parseInt(fileId);
        await db.update(audioFiles)
          .set({
            summaryStatus: 'failed',
            lastError: error.message
          })
          .where(eq(audioFiles.id, numericFileId));
      }
    } catch (err) {
      console.error('Error updating file status:', err);
    }

    return new Response('Failed to generate summary', {
      status: 500
    });
  }
}