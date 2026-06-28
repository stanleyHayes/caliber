import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { SupplyDemandItem } from '../../api/types';
import { SupplyDemandPanel } from './SupplyDemandPanel';

describe('SupplyDemandPanel', () => {
  it('lists each role family with its open/available/gap counts', () => {
    const items: SupplyDemandItem[] = [
      { roleFamily: 'mid', openRoles: 3, availableCandidates: 5, gap: -2 },
      { roleFamily: 'senior', openRoles: 2, availableCandidates: 1, gap: 1 },
    ];
    render(<SupplyDemandPanel items={items} />);
    expect(screen.getByText('mid')).toBeInTheDocument();
    expect(screen.getByText(/3 open · 5 candidates · gap -2/)).toBeInTheDocument();
    expect(screen.getByText('senior')).toBeInTheDocument();
    expect(screen.getByText(/2 open · 1 candidates · gap 1/)).toBeInTheDocument();
  });

  it('shows an empty state when there are no open roles', () => {
    render(<SupplyDemandPanel items={[]} />);
    expect(screen.getByText('No open roles yet.')).toBeInTheDocument();
  });
});
