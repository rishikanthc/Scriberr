import { WebPlugin } from '@capacitor/core';
export class SpeechRecognitionWeb extends WebPlugin {
    available() {
        throw this.unimplemented('Method not implemented on web.');
    }
    async startMicrophoneStream() {
        throw this.unimplemented('Not implemented on web.');
    }
    async stopMicrophoneStream() {
        throw this.unimplemented('Not implemented on web.');
    }
    async record(_options) {
        throw this.unimplemented('Not implemented on web.');
    }
    async stopRecording() {
        throw this.unimplemented('Not implemented on web.');
    }
    start(_options) {
        throw this.unimplemented('Method not implemented on web.');
    }
    stop() {
        throw this.unimplemented('Method not implemented on web.');
    }
    getSupportedLanguages() {
        throw this.unimplemented('Method not implemented on web.');
    }
    hasPermission() {
        throw this.unimplemented('Method not implemented on web.');
    }
    isListening() {
        throw this.unimplemented('Method not implemented on web.');
    }
    requestPermission() {
        throw this.unimplemented('Method not implemented on web.');
    }
    checkPermissions() {
        throw this.unimplemented('Method not implemented on web.');
    }
    requestPermissions() {
        throw this.unimplemented('Method not implemented on web.');
    }
}
const SpeechRecognition = new SpeechRecognitionWeb();
export { SpeechRecognition };
//# sourceMappingURL=web.js.map