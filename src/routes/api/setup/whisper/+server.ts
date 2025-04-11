import { exec } from 'child_process';
import { db } from '$lib/server/db';
import { systemSettings } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { promisify } from 'util';
import { mkdir } from 'fs/promises';
import { join } from 'path';
import type { RequestHandler } from './$types';
import { platform } from 'os';
import { env } from 'process';

const execAsync = promisify(exec);
const isWindows = platform() === 'win32';
const maxBuffer = 1024 * 1024 * 100; // 100MB buffer

export const GET: RequestHandler = async ({ url }) => {
   console.log("Whisper setup API route called with params:", {
     models: url.searchParams.get('models'),
     multilingual: url.searchParams.get('multilingual'),
     diarization: url.searchParams.get('diarization'),
     compute_type: url.searchParams.get('compute_type'),
     hasHfKey: !!url.searchParams.get('hf_api_key'),
     test: url.searchParams.get('test')
   });
   
   // Special case for test requests - skip EventSource
   if (url.searchParams.get('test') === 'true') {
     console.log("TEST MODE: Returning direct JSON response without EventSource");
     
     // Log DB connection for debugging
     try {
       const testSettings = await db.select().from(systemSettings).limit(1);
       console.log("DB connection successful during test, found settings:", testSettings);
       
       return new Response(JSON.stringify({
         success: true,
         message: "API endpoint is functional",
         dbTest: {
           connected: true,
           settings: testSettings.length > 0 ? "found" : "empty"
         },
         env: {
           NODE_ENV: process.env.NODE_ENV || "not set",
           DATABASE_URL: process.env.DATABASE_URL ? "set (hidden)" : "not set"
         }
       }), {
         status: 200,
         headers: {
           'Content-Type': 'application/json'
         }
       });
     } catch (dbError) {
       console.error("Database connection error during test:", dbError);
       return new Response(JSON.stringify({
         success: false,
         error: "Database connection failed: " + (dbError instanceof Error ? dbError.message : "Unknown error"),
         env: {
           NODE_ENV: process.env.NODE_ENV || "not set",
           DATABASE_URL: process.env.DATABASE_URL ? "set (hidden)" : "not set"
         }
       }), {
         status: 500,
         headers: {
           'Content-Type': 'application/json'
         }
       });
     }
   }

   const models = JSON.parse(url.searchParams.get('models') || '[]');
   const multilingual = url.searchParams.get('multilingual') === 'true';
   const enableDiarization = url.searchParams.get('diarization') === 'true';
   const computeType = url.searchParams.get('compute_type') || 'float32';
   const hfApiKey = url.searchParams.get('hf_api_key') || '';

   // Get environment variables
   const modelsDir = env.MODELS_DIR || '/scriberr/models';
   const diarizationModel = env.DIARIZATION_MODEL || 'pyannote/speaker-diarization@3.1';
   
   // Test DB connection first
   try {
     const testSettings = await db.select().from(systemSettings).limit(1);
     console.log("DB connection successful, found settings:", testSettings);
   } catch (dbError) {
     console.error("Database connection error:", dbError);
     return new Response(JSON.stringify({
       error: "Database connection failed: " + (dbError instanceof Error ? dbError.message : "Unknown error")
     }), {
       status: 500,
       headers: {
         'Content-Type': 'application/json'
       }
     });
   }
   
   const stream = new ReadableStream({
       async start(controller) {
           try {
               const baseDir = join(process.cwd(), 'whisper');
               const whisperDir = join(baseDir, 'whisper.cpp');
               const diarizeDir = join(process.cwd(), 'diarize');

               console.log("SETUP LOGS --->", baseDir, whisperDir, diarizeDir);
               
               const sendMessage = (message: string, progress?: number, status?: string) => {
                   controller.enqueue(`data: ${JSON.stringify({ message, progress, status })}\n\n`);
               };

               const execWithProgress = async (command: string, options: any) => {
                   try {
                       const { stdout, stderr } = await execAsync(command, { 
                           ...options,
                           maxBuffer
                       });
                       return { stdout, stderr };
                   } catch (err) {
                       throw err;
                   }
               };

               await mkdir(baseDir, { recursive: true });
               await mkdir(modelsDir, { recursive: true });
               
               const gitCommand = isWindows ? 'git.exe' : 'git';
               const makeCommand = isWindows ? 'mingw32-make' : 'make';
               const shellScript = isWindows ? 'bash.exe' : 'bash';
               const pipCommand = isWindows ? 'pip.exe' : 'pip';
               const pythonCommand = isWindows ? 'python.exe' : 'python';

               // Configure environment to bypass HuggingFace authentication for whisper models
               env.HF_HUB_DISABLE_TELEMETRY = "1";
               env.HF_HUB_ENABLE_HF_TRANSFER = "0";
               env.TRUST_REMOTE_CODE = "1";
               
               // Don't set these if we have an API key
               if (!hfApiKey) {
                  env.HF_TOKEN = "";
                  env.HF_HUB_OFFLINE = "0";
               }

               sendMessage('Installing dependencies...', 10);
               try {
                   await execWithProgress(
                       `uv pip install torch torchaudio --index-url https://download.pytorch.org/whl/cpu`,
                       { shell: true }
                   );
               } catch (err) {
                   sendMessage(`INFO: PyTorch might be already installed, continuing...`, 15);
               }

               sendMessage('Installing WhisperX...', 20);
               try {
                   await execWithProgress(
                       `uv pip install -U whisperx huggingface_hub`,
                       { shell: true }
                   );
               } catch (err) {
                   throw new Error(`Failed to install WhisperX: ${err instanceof Error ? err.message : 'Unknown error'}`);
               }

               // Download selected Whisper models
               let currentProgress = 30;
               const totalModelsToDownload = models.length + (enableDiarization ? 1 : 0);
               const progressPerModel = totalModelsToDownload > 0 ? 60 / totalModelsToDownload : 60;
               
               if (models.length > 0) {
                   for (const model of models) {
                       sendMessage(`Downloading Whisper ${model} model...`, currentProgress);
                       
                       try {
                           // Create Python script to download model
                           const downloadScript = `
import whisperx
import sys
import os

# Download model and save to models directory
os.environ["HF_HUB_DISABLE_TELEMETRY"] = "1"
os.environ["HF_HUB_ENABLE_HF_TRANSFER"] = "0"
os.environ["TRUST_REMOTE_CODE"] = "1"
os.environ["HF_TOKEN"] = ""
os.environ["HF_HUB_OFFLINE"] = "0"

# Download whisper model
model = whisperx.load_model("${model}", device="cpu", download_root="${modelsDir}", compute_type="${computeType}")
print(f"Successfully downloaded {model} to ${modelsDir}")
                           `;
                           
                           // Write temporary Python script
                           const scriptPath = join(baseDir, `download_${model.replace(/\./g, '_')}.py`);
                           await execWithProgress(`echo '${downloadScript}' > ${scriptPath}`, { shell: true });
                           
                           // Execute the script
                           await execWithProgress(`${pythonCommand} ${scriptPath}`, { shell: true });
                           currentProgress += progressPerModel;
                       } catch (err) {
                           sendMessage(`Warning: Failed to download ${model} model: ${err instanceof Error ? err.message : 'Unknown error'}`, currentProgress);
                           currentProgress += progressPerModel;
                       }
                   }
               } else {
                   sendMessage('No Whisper models selected, skipping model download', 40);
                   currentProgress = 80;
               }

               // Download diarization model if enabled
               if (enableDiarization) {
                   sendMessage(`Downloading diarization model (${diarizationModel})...`, currentProgress);
                   
                   if (!hfApiKey) {
                      sendMessage(`Error: HuggingFace API key is required for diarization model download`, currentProgress);
                      throw new Error('HuggingFace API key is required for diarization model download');
                   }
                   
                   try {
                       // Create Python script to download diarization model using API key
                       const diarizeScript = `
from pyannote.audio import Pipeline
import os

# Set environment variables for authentication
os.environ["HF_HUB_DISABLE_TELEMETRY"] = "1"
os.environ["HF_HUB_ENABLE_HF_TRANSFER"] = "0"
os.environ["TRUST_REMOTE_CODE"] = "1"

# Initialize diarization pipeline using API key
pipeline = Pipeline.from_pretrained(
    "${diarizationModel}",
    use_auth_token="${hfApiKey}"
)

# Create models directory and save model
os.makedirs("${modelsDir}/pyannote", exist_ok=True)
pipeline.to_disk("${modelsDir}/pyannote/speaker-diarization")
print(f"Successfully downloaded diarization model to ${modelsDir}/pyannote")
                       `;
                       
                       // Write temporary Python script
                       const diarizeScriptPath = join(baseDir, "download_diarize.py");
                       await execWithProgress(`echo '${diarizeScript}' > ${diarizeScriptPath}`, { shell: true });
                       
                       // Execute the script
                       await execWithProgress(`${pythonCommand} ${diarizeScriptPath}`, { shell: true });
                   } catch (err) {
                       sendMessage(`Warning: Failed to download diarization model: ${err instanceof Error ? err.message : 'Unknown error'}`, 90);
                   }
               } else {
                   sendMessage('Diarization disabled, skipping diarization model download', currentProgress);
               }

               sendMessage('Installation completed successfully!', 99);
               sendMessage('Saving configuration settings!', 100);

               // First check if settings exist, if not create one
               const existingSettings = await db.select().from(systemSettings).limit(1);
               
               if (existingSettings.length === 0) {
                  sendMessage('Creating system settings record...', 100);
                  // Create default settings
                  await db.insert(systemSettings).values({
                    isInitialized: true,
                    firstStartupDate: new Date(),
                    lastStartupDate: new Date(),
                    whisperModelSizes: models,
                  });
                  sendMessage('Database configuration completed!', 100, 'complete');
               } else {
                  // Update existing settings
                  await db.update(systemSettings).set({
                    isInitialized: true,
                    firstStartupDate: new Date(),
                    lastStartupDate: new Date(),
                    whisperModelSizes: models,
                  }).where(eq(systemSettings.id, existingSettings[0].id));
                  sendMessage('Database configuration updated!', 100, 'complete');
               }
            } catch (err) {
               console.error('Setup error:', err);
               controller.enqueue(`data: ${JSON.stringify({
                   message: `Error: ${err instanceof Error ? err.message : 'Unknown error'}`,
                   status: 'error'
               })}\n\n`);
           } finally {
               controller.close();
           }
       }
   });

   return new Response(stream, {
       headers: {
           'Content-Type': 'text/event-stream',
           'Cache-Control': 'no-cache',
           'Connection': 'keep-alive'
       }
   });
};