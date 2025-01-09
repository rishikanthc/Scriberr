import type { PermissionState, PluginListenerHandle } from '@capacitor/core';
export interface PermissionStatus {
    /**
     * Permission state for speechRecognition alias.
     *
     * On Android it requests/checks RECORD_AUDIO permission
     *
     * On iOS it requests/checks the speech recognition and microphone permissions.
     *
     * @since 5.0.0
     */
    speechRecognition: PermissionState;
}
export interface RecordOptions {
    fileName?: string;
    outputPath?: string;
}
export interface RecordingResult {
    /**
     * Path to the recorded audio file
     */
    path: string;
    status?: 'started' | 'stopped';
}
export interface AudioData {
    buffer: number[];
}
export interface SpeechRecognitionPlugin {
    /**
     * This method will check if speech recognition feature is available on the device.
     * @param none
     * @returns available - boolean true/false for availability
     */
    available(): Promise<{
        available: boolean;
    }>;
    /**
     * Start recording audio to a file
     * @returns Promise with the path to the recorded file
     */
    record(options?: RecordOptions): Promise<RecordingResult>;
    /**
       * Start streaming microphone data for visualization
       */
    startMicrophoneStream(): Promise<RecordingResult>;
    /**
     * Stop streaming microphone data
     */
    stopMicrophoneStream(): Promise<RecordingResult>; /**
     * Stop the current recording
     * @returns Promise that resolves when recording is stopped
     */
    stopRecording(): Promise<void>;
    /**
     * This method will start to listen for utterance.
     *
     * if `partialResults` is `true`, the function respond directly without result and
     * event `partialResults` will be emit for each partial result, until stopped.
     *
     * @param options
     * @returns void or array of string results
     */
    start(options?: UtteranceOptions): Promise<{
        matches?: string[];
    }>;
    /**
     * This method will stop listening for utterance
     * @param none
     * @returns void
     */
    stop(): Promise<void>;
    /**
     * This method will return list of languages supported by the speech recognizer.
     *
     * It's not available on Android 13 and newer.
     *
     * @param none
     * @returns languages - array string of languages
     */
    getSupportedLanguages(): Promise<{
        languages: any[];
    }>;
    /**
     * This method will check if speech recognition is listening.
     * @param none
     * @returns boolean true/false if speech recognition is currently listening
     *
     * @since 5.1.0
     */
    isListening(): Promise<{
        listening: boolean;
    }>;
    /**
     * Check the speech recognition permission.
     *
     * @since 5.0.0
     */
    checkPermissions(): Promise<PermissionStatus>;
    /**
     * Request the speech recognition permission.
     *
     * @since 5.0.0
     */
    requestPermissions(): Promise<PermissionStatus>;
    /**
     * Called when partialResults set to true and result received.
     *
     * On Android it doesn't work if popup is true.
     *
     * Provides partial result.
     *
     * @since 2.0.2
     */
    addListener(eventName: 'partialResults', listenerFunc: (data: {
        matches: string[];
    }) => void): Promise<PluginListenerHandle>;
    /**
     * Called when listening state changed.
     *
     * @since 5.1.0
     */
    addListener(eventName: 'listeningState', listenerFunc: (data: {
        status: 'started' | 'stopped';
    }) => void): Promise<PluginListenerHandle>;
    /**
     * Remove all the listeners that are attached to this plugin.
     *
     * @since 4.0.0
     */
    removeAllListeners(): Promise<void>;
}
export interface UtteranceOptions {
    /**
     * key returned from `getSupportedLanguages()`
     */
    language?: string;
    /**
     * maximum number of results to return (5 is max)
     */
    maxResults?: number;
    /**
     * prompt message to display on popup (Android only)
     */
    prompt?: string;
    /**
     * display popup window when listening for utterance (Android only)
     */
    popup?: boolean;
    /**
     * return partial results if found
     */
    partialResults?: boolean;
}
