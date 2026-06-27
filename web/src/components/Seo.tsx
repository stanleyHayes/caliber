// Project's canonical production origin (no trailing slash). The POC has no live
// domain yet; this documents intent and is overridden by a real host at deploy.
const SITE_URL = 'https://projectcaliber.app';
const SITE_NAME = 'Project Caliber';

export type SeoProps = {
  title: string;
  description: string;
  /** Canonical path for this view, e.g. "/login". */
  path?: string;
  /** Keep private/app routes out of the index. */
  noindex?: boolean;
  /** Optional JSON-LD structured data (e.g. Organization on the landing page). */
  jsonLd?: Record<string, unknown>;
};

/**
 * Seo emits per-route document metadata. On React 19 these tags are hoisted into
 * <head> automatically, so each page gets its own title, description, canonical,
 * and Open Graph / Twitter cards for crawlers and rich link previews.
 */
export function Seo({ title, description, path = '/', noindex = false, jsonLd }: SeoProps) {
  const fullTitle = title === SITE_NAME ? title : `${title} · ${SITE_NAME}`;
  const url = `${SITE_URL}${path}`;
  return (
    <>
      <title>{fullTitle}</title>
      <meta name="description" content={description} />
      <link rel="canonical" href={url} />
      {noindex ? <meta name="robots" content="noindex, nofollow" /> : null}
      <meta property="og:type" content="website" />
      <meta property="og:site_name" content={SITE_NAME} />
      <meta property="og:title" content={fullTitle} />
      <meta property="og:description" content={description} />
      <meta property="og:url" content={url} />
      <meta name="twitter:card" content="summary_large_image" />
      <meta name="twitter:title" content={fullTitle} />
      <meta name="twitter:description" content={description} />
      {jsonLd ? (
        <script
          type="application/ld+json"
          // JSON-LD requires injecting a serialized, app-controlled object.
          dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
        />
      ) : null}
    </>
  );
}
