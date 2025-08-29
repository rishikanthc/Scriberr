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
        'index': resolve(__dirname, 'index.html'),
        'api': resolve(__dirname, 'api.html'),
        'changelog': resolve(__dirname, 'changelog.html'),
        'docs/intro': resolve(__dirname, 'docs-intro.html'),
        'docs/installation': resolve(__dirname, 'docs-installation.html'),
        'docs/diarization': resolve(__dirname, 'docs-diarization.html'),
        'docs/contributing': resolve(__dirname, 'docs-contributing.html'),
      }
    }
  }
});

