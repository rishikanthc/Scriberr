// stores/speakerLabelStore.ts
import { writable } from 'svelte/store';
import { apiFetch } from '$lib/api';
import type { TranscriptSegment } from '$lib/types';

function createSpeakerLabelStore() {
  const { subscribe, set, update } = writable<Record<number, Record<string, string>>>({});

  return {
    subscribe,
    updateLabels: (fileId: number, labels: Record<string, string>) => {
      update(state => ({ ...state, [fileId]: labels }));
    },
    async loadLabels(fileId: number) {
      try {
        const response = await apiFetch(`/api/audio-files/${fileId}/transcript`);
        if (!response.ok) {
          console.error('Failed to fetch transcript data');
          return {};
        }
        
        const data = await response.json();
        if (data.speakerLabels) {
          this.updateLabels(fileId, data.speakerLabels);
          return data.speakerLabels;
        }
        return {};
      } catch (error) {
        console.error('Error loading speaker labels:', error);
        return {};
      }
    },
    async saveLabels(fileId: number, transcript: TranscriptSegment[], labels: Record<string, string>) {
      try {
        const response = await apiFetch(`/api/audio-files/${fileId}/transcript`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ transcript, speakerLabels: labels })
        });
        if (!response.ok) throw new Error('Failed to update transcript');
        
        this.updateLabels(fileId, labels);
        return await response.json();
      } catch (error) {
        console.error('Error updating transcript:', error);
        throw error;
      }
    }
  };
}

export const speakerLabels = createSpeakerLabelStore();