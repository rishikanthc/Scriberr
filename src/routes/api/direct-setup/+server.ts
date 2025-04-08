import { json } from '@sveltejs/kit';
import { db } from '$lib/server/db';
import { systemSettings } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import type { RequestHandler } from './$types';
import { exec } from 'child_process';
import { promisify } from 'util';
import { mkdir } from 'fs/promises';
import { join } from 'path';
import { platform } from 'os';
import { env } from 'process';

const execAsync = promisify(exec);
const isWindows = platform() === 'win32';
const maxBuffer = 1024 * 1024 * 100; // 100MB buffer

export const GET: RequestHandler = async ({ url }) => {
  console.log("Direct setup endpoint called");
  
  // Parse query parameters
  const models = url.searchParams.get('models') ? JSON.parse(url.searchParams.get('models') || '[]') : ['base'];
  const multilingual = url.searchParams.get('multilingual') === 'true';
  const enableDiarization = url.searchParams.get('diarization') === 'true';
  const computeType = url.searchParams.get('compute_type') || 'float32';
  const hfApiKey = url.searchParams.get('hf_api_key') || '';
  
  // Get environment variables
  const modelsDir = env.MODELS_DIR || '/scriberr/models';
  const diarizationModel = env.DIARIZATION_MODEL || 'pyannote/speaker-diarization@3.1';
  
  try {
    // Get any existing settings
    const existingSettings = await db.select().from(systemSettings).limit(1);
    console.log("DB connection successful, found settings:", existingSettings);
    
    // First let's update DB to mark as configured
    if (existingSettings.length === 0) {
      // Create new settings record
      console.log("Creating new system settings record");
      await db.insert(systemSettings).values({
        isInitialized: true,
        firstStartupDate: new Date(),
        lastStartupDate: new Date(),
        whisperModelSizes: models,
      });
    } else {
      // Update existing settings
      console.log("Updating existing system settings");
      await db.update(systemSettings)
        .set({
          isInitialized: true,
          lastStartupDate: new Date(),
          whisperModelSizes: models,
        })
        .where(eq(systemSettings.id, existingSettings[0].id));
    }
    
    // Now start model downloads
    const baseDir = join(process.cwd(), 'whisper');
    const whisperDir = join(baseDir, 'whisper.cpp');
    const diarizeDir = join(process.cwd(), 'diarize');
    const pythonCommand = isWindows ? 'python.exe' : 'python';
    
    // Create base directory
    await mkdir(baseDir, { recursive: true });
    await mkdir(modelsDir, { recursive: true });
    
    // Configure environment to bypass HuggingFace authentication for whisper models
    env.HF_HUB_DISABLE_TELEMETRY = "1";
    env.HF_HUB_ENABLE_HF_TRANSFER = "0";
    env.TRUST_REMOTE_CODE = "1";
    
    // Don't set these if we have an API key
    if (!hfApiKey) {
      env.HF_TOKEN = "";
      env.HF_HUB_OFFLINE = "0";
    }
    
    console.log("Installing dependencies...");
    try {
      await execAsync(
        `uv pip install torch torchaudio --index-url https://download.pytorch.org/whl/cpu`,
        { shell: true, maxBuffer }
      );
    } catch (err) {
      console.log("PyTorch might be already installed, continuing...");
    }
    
    console.log("Installing WhisperX...");
    try {
      await execAsync(
        `uv pip install -U whisperx huggingface_hub`,
        { shell: true, maxBuffer }
      );
    } catch (err) {
      throw new Error(`Failed to install WhisperX: ${err instanceof Error ? err.message : 'Unknown error'}`);
    }
    
    // Download selected Whisper models
    const downloadResults = [];
    if (models.length > 0) {
      for (const model of models) {
        console.log(`Downloading Whisper ${model} model...`);
        
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
          await execAsync(`echo '${downloadScript}' > ${scriptPath}`, { shell: true, maxBuffer });
          
          // Execute the script
          const result = await execAsync(`${pythonCommand} ${scriptPath}`, { shell: true, maxBuffer });
          downloadResults.push({ model, success: true, output: result.stdout });
        } catch (err) {
          console.error(`Failed to download ${model} model:`, err);
          downloadResults.push({ 
            model, 
            success: false, 
            error: err instanceof Error ? err.message : 'Unknown error' 
          });
        }
      }
    }
    
    // Download diarization model if enabled
    let diarizationResult = null;
    if (enableDiarization) {
      console.log(`Downloading diarization model (${diarizationModel})...`);
      
      if (!hfApiKey) {
        console.error("HuggingFace API key is required for diarization model download");
        diarizationResult = {
          success: false,
          error: "HuggingFace API key is required for diarization model download"
        };
      } else {
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
          await execAsync(`echo '${diarizeScript}' > ${diarizeScriptPath}`, { shell: true, maxBuffer });
          
          // Execute the script
          const result = await execAsync(`${pythonCommand} ${diarizeScriptPath}`, { shell: true, maxBuffer });
          diarizationResult = { success: true, output: result.stdout };
        } catch (err) {
          console.error("Failed to download diarization model:", err);
          diarizationResult = {
            success: false,
            error: err instanceof Error ? err.message : 'Unknown error'
          };
        }
      }
    }
    
    console.log("Installation completed!");
    
    return json({
      success: true,
      message: "System has been configured and models have been downloaded",
      modelResults: downloadResults,
      diarizationResult
    });
  } catch (error) {
    console.error("Error during direct setup:", error);
    return json({
      success: false,
      error: error instanceof Error ? error.message : "Unknown error"
    }, { status: 500 });
  }
};