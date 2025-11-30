import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { resolve } from 'path';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    react(),
    {
      name: 'docs-rewrite',
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          // Rewrite /docs/*.html to /docs-*.html for dev server
          if (req.url?.startsWith('/docs/') && req.url.endsWith('.html')) {
            const page = req.url.replace('/docs/', '').replace('.html', '');
            req.url = `/docs-${page}.html`;
          }
          next();
        });
      },
    },
  ],
  // Base path for GitHub Pages (empty for custom domain)
  base: '/',
  server: {
    port: 5175
  },
  build: {
    // Output directly to /docs for GitHub Pages
    outDir: resolve(__dirname, '../../docs'),
    emptyOutDir: true,
    rollupOptions: {
      input: {
        // Root level pages
        'index': resolve(__dirname, 'index.html'),
        'api': resolve(__dirname, 'api.html'),
        'changelog': resolve(__dirname, 'changelog.html'),

        // Docs pages - built directly in /docs/ subdirectory
        'docs/index': resolve(__dirname, 'docs-index.html'),
        'docs/intro': resolve(__dirname, 'docs-intro.html'),
        'docs/features': resolve(__dirname, 'docs-features.html'),
        'docs/installation': resolve(__dirname, 'docs-installation.html'),
        'docs/configuration': resolve(__dirname, 'docs-configuration.html'),
        'docs/usage': resolve(__dirname, 'docs-usage.html'),
        'docs/contributing': resolve(__dirname, 'docs-contributing.html'),
      }
    }
  }
});
