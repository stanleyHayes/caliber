import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { PoolCandidate } from '../../api/types';
import { PoolPanel } from './PoolPanel';

describe('PoolPanel', () => {
  it('lists candidates with name, passport status, and headline score', () => {
    const candidates: PoolCandidate[] = [
      { candidateId: 'c1', name: 'Ama Mensah', passportStatus: 'PASSPORT_STATUS_SCREENED', headlineScore: 0.91 },
    ];
    render(<PoolPanel candidates={candidates} />);
    expect(screen.getByText('Ama Mensah')).toBeInTheDocument();
    expect(screen.getByText('Screened')).toBeInTheDocument();
    expect(screen.getByText('91%')).toBeInTheDocument();
  });

  it('falls back to a short id when a candidate has no name', () => {
    const candidates: PoolCandidate[] = [
      { candidateId: 'abcdef1234567890', name: '', passportStatus: 'PASSPORT_STATUS_CV_ONLY', headlineScore: 0 },
    ];
    render(<PoolPanel candidates={candidates} />);
    expect(screen.getByText('abcdef12')).toBeInTheDocument();
  });

  it('shows an empty state for an empty pool', () => {
    render(<PoolPanel candidates={[]} />);
    expect(screen.getByText('No candidates in the pool yet.')).toBeInTheDocument();
  });
});
