import { registerPlugin } from '@capacitor/core';

import type { SpeechRecognitionPlugin } from './definitions';

const SpeechRecognition = registerPlugin<SpeechRecognitionPlugin>(
  'SpeechRecognition',
  {
    web: () => import('./web').then(m => new m.SpeechRecognitionWeb()),
  },
);

export * from './definitions';
export { SpeechRecognition };
