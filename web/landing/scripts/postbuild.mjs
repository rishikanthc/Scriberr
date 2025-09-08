#!/usr/bin/env node

import { fileURLToPath } from 'url';
import { dirname, join } from 'path';
import { readFileSync, writeFileSync, mkdirSync, renameSync, existsSync } from 'fs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const distDir = join(__dirname, '..', 'dist');
const docsDir = join(distDir, 'docs');

console.log('🔧 Post-build processing...');

// Create docs directory
if (!existsSync(docsDir)) {
  mkdirSync(docsDir, { recursive: true });
  console.log('✓ Created docs directory');
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
  const fromPath = join(distDir, file.from);
  const toPath = join(distDir, file.to);
  
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