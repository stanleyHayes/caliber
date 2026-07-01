import { useLocation } from 'react-router-dom';

import { getAnalyticsConfig } from '../analytics/config';
import { Seo } from './Seo';

const ORG_JSON_LD = {
  '@context': 'https://schema.org',
  '@type': 'Organization',
  name: 'Project Caliber',
  description: 'Explainable, bias-safe talent intelligence for Ghana and West Africa.',
  url: 'https://projectcaliber.app',
};

type Meta = { title: string; description: string; noindex?: boolean };

// Public pages are indexable with rich descriptions; authenticated app routes are
// noindex (they are also behind ProtectedRoute and carry no public content).
const ROUTES: Record<string, Meta> = {
  '/': {
    title: 'Project Caliber',
    description:
      'Explainable, bias-safe talent intelligence: structured role specs and ranked shortlists where every score traces to evidence, AI screening interviews, and an honest candidate agent.',
  },
  '/login': { title: 'Sign in', description: 'Sign in to Project Caliber.' },
  '/register': {
    title: 'Create your account',
    description: 'Create a Project Caliber account — for employers and candidates.',
  },
  '/404': { title: 'Page not found', description: 'This page could not be found.', noindex: true },
  '/app': { title: 'Talent Radar', description: 'The Talent Radar dashboard.', noindex: true },
  '/roles': { title: 'Your roles', description: 'Manage your open roles.', noindex: true },
  '/roles/new': { title: 'New role', description: 'Create a role from a plain-language brief.', noindex: true },
  '/interview': { title: 'Screening interview', description: 'AI screening interview.', noindex: true },
  '/profile': { title: 'Your profile', description: 'Your talent passport.', noindex: true },
  '/agent': { title: 'Candidate agent', description: 'Your autonomous candidate agent.', noindex: true },
  '/radar': { title: 'Talent Radar', description: 'Talent Radar.', noindex: true },
};

const FALLBACK: Meta = {
  title: 'Project Caliber',
  description: 'Explainable, bias-safe talent intelligence.',
  noindex: true,
};

/** RouteSeo sets per-route document metadata from a central map. */
export function RouteSeo() {
  const { pathname } = useLocation();
  const meta = ROUTES[pathname] ?? FALLBACK;
  const { searchConsoleVerification } = getAnalyticsConfig();
  // Search Console verification belongs on public pages only; app routes are
  // noindex and should not claim ownership of the crawlable surface.
  const verification = !meta.noindex ? searchConsoleVerification : undefined;
  return (
    <Seo
      title={meta.title}
      description={meta.description}
      path={pathname}
      noindex={meta.noindex}
      jsonLd={pathname === '/' ? ORG_JSON_LD : undefined}
      searchConsoleVerification={verification}
    />
  );
}
