// routes/api/summarize/+server.ts
import { json } from '@sveltejs/kit';
import OpenAI from 'openai';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { processThinkingSections } from '$lib/utils';

// Initialize OpenAI client at request time, not at build time
// This ensures we use the runtime environment variables
function getOpenAI() {
  try {
    // Only run if debug environemnt variable is enabled
    // Get ALL environment variables for debugging
    if (process.env.DEBUG === "true") {
        console.log("ALL ENV VARIABLES:");
        console.log(JSON.stringify(process.env, null, 2));
    }
    
    // Get environment variables at runtime (when function is called)
    const runtimeOllamaBaseUrl = process.env.OLLAMA_BASE_URL || "";
    const runtimeAiModel = process.env.AI_MODEL || "gpt-3.5-turbo";
    const runtimeOpenaiApiKey = process.env.OPENAI_API_KEY || "";
    
    console.log("Configuration values at request time:");
    console.log("- OLLAMA_BASE_URL:", runtimeOllamaBaseUrl);
    console.log("- AI_MODEL:", runtimeAiModel);
    console.log("- OPENAI_API_KEY length:", runtimeOpenaiApiKey ? runtimeOpenaiApiKey.length : 0);
    console.log("- NODE_ENV:", process.env.NODE_ENV);
    
    // IMPORTANT: Force Ollama usage for now to debug
    if (runtimeOllamaBaseUrl) {
      console.log("Using Ollama API for summarization with URL:", runtimeOllamaBaseUrl);
      
      const ollamaClient = new OpenAI({
        baseURL: runtimeOllamaBaseUrl,
        apiKey: "ollama", // Dummy key for Ollama
        dangerouslyAllowBrowser: true // Allow browser usage
      });
      
      // Print debug info about the client config
      console.log("OpenAI client configuration:", {
        baseURL: ollamaClient.baseURL,
        apiKey: ollamaClient.apiKey ? "[REDACTED]" : "missing",
        defaultHeaders: ollamaClient.defaultHeaders,
        defaultQuery: ollamaClient.defaultQuery
      });
      
      return ollamaClient;
    } else {
      // For safety, don't use OpenAI if we have no Ollama URL
      console.error("No OLLAMA_BASE_URL configured - refusing to create client");
      return null;
    }
  } catch (error) {
    console.error("Error initializing OpenAI client:", error);
    return null;
  }
}

export async function POST({ request }) {
  try {
    const { fileId, prompt, transcript, processThinking = false } = await request.json();
    
    if (!fileId || !prompt || !transcript) {
      return new Response('Missing fileId, prompt, or transcript', { status: 400 });
    }

    // Initialize the OpenAI client at request time
    const openai = getOpenAI();
    
    if (!openai) {
      return new Response('OpenAI client not initialized. Check your OLLAMA_BASE_URL configuration.', { status: 500 });
    }
    
    const numericFileId = parseInt(fileId);

    // Update status to processing
    await db.update(audioFiles)
      .set({ 
        summaryStatus: 'processing',
        summaryPrompt: prompt
      })
      .where(eq(audioFiles.id, numericFileId));

    console.log("Sending request to AI for summarization...");
    
    try {
      // Get the runtime model name
      const runtimeAiModel = process.env.AI_MODEL || "gpt-3.5-turbo";
      
      // Now proceed with the OpenAI library call
      const chatCompletion = await openai.chat.completions.create({
        model: runtimeAiModel,
        messages: [
          { 
            role: 'user', 
            content: `${prompt}\n\nTranscript:\n${transcript}`
          }
        ]
      });

      // Get the raw summary from the model
      const rawSummary = chatCompletion.choices[0].message.content;
      console.log("Summary generated successfully");

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
        .where(eq(audioFiles.id, numericFileId));

      // For the response, process the thinking sections if requested
      const processedSummary = processThinking 
        ? processThinkingSections(summary, 'remove').processedText
        : summary;

      return json({ 
        summary: processedSummary,
        hasThinking: processThinkingSections(summary).hasThinkingSections
      });
    } catch (apiError) {
      console.error("API error details:", apiError);
      
      // Try a fallback direct request to Ollama's native API
      try {
        console.log("Attempting direct Ollama API call as fallback");
        
        // Get the base URL from environment variables at runtime
        const runtimeOllamaBaseUrl = process.env.OLLAMA_BASE_URL || "";
        if (!runtimeOllamaBaseUrl) {
          throw new Error("No OLLAMA_BASE_URL configured for fallback");
        }
        
        // Extract the base URL without any API paths
        const baseUrlWithoutPath = runtimeOllamaBaseUrl.replace(/\/api.*$/, '');
        const chatUrl = `${baseUrlWithoutPath}/api/chat`;
        
        console.log(`Sending direct request to: ${chatUrl}`);
        
        // Get the runtime model name
        const runtimeAiModel = process.env.AI_MODEL || "gpt-3.5-turbo";
        
        const directOllamaResponse = await fetch(chatUrl, {
          method: "POST",
          headers: {
            "Content-Type": "application/json"
          },
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
        
        if (directOllamaResponse.ok) {
          const ollamaData = await directOllamaResponse.json();
          console.log("Direct Ollama call succeeded");
          
          const directSummary = ollamaData.message?.content || "Summary could not be generated.";
          
          // Update the database with the summary from direct call
          await db.update(audioFiles)
            .set({ 
              summary: directSummary,
              summaryStatus: 'completed',
              summarizedAt: new Date()
            })
            .where(eq(audioFiles.id, numericFileId));
            
          return json({ 
            summary: directSummary,
            hasThinking: processThinkingSections(directSummary).hasThinkingSections
          });
        } else {
          throw new Error(`Direct Ollama call failed: ${await directOllamaResponse.text()}`);
        }
      } catch (fallbackError) {
        console.error("Fallback attempt also failed:", fallbackError);
        
        // Update status to failed
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
    console.error('Summarization error:', error);
    
    // Update status to failed and store error
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