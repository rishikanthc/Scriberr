import { WebPlugin } from '@capacitor/core';
import type { PermissionStatus, SpeechRecognitionPlugin, UtteranceOptions, RecordingResult, RecordOptions } from './definitions';
export declare class SpeechRecognitionWeb extends WebPlugin implements SpeechRecognitionPlugin {
    available(): Promise<{
        available: boolean;
    }>;
    startMicrophoneStream(): Promise<RecordingResult>;
    stopMicrophoneStream(): Promise<RecordingResult>;
    record(_options?: RecordOptions): Promise<RecordingResult>;
    stopRecording(): Promise<void>;
    start(_options?: UtteranceOptions): Promise<{
        matches?: string[];
    }>;
    stop(): Promise<void>;
    getSupportedLanguages(): Promise<{
        languages: any[];
    }>;
    hasPermission(): Promise<{
        permission: boolean;
    }>;
    isListening(): Promise<{
        listening: boolean;
    }>;
    requestPermission(): Promise<void>;
    checkPermissions(): Promise<PermissionStatus>;
    requestPermissions(): Promise<PermissionStatus>;
}
declare const SpeechRecognition: SpeechRecognitionWeb;
export { SpeechRecognition };
