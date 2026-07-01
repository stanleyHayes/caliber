#!/usr/bin/env node
/**
 * Build-time prerender pipeline for Project Caliber public pages (CAL-121).
 *
 * The SPA is rendered for each public route using the same React components as
 * the client build. The resulting markup is injected into dist/<route>/index.html
 * so crawlers see real content without running JavaScript.
 *
 * Usage:
 *   node scripts/prerender.js
 *
 * This is intended to run after `vite build` has produced dist/index.html and
 * the client assets.
 */

import { execSync } from 'node:child_process';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { JSDOM } from 'jsdom';

const __dirname = dirname(fileURLToPath(import.meta.url));
const projectRoot = join(__dirname, '..');
const distDir = join(projectRoot, 'dist');
const serverDir = join(distDir, 'server');
const serverEntry = join(serverDir, 'entry-server.js');

// Public routes that should ship as crawlable HTML. The authenticated app shell
// is intentionally omitted — it stays CSR.
const PUBLIC_ROUTES = [
  { path: '/', title: 'Project Caliber', snippet: 'Hire on evidence' },
  { path: '/login', title: 'Sign in', snippet: 'Welcome back' },
  { path: '/register', title: 'Create your account', snippet: 'Passwords must be at least 12 characters' },
  { path: '/404', title: 'Page not found', snippet: 'Not found' },
];

function setupDomEnvironment() {
  const { window } = new JSDOM(
    '<!DOCTYPE html><html lang="en"><head></head><body></body></html>',
    { url: 'https://projectcaliber.app/', pretendToBeVisual: true },
  );

  // JSDOM supplies localStorage, document and window; polyfill the few APIs that
  // Motion/MUI may touch during module load or render.
  globalThis.window = window;
  globalThis.document = window.document;
  globalThis.localStorage = window.localStorage;
  Object.defineProperty(globalThis, 'navigator', {
    value: window.navigator,
    configurable: true,
  });
  window.requestAnimationFrame = window.requestAnimationFrame || ((cb) => setTimeout(cb, 0));
  window.cancelAnimationFrame = window.cancelAnimationFrame || ((id) => clearTimeout(id));
  window.ResizeObserver = window.ResizeObserver || class ResizeObserver { observe() {} unobserve() {} disconnect() {} };
  window.IntersectionObserver = window.IntersectionObserver || class IntersectionObserver { observe() {} unobserve() {} disconnect() {} };
}

function buildSsrBundle() {
  console.log('[prerender] building SSR bundle...');
  execSync(
    'npx vite build --ssr src/entry-server.tsx --outDir dist/server',
    { cwd: projectRoot, stdio: 'inherit' },
  );
}

function loadTemplate() {
  const templatePath = join(distDir, 'index.html');
  if (!existsSync(templatePath)) {
    throw new Error(`Missing ${templatePath}. Run vite build first.`);
  }
  return readFileSync(templatePath, 'utf8');
}

const HEAD_TAG_RE = /^(?:<title>[^]*?<\/title>|<meta\s[^>]*>|<link\s[^>]*>|<script\s+type="application\/ld\+json">[^]*?<\/script>)/i;

function splitRenderedHtml(html) {
  const headTags = [];
  let remaining = html;
  while (true) {
    const match = remaining.match(HEAD_TAG_RE);
    if (!match) break;
    headTags.push(match[0]);
    remaining = remaining.slice(match[0].length);
  }
  return { headTags: headTags.join(''), bodyHtml: remaining };
}

function cleanHead(template) {
  // Remove generic SEO tags that the route-specific head tags will replace.
  return template
    .replace(/<title>[^]*?<\/title>/i, '')
    .replace(/<meta\s+name="description"\s+content="[^"]*"\s*\/?>/i, '')
    .replace(/<meta\s+property="og:title"\s+content="[^"]*"\s*\/?>/gi, '')
    .replace(/<meta\s+property="og:description"\s+content="[^"]*"\s*\/?>/gi, '')
    .replace(/<meta\s+property="og:url"\s+content="[^"]*"\s*\/?>/gi, '')
    .replace(/<meta\s+name="twitter:title"\s+content="[^"]*"\s*\/?>/gi, '')
    .replace(/<meta\s+name="twitter:description"\s+content="[^"]*"\s*\/?>/gi, '')
    .replace(/<link\s+rel="canonical"\s+href="[^"]*"\s*\/?>/gi, '');
}

function inject(template, headTags, bodyHtml) {
  let html = cleanHead(template);
  html = html.replace(/<\/head>/i, `${headTags}</head>`);
  html = html.replace(/<div\s+id="root"\s*>\s*<\/div>/i, `<div id="root">${bodyHtml}</div>`);
  return html;
}

function writePage(routePath, html) {
  const outDir = routePath === '/' ? distDir : join(distDir, routePath);
  mkdirSync(outDir, { recursive: true });
  writeFileSync(join(outDir, 'index.html'), html);
}

async function prerender() {
  setupDomEnvironment();
  buildSsrBundle();

  const { render } = await import(serverEntry);
  const template = loadTemplate();

  console.log('[prerender] rendering public routes...');
  for (const route of PUBLIC_ROUTES) {
    const { html } = render(route.path);
    const { headTags, bodyHtml } = splitRenderedHtml(html);
    const page = inject(template, headTags, bodyHtml);
    writePage(route.path, page);
    console.log(`[prerender]  ${route.path} -> ${route.path === '/' ? 'dist/index.html' : `dist${route.path}/index.html`}`);
  }

  console.log('[prerender] done.');
}

prerender().catch((err) => {
  console.error('[prerender] failed:', err);
  process.exit(1);
});
