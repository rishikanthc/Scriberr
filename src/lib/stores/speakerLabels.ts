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
        const response = await apiFetch(`/api/audio/${fileId}/transcript`, {
          method: 'GET'
        });
        if (!response.ok) throw new Error('Failed to load transcript');
        
        const data = await response.json();
        if (data?.speakerLabels) {
          this.updateLabels(fileId, data.speakerLabels);
        }
        return data;
      } catch (error) {
        console.error('Error loading transcript:', error);
        return {}; // Return empty object on error to avoid null/undefined errors
      }
    },
    async saveLabels(fileId: number, transcript: TranscriptSegment[], labels: Record<string, string>) {
      try {
        const response = await apiFetch(`/api/audio/${fileId}/transcript`, {
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