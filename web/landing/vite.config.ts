import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { resolve } from 'path';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5175
  },
  build: {
    rollupOptions: {
      input: {
        main: resolve(__dirname, 'index.html'),
        api: resolve(__dirname, 'api.html'),
        changelog: resolve(__dirname, 'changelog.html'),
        intro: resolve(__dirname, 'docs/intro.html'),
        installation: resolve(__dirname, 'docs/installation.html'),
        diarization: resolve(__dirname, 'docs/diarization.html'),
        contributing: resolve(__dirname, 'docs/contributing.html'),
      }
    }
  }
});

