import { registerPlugin } from '@capacitor/core';
const SpeechRecognition = registerPlugin('SpeechRecognition', {
    web: () => import('./web').then(m => new m.SpeechRecognitionWeb()),
});
export * from './definitions';
export { SpeechRecognition };
//# sourceMappingURL=index.js.map