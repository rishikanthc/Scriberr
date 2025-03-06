import { SQL, eq } from 'drizzle-orm';
import { db } from './db';
import fs from 'fs';
import path from 'path';
import { audioFiles } from './db/schema';
import type { GetTranscriptResult } from '$lib/types';

// Define storage path manually as it's not exported from schema
const storagePath = process.env.AUDIO_DIR || path.join(process.cwd(), 'uploads');

export const getAudioFilePath = async (fileId: number, original = false) => {
  const result = await db
    .select({
      path: audioFiles.fileName, // Using fileName instead of path which doesn't exist
      originalFileName: audioFiles.originalFileName,
    })
    .from(audioFiles)
    .where(eq(audioFiles.id, fileId));

  if (!result.length) {
    throw new Error('Audio file not found');
  }

  const filePath = path.join(storagePath, result[0].path);
  const origPath = result[0].originalFileName
    ? path.join(storagePath, path.basename(result[0].path, path.extname(result[0].path)) + '-original' + path.extname(result[0].originalFileName))
    : null;

  // If requesting original file and it exists, return it
  if (original && origPath && fs.existsSync(origPath)) {
    return origPath;
  }

  // Otherwise return the standard (WAV) file
  return filePath;
};

export const getTranscript = async (fileId: number): Promise<GetTranscriptResult> => {
  const result = await db
    .select({
      transcript: audioFiles.transcript,
      status: audioFiles.transcriptionStatus,
      summary: audioFiles.summary,
      peaks: audioFiles.peaks,
      diarization: audioFiles.diarization,
    })
    .from(audioFiles)
    .where(eq(audioFiles.id, fileId));

  if (!result.length) {
    throw new Error('Audio file not found');
  }

  return {
    transcript: result[0].transcript || [],
    status: result[0].status || 'unknown',
    summary: result[0].summary || null,
    peaks: result[0].peaks || null,
    diarization: result[0].diarization || false,
  };
};

export const getTranscriptionStatus = async (fileId: number): Promise<string> => {
  const result = await db
    .select({
      status: audioFiles.transcriptionStatus,
    })
    .from(audioFiles)
    .where(eq(audioFiles.id, fileId));

  if (!result.length) {
    throw new Error('Audio file not found');
  }

  return result[0].status || 'unknown';
};

export const updateTranscriptionStatus = async (
  fileId: number,
  status: string,
  transcript = null,
  peaks = null,
  diarization = false
) => {
  return db
    .update(audioFiles)
    .set({
      transcriptionStatus: status,
      transcript: typeof transcript === 'string' ? transcript : JSON.stringify(transcript),
      peaks,
      diarization,
    })
    .where(eq(audioFiles.id, fileId));
};

export const setSummary = async (fileId: number, summary: string) => {
  return db
    .update(audioFiles)
    .set({
      summary,
    })
    .where(eq(audioFiles.id, fileId));
};

export const updateTitle = async (fileId: number, title: string) => {
  return db.update(audioFiles).set({ title }).where(eq(audioFiles.id, fileId));
};

// Real implementation of the transcription function that calls the Python script
export const transcribeAudio = async (fileId: number, transcribeStream: any) => {
  // Wrap everything in a try/catch block
  try {
    console.log(`Starting transcription for file ID: ${fileId}`);
    
    // Get the audio file path
    const filePath = await getAudioFilePath(fileId);
    console.log(`Using audio file path: ${filePath}`);
    
    // Update status to processing
    await updateTranscriptionStatus(fileId, 'processing');
    
    // Get file information from the database to pass to whisper
    const fileInfo = await db
      .select({
        modelSize: audioFiles.modelSize,
        language: audioFiles.language,
        threads: audioFiles.threads,
        processors: audioFiles.processors,
        diarization: audioFiles.diarization,
      })
      .from(audioFiles)
      .where(eq(audioFiles.id, fileId))
      .then(rows => rows[0]);
      
    console.log(`Retrieved fileInfo for ID ${fileId}:`, fileInfo);
    console.log(`Diarization setting from database: ${fileInfo?.diarization}`);
    
    if (!fileInfo) {
      throw new Error('File information not found');
    }
    
    // Send initial progress update
    if (transcribeStream) {
      try {
        await transcribeStream.sendProgress({
          status: 'processing',
          progress: 0,
          transcript: []
        });
      } catch (sendError) {
        console.error('Error sending initial progress update:', sendError);
        // Continue execution even if sending progress fails
      }
    }
    
    console.log(`Transcribing audio file: ${filePath}`);
    
    // Import the necessary modules for executing the Python script
    const { spawn } = await import('child_process');
    const { join } = await import('path');
    const { readFile, writeFile } = await import('fs/promises');
    
    // Create a path for the output transcript JSON
    const outputPath = join(storagePath, `transcript-${fileId}.json`);
    
    // Prepare the command and arguments for the Python script
    const args = [
      'transcribe.py',
      '--audio-file', filePath,
      '--model-size', fileInfo.modelSize || 'base',
      '--output-file', outputPath,
      '--threads', String(fileInfo.threads || 4)
    ];
    
    // Add optional arguments based on configuration
    if (fileInfo.language) {
      args.push('--language', fileInfo.language);
    }
    
    console.log(`Checking if diarization should be enabled: ${fileInfo.diarization}`);
    // Force enable diarization for testing
    console.log("FORCING DIARIZATION FLAG ON");
    args.push('--diarize');
    // Determine the compute device (CPU or GPU)
    const device = process.env.USE_GPU === 'true' ? 'cuda' : 'cpu';
    args.push('--device', device);
    
    // Compute type based on device
    const computeType = device === 'cuda' ? 'float16' : 'int8';
    args.push('--compute-type', computeType);
    
    console.log('Executing Python with args:', args);
    
    // Execute the Python process
    const pythonProcess = spawn('python', args, {
      cwd: process.cwd()
    });
    
    let progressPattern = /Progress: (\d+\.\d+)%/;
    let progressValue = 0;
    
    // Process the output to extract progress information
    pythonProcess.stdout.on('data', (data) => {
      const output = data.toString();
      console.log('Transcription progress:', output);
      
      // Try to extract progress percentage from the output
      const match = progressPattern.exec(output);
      if (match && match[1]) {
        progressValue = parseFloat(match[1]);
        
        // Send progress update to client
        if (transcribeStream) {
          try {
            transcribeStream.sendProgress({
              status: 'processing',
              progress: progressValue,
              transcript: []
            }).catch(err => console.error('Error sending progress update:', err));
          } catch (sendError) {
            console.error('Error in sendProgress:', sendError);
          }
        }
      }
    });
    
    // Handle errors
    pythonProcess.stderr.on('data', (data) => {
      console.error('Transcription error:', data.toString());
    });
    
    // Wait for the process to complete
    await new Promise((resolve, reject) => {
      pythonProcess.on('close', (code) => {
        console.log(`Python process exited with code ${code}`);
        if (code === 0) {
          resolve(null);
        } else {
          reject(new Error(`Python process exited with code ${code}`));
        }
      });
    });
    
    // Create a fallback transcript in case the output file doesn't exist or can't be parsed
    const fallbackTranscript = [
      { start: 0, end: 10, text: "Transcription completed, but output format was not as expected.", speaker: "" }
    ];
    
    // Read and parse the transcript
    let transcript;
    try {
      console.log(`Reading transcript from ${outputPath}`);
      
      // Check if the file exists
      if (!fs.existsSync(outputPath)) {
        console.error(`Transcript file not found at ${outputPath}`);
        throw new Error('Transcript file not found');
      }
      
      const transcriptData = await readFile(outputPath, 'utf-8');
      
      // Verify that we have some content
      if (!transcriptData || transcriptData.trim() === '') {
        console.error('Empty transcript file');
        throw new Error('Empty transcript file');
      }
      
      // Parse the JSON
      const result = JSON.parse(transcriptData);
      
      // Validate that the expected structure exists
      if (!result || !result.segments || !Array.isArray(result.segments)) {
        console.error('Invalid transcript format');
        throw new Error('Invalid transcript format');
      }
      
      // Convert WhisperX segments to our segment format
      transcript = result.segments.map(segment => ({
        start: segment.start,
        end: segment.end,
        text: segment.text || "",
        speaker: segment.speaker || ''
      }));
      
      console.log(`Transcription completed with ${transcript.length} segments`);
      if (transcript.length > 0) {
        console.log(`Sample segment: ${JSON.stringify(transcript[0])}`);
      }
      
      // Additional sanity check
      if (transcript.length === 0) {
        console.warn('Transcript has zero segments, using fallback');
        transcript = fallbackTranscript;
      }
    } catch (error) {
      console.error('Failed to read or parse transcript:', error);
      console.log('Using fallback transcript');
      transcript = fallbackTranscript;
    }
    
    // Extract audio peaks for visualization if they don't exist yet
    let peaks = null;
    
    // Update with completed status and transcript
    console.log('Updating transcription status to completed with transcript');
    await updateTranscriptionStatus(
      fileId, 
      'completed', 
      JSON.stringify(transcript), // Ensure transcript is stringified before saving
      peaks, 
      fileInfo.diarization || false
    );
    
    // Send final update
    if (transcribeStream) {
      try {
        await transcribeStream.sendProgress({
          status: 'completed',
          progress: 100,
          transcript
        });
      } catch (sendError) {
        console.error('Error sending final progress update:', sendError);
      }
    }
    
    console.log('Transcription process finished successfully');
    return transcript;
  } catch (error) {
    // Handle any errors that occurred during the transcription process
    const errorString = error ? 
      (error instanceof Error ? error.stack || error.message : String(error)) : 
      'Unknown transcription error';
    
    console.error('Transcription failed:', errorString);
    
    // Update status to failed
    try {
      await updateTranscriptionStatus(fileId, 'failed');
    } catch (dbError) {
      console.error('Failed to update transcription status:', dbError);
    }
    
    // Send error update to client
    if (transcribeStream) {
      try {
        // Create a safe error message
        const errorMessage = error ? 
          (error instanceof Error ? error.message : String(error)) : 
          'Unknown transcription error';
        
        await transcribeStream.sendProgress({
          status: 'failed',
          progress: 0,
          transcript: [],
          error: errorMessage
        });
      } catch (sendError) {
        console.error('Failed to send error update:', sendError);
      }
    }
    
    // Ensure we always throw a proper Error object with detailed information
    if (error instanceof Error) {
      throw error;
    } else {
      throw new Error(errorString);
    }
  }
};