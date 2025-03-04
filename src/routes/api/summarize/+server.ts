// routes/api/summarize/+server.ts
import { json } from '@sveltejs/kit';
import OpenAI from 'openai';
import { db } from '$lib/server/db';
import { audioFiles } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { processThinkingSections } from '$lib/utils';

// Force using Ollama for now to fix the authentication issue
let openai = null;
try {
  // Get environment variables directly at runtime
  const runtimeOllamaBaseUrl = process.env.OLLAMA_BASE_URL || "";
  const runtimeAiModel = process.env.AI_MODEL || "gpt-3.5-turbo";
  const runtimeOpenaiApiKey = process.env.OPENAI_API_KEY || "";
  
  console.log("Configuration values:");
  console.log("- OLLAMA_BASE_URL:", runtimeOllamaBaseUrl);
  console.log("- AI_MODEL:", runtimeAiModel);
  console.log("- OPENAI_API_KEY length:", runtimeOpenaiApiKey ? runtimeOpenaiApiKey.length : 0);
  
  // Use environment variable or fallback to default
  console.log("Using Ollama API for summarization");
  const baseUrl = runtimeOllamaBaseUrl || "http://ollama:11434/api";
  console.log(`Actual base URL being used: ${baseUrl}`);
  
  openai = new OpenAI({
    baseURL: baseUrl,
    apiKey: "ollama", // Dummy key for Ollama
    dangerouslyAllowBrowser: true // Allow browser usage
  });
  
  console.log("OpenAI client initialized with Ollama configuration");
} catch (error) {
  console.error("Error initializing OpenAI client:", error);
}

export async function POST({ request }) {
  try {
    const { fileId, prompt, transcript, processThinking = false } = await request.json();
    
    if (!fileId || !prompt || !transcript) {
      return new Response('Missing fileId, prompt, or transcript', { status: 400 });
    }

    if (!openai) {
      return new Response('OpenAI client not initialized. Check your API key or Ollama configuration.', { status: 500 });
    }
    
    const numericFileId = parseInt(fileId);

    // Update status to processing
    await db.update(audioFiles)
      .set({ 
        summaryStatus: 'processing',
        summaryPrompt: prompt
      })
      .where(eq(audioFiles.id, numericFileId));

    console.log("Sending request to Ollama for summarization...");
    
    try {
      // Try a direct fetch to Ollama first to validate connection
      const runtimeOllamaBaseUrl = process.env.OLLAMA_BASE_URL || "";
      const baseUrl = runtimeOllamaBaseUrl || "http://ollama:11434/api";
      const versionUrl = baseUrl.endsWith('/api') ? baseUrl.replace('/api', '/api/version') : `${baseUrl}/version`;
      
      console.log(`Testing connection to: ${versionUrl}`);
      const testResponse = await fetch(versionUrl, {
        method: "GET"
      });
      
      if (testResponse.ok) {
        const testData = await testResponse.json();
        console.log("Ollama connection test successful:", testData);
      } else {
        console.error("Ollama connection test failed:", await testResponse.text());
      }
      
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
        const baseUrl = runtimeOllamaBaseUrl || "http://ollama:11434/api";
        const chatUrl = baseUrl.endsWith('/api') ? `${baseUrl}/chat` : `${baseUrl}/api/chat`;
        
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