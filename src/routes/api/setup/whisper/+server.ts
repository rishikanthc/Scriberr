import { exec } from 'child_process';
import { db } from '$lib/server/db';
import { systemSettings } from '$lib/server/db/schema';
import { eq } from 'drizzle-orm';
import { promisify } from 'util';
import { mkdir } from 'fs/promises';
import { join } from 'path';
import type { RequestHandler } from './$types';
import { platform } from 'os';

const execAsync = promisify(exec);
const isWindows = platform() === 'win32';
const maxBuffer = 1024 * 1024 * 100; // 100MB buffer

export const GET: RequestHandler = async ({ url }) => {
   const models = JSON.parse(url.searchParams.get('models') || '[]');
   const multilingual = url.searchParams.get('multilingual') === 'true';

   
   
   const stream = new ReadableStream({
       async start(controller) {
           try {
               const baseDir = join(process.cwd(), 'whisper');
               const whisperDir = join(baseDir, 'whisper.cpp');
               const diarizeDir = join(process.cwd(), 'diarize');

               console.log("SETUP LOGS --->", baseDir, whisperDir, diarizeDir)
               
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
               const gitCommand = isWindows ? 'git.exe' : 'git';
               const makeCommand = isWindows ? 'mingw32-make' : 'make';
               const shellScript = isWindows ? 'bash.exe' : 'bash';
               const pipCommand = isWindows ? 'pip.exe' : 'pip';

               // sendMessage('Cloning whisper.cpp repository...', 0);
               // await execWithProgress(`${gitCommand} clone https://github.com/ggerganov/whisper.cpp.git`, { 
               //     cwd: baseDir,
               //     shell: true
               // });

               // console.log("CLONED SUCC -->")

               sendMessage('Installing dependencies...', 0);
               try {
                   await execWithProgress(
                       `python3 -m pip install torch==2.0.0 torchaudio==2.0.0 --index-url https://download.pytorch.org/whl/cpu`, 
                       { shell: true }
                   );
               } catch (err) {
                   throw new Error(`Failed to install dependencies: ${err instanceof Error ? err.message : 'Unknown error'}`);
               }

               let currentProgress = 10;

               
               sendMessage('Installing WhisperX...', 30);
               try {
                   await execWithProgress(
                       `python3 -m pip install whisperx`, 
                       { shell: true }
                   );
               } catch (err) {
                   throw new Error(`Failed to install WhisperX: ${err instanceof Error ? err.message : 'Unknown error'}`);
               }

               currentProgress += 30;

               // const progressPerStep = 80 / models.length;
               // for (const model of models) {
               //     sendMessage(`Downloading ${model} model...`, currentProgress);
               //     await execWithProgress(`${shellScript} ./models/download-ggml-model.sh ${model}`, { 
               //         cwd: whisperDir,
               //         shell: true
               //     });
               //     currentProgress += progressPerStep;
               // }

               // sendMessage('Compiling whisper.cpp...', 90);
               // await execWithProgress(makeCommand, { 
               //     cwd: whisperDir,
               //     shell: true
               // });

               // sendMessage('Testing whisper installation...', 95);
               // await execWithProgress(`make base.en`, { 
               //     cwd: whisperDir,
               //     shell: true
               // });

               sendMessage('Installation completed successfully!', 99);

               sendMessage('Saving configuration settings!', 100, 'complete');

              await db.update(systemSettings).set({
                isInitialized: true,
                firstStartupDate: new Date(),
                lastStartupDate: new Date(),
                whisperModelSizes: models,
              }).where(eq(systemSettings.isInitialized, false));
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
