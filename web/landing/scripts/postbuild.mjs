#!/usr/bin/env node

import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import { readFileSync, writeFileSync, mkdirSync, renameSync, existsSync } from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
// Output is now directly in /docs (two levels up from web/landing/scripts)
const docsDir = join(__dirname, '..', '..', '..', 'docs');
const docsSubDir = join(docsDir, 'docs');

console.log('🔧 Post-build processing...');

// Create docs subdirectory if it doesn't exist
if (!existsSync(docsSubDir)) {
  mkdirSync(docsSubDir, { recursive: true });
  console.log('✓ Created docs subdirectory');
}

// Files to move and rename
const docsFiles = [
  { from: 'docs-index.html', to: 'docs/index.html' },
  { from: 'docs-intro.html', to: 'docs/intro.html' },
  { from: 'docs-installation.html', to: 'docs/installation.html' },
  { from: 'docs-diarization.html', to: 'docs/diarization.html' },
  { from: 'docs-contributing.html', to: 'docs/contributing.html' },
];

// Move and rename docs files
for (const file of docsFiles) {
  const fromPath = join(docsDir, file.from);
  const toPath = join(docsDir, file.to);

  if (existsSync(fromPath)) {
    renameSync(fromPath, toPath);
    console.log(`✓ Moved ${file.from} -> ${file.to}`);

    // Fix asset paths in docs files (they need to go up one directory)
    const content = readFileSync(toPath, 'utf8');
    const fixedContent = content.replace(/\/assets\//g, '../assets/');
    writeFileSync(toPath, fixedContent);
    console.log(`✓ Fixed asset paths in ${file.to}`);
  } else {
    console.warn(`⚠ File not found: ${file.from}`);
  }
}

console.log('🎉 Post-build processing complete!');