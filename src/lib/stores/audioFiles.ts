// stores/audioFileStore.ts
import { writable, derived } from 'svelte/store';
import { apiFetch } from '$lib/api';
import type { TranscriptSegment } from '$lib/types';

interface AudioFile {
  id: number;
  fileName: string;
  title?: string;
  duration: number | null;
  peaks: number[];
  transcriptionStatus: 'pending' | 'processing' | 'completed' | 'failed';
  language: string;
  uploadedAt: string;
  summary: string;
  transcribedAt: string | null;
  transcript: TranscriptSegment[] | null;
  diarization: boolean;
  lastError?: string;
}

function createAudioFileStore() {
  const { subscribe, set, update } = writable<AudioFile[]>([]);
  
  const store = {
    subscribe,
    setFiles: (files: AudioFile[]) => {
      set(files.map(file => ({ ...file, diarization: Boolean(file.diarization) })));
    },
    addFile: (file: AudioFile) => {
      update(files => [{ ...file, diarization: Boolean(file.diarization) }, ...files]);
    },
    deleteFile: async (id: number) => {
      try {
        const response = await apiFetch(`/api/audio/${id}`, {
          method: 'DELETE'
        });

        if (!response.ok) {
          const error = await response.text();
          throw new Error(`Failed to delete file: ${error}`);
        }

        // Update the store immediately after successful deletion
        update(files => files.filter(f => f.id !== id));
        
        // No need to call refresh() here as we've already updated the store
        return true;
      } catch (error) {
        console.error('Failed to delete file:', error);
        throw error;
      }
    },
    updateFile: async (id: number, data: Partial<AudioFile>) => {
      try {
        // First update the API
        const response = await apiFetch(`/api/audio/${id}`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            const error = await response.text();
            throw new Error(`Failed to update file: ${error}`);
        }
        
        const updatedFile = await response.json();
        
        // Then update the store with the response from the API
        update(files => 
            files.map(file => 
                file.id === id 
                    ? { ...file, ...updatedFile, diarization: Boolean(updatedFile.diarization) }
                    : file
            )
        );
        
        return updatedFile;
      } catch (error) {
          console.error('Failed to update file:', error);
          throw error;
      }
    },
    async refresh() {
      try {
        const response = await apiFetch('/api/audio-files');
        const files = await response.json();
        const processedFiles = await Promise.all(
          files.map(async (file: AudioFile) => {
            if (file.transcriptionStatus === 'completed' && !file.transcript) {
              const transcriptResponse = await apiFetch(`/api/transcription/${file.id}`);
              const data = await transcriptResponse.json();
              return { ...file, transcript: data.transcript, diarization: Boolean(file.diarization) };
            }
            return { ...file, diarization: Boolean(file.diarization) };
          })
        );
        set(processedFiles.sort((a, b) => 
          new Date(b.uploadedAt).getTime() - new Date(a.uploadedAt).getTime()
        ));
      } catch (error) {
        console.error('Error fetching audio files:', error);
        throw error;
      }
    }
  };

  return store;
}

export const audioFiles = createAudioFileStore();
