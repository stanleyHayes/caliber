import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { ShortlistResponse } from '../../api/types';
import { ShortlistSection } from './ShortlistSection';

const shortlist = vi.fn();
vi.mock('../../api/flow', () => ({ flowApi: { shortlist: (...args: unknown[]) => shortlist(...args) } }));

function renderWithClient(node: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}>{node}</QueryClientProvider>);
}

const response: ShortlistResponse = {
  shortlist: {
    poolDepth: 5,
    matches: [
      {
        id: 'm1',
        roleId: 'r1',
        candidateId: 'cand-0123456789',
        overallScore: 0.87,
        confidence: 'CONFIDENCE_HIGH',
        breakdown: [{ competency: 'Go', score: 4.5, evidence: 'built payment services in Go' }],
        rationale: 'Strong backend fit.',
        watchOuts: [],
        thinEvidence: false,
      },
    ],
    exclusions: [{ candidateId: 'cand-excluded-9', gate: 'location', reason: 'Based in Lagos, role is Accra-only' }],
  },
};

beforeEach(() => shortlist.mockReset());
afterEach(() => vi.clearAllMocks());

describe('ShortlistSection', () => {
  it('gates ranking behind an explicit generate action (no auto-run on mount)', () => {
    renderWithClient(<ShortlistSection roleId="r1" version={0} />);
    expect(screen.getByText('Generate shortlist')).toBeInTheDocument();
    expect(shortlist).not.toHaveBeenCalled();
  });

  it('renders the ranked matches and surfaces exclusions (never silently dropped)', async () => {
    shortlist.mockResolvedValue(response);
    renderWithClient(<ShortlistSection roleId="r1" version={0} />);

    fireEvent.click(screen.getByText('Generate shortlist'));

    // The ranked, explainable match appears.
    expect(await screen.findByText('Strong backend fit.')).toBeInTheDocument();
    expect(screen.getByText('5 in pool')).toBeInTheDocument();
    // The filtered-out candidate is surfaced with its gate + reason, not dropped.
    expect(screen.getByText('1 candidate filtered out')).toBeInTheDocument();
    expect(screen.getByText('location')).toBeInTheDocument();
    expect(screen.getByText('Based in Lagos, role is Accra-only')).toBeInTheDocument();
  });
});
