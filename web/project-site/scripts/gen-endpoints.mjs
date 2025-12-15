#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';

const ROOT = path.resolve(process.cwd(), '../../');
const apiDir = path.join(ROOT, 'internal', 'api');
const outDir = path.resolve(process.cwd(), 'public', 'api');
const outFile = path.join(outDir, 'undocumented.json');

function walk(dir) {
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const e of entries) {
        const p = path.join(dir, e.name);
        if (e.isDirectory()) walk(p);
        else if (e.isFile() && e.name.endsWith('.go')) parseFile(p);
    }
}

const endpoints = [];

function parseFile(file) {
    const src = fs.readFileSync(file, 'utf8');
    const lines = src.split(/\r?\n/);
    let block = [];
    function flush() {
        const text = block.join('\n');
        const router = text.match(/@Router\s+([^\s]+)\s+\[([a-zA-Z]+)\]/);
        if (!router) return;
        const pathStr = router[1];
        const method = router[2].toUpperCase();
        const summary = (text.match(/@Summary\s+(.+)/) || [])[1] || '';
        const description = (text.match(/@Description\s+([\s\S]*?)(?:\n\/\/\s*@|$)/) || [])[1]?.trim() || '';
        const tagLine = (text.match(/@Tags\s+(.+)/) || [])[1] || '';
        const tag = tagLine.split(',')[0].trim();
        endpoints.push({ method, path: pathStr, summary, description, tag });
    }
    for (const line of lines) {
        const m = line.match(/^\/\/\s*@(.+)/);
        if (m) {
            block.push('// @' + m[1]);
            if (/@Router\b/.test(line)) {
                flush();
                block = [];
            }
        } else {
            block = [];
        }
    }
}

if (!fs.existsSync(apiDir)) {
    console.error('Cannot find internal/api directory at', apiDir);
    process.exit(1);
}

walk(apiDir);

fs.mkdirSync(outDir, { recursive: true });
fs.writeFileSync(outFile, JSON.stringify({ endpoints }, null, 2));
console.log('Wrote', outFile, 'with', endpoints.length, 'endpoints');
