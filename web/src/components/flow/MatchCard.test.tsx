import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';

import type { Match } from '../../api/types';
import { MatchCard } from './MatchCard';

// MatchCard links the candidate id to /candidates/:id, so it needs a Router.
const wrap = (match: Match, rank = 1) => (
  <MemoryRouter>
    <MatchCard match={match} rank={rank} />
  </MemoryRouter>
);

// MatchCard embeds DeclineCandidate, which uses useRecordRejection.
vi.mock('../../query/flow', () => ({
  useRecordRejection: () => ({ mutate: vi.fn(), isPending: false, isError: false, isSuccess: false, error: null }),
}));

const baseMatch: Match = {
  id: 'm1',
  roleId: 'r1',
  candidateId: 'cand-0123456789',
  overallScore: 0.87,
  confidence: 'CONFIDENCE_HIGH',
  breakdown: [
    { competency: 'Go', score: 4.5, evidence: 'built payment services in Go' },
    { competency: 'SQL', score: 3.0, evidence: 'designed schemas' },
  ],
  rationale: 'Strong backend fit with production Go experience.',
  watchOuts: ['Limited mentoring experience'],
  thinEvidence: false,
};

describe('MatchCard', () => {
  it('renders the explainable match: fit score, confidence, rationale, and per-competency breakdown', () => {
    render(wrap(baseMatch));
    expect(screen.getByText('87%')).toBeInTheDocument();
    expect(screen.getByText('High')).toBeInTheDocument();
    expect(screen.getByText('Strong backend fit with production Go experience.')).toBeInTheDocument();
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('SQL')).toBeInTheDocument();
    expect(screen.getByText('“built payment services in Go”')).toBeInTheDocument();
    expect(screen.getByText('Limited mentoring experience')).toBeInTheDocument();
  });

  it('flags thin evidence only when the match is sparsely supported', () => {
    const { rerender } = render(wrap(baseMatch));
    expect(screen.queryByText('thin evidence')).not.toBeInTheDocument();

    rerender(wrap({ ...baseMatch, thinEvidence: true }));
    expect(screen.getByText('thin evidence')).toBeInTheDocument();
  });
});
