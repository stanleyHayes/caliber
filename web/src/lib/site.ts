// Canonical production origin (no trailing slash), used for SEO canonical URLs,
// OG/JSON-LD tags, and the prerender base. Configurable per deploy via the
// build-time VITE_SITE_URL; defaults to the current Vercel deployment URL.
export const SITE_URL = (import.meta.env.VITE_SITE_URL ?? 'https://calibergh.vercel.app').replace(/\/$/, '');
