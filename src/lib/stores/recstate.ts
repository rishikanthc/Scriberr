// stores.ts
import { writable } from 'svelte/store';

export let isRecording = writable(false);
export let audioStream = writable<MediaStream | null>(null);
export let audioContext = writable<AudioContext | null>(null);
export let isPlaying = writable(false);
export let audioSource = writable<MediaStreamAudioSourceNode | MediaElementAudioSourceNode | null>(null);
