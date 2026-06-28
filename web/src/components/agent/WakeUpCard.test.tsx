import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { WakeUpView } from '../../api/types';
import { WakeUpCard } from './WakeUpCard';

const wakeUp: WakeUpView = {
  newMatches: 3,
  applicationsSubmitted: 2,
  screeningsCompleted: 1,
  employersInterested: 4,
  highlights: ['Applied to Senior Go Engineer at Acme', 'Cleared screening for Payments Lead'],
};

describe('WakeUpCard', () => {
  it('renders the four progress stats with their labels', () => {
    render(<WakeUpCard wakeUp={wakeUp} />);
    expect(screen.getByText('While you were away')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
    expect(screen.getByText('New matches')).toBeInTheDocument();
    expect(screen.getByText('Applications submitted')).toBeInTheDocument();
    expect(screen.getByText('Screenings completed')).toBeInTheDocument();
    expect(screen.getByText('Employers interested')).toBeInTheDocument();
  });

  it('lists highlights when present', () => {
    render(<WakeUpCard wakeUp={wakeUp} />);
    expect(screen.getByText('Applied to Senior Go Engineer at Acme')).toBeInTheDocument();
    expect(screen.getByText('Cleared screening for Payments Lead')).toBeInTheDocument();
  });

  it('renders without a highlights list when there are none', () => {
    render(<WakeUpCard wakeUp={{ ...wakeUp, highlights: [] }} />);
    expect(screen.getByText('While you were away')).toBeInTheDocument();
    expect(screen.queryByRole('list')).not.toBeInTheDocument();
  });
});
