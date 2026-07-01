#!/usr/bin/env node
/**
 * Verifies that the build-time prerender pipeline produced crawlable HTML for
 * every public route.
 */

import { existsSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const distDir = join(__dirname, '..', 'dist');

const CHECKS = [
  { path: '/', file: 'index.html', title: 'Project Caliber', snippet: 'Hire on evidence' },
  { path: '/login', file: 'login/index.html', title: 'Sign in', snippet: 'Welcome back' },
  { path: '/register', file: 'register/index.html', title: 'Create your account', snippet: 'Passwords must be at least 12 characters' },
  { path: '/404', file: '404/index.html', title: 'Page not found', snippet: 'Not found' },
];

let failed = false;

for (const check of CHECKS) {
  const filePath = join(distDir, check.file);
  if (!existsSync(filePath)) {
    console.error(`[verify-prerender] missing ${filePath}`);
    failed = true;
    continue;
  }

  const html = readFileSync(filePath, 'utf8');
  const errors = [];

  if (!html.includes('<div id="root">') || html.match(/<div\s+id="root"\s*>\s*<\/div>/)) {
    errors.push('root container is empty');
  }

  const titleMatch = html.match(/<title>([^<]+)<\/title>/);
  if (!titleMatch || !titleMatch[1].includes(check.title)) {
    errors.push(`expected title containing "${check.title}"`);
  }

  if (!html.includes(check.snippet)) {
    errors.push(`expected snippet "${check.snippet}"`);
  }

  if (errors.length) {
    console.error(`[verify-prerender] ${check.path} FAILED: ${errors.join('; ')}`);
    failed = true;
  } else {
    console.log(`[verify-prerender] ${check.path} OK`);
  }
}

if (failed) {
  process.exit(1);
}
console.log('[verify-prerender] all public routes prerendered successfully.');
