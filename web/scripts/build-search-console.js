#!/usr/bin/env node
/**
 * Build-time Search Console verification file generator (CAL-128).
 *
 * Google Search Console supports domain verification via a DNS record, an HTML
 * meta tag (handled in Seo.tsx/RouteSeo.tsx via VITE_SEARCH_CONSOLE_VERIFICATION),
 * or an HTML file placed at the site root (e.g. /google123abc.html).
 *
 * This script generates that file in the build output when
 * VITE_SEARCH_CONSOLE_HTML_FILE_TOKEN is set, so the token never needs to be
 * committed to source control.
 */

import { existsSync, mkdirSync, readdirSync, unlinkSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const projectRoot = join(__dirname, '..');
const distDir = join(projectRoot, 'dist');

const token = (process.env.VITE_SEARCH_CONSOLE_HTML_FILE_TOKEN || '').trim();

function cleanGeneratedFiles() {
  if (!existsSync(distDir)) {
    return;
  }
  for (const file of readdirSync(distDir)) {
    if (file.startsWith('google') && file.endsWith('.html')) {
      unlinkSync(join(distDir, file));
    }
  }
}

function writeVerificationFile() {
  if (!token) {
    console.log('[search-console] no VITE_SEARCH_CONSOLE_HTML_FILE_TOKEN set; skipping file verification.');
    return;
  }

  mkdirSync(distDir, { recursive: true });
  cleanGeneratedFiles();

  const fileName = `google${token}.html`;
  const filePath = join(distDir, fileName);
  // The exact response Google expects is the token as the only body content.
  writeFileSync(filePath, `google-site-verification: google${token}.html\n`, 'utf8');
  console.log(`[search-console] wrote ${filePath}`);
}

writeVerificationFile();
