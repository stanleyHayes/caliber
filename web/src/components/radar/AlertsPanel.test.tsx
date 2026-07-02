import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { MatchAlert } from '../../api/types';
import { AlertsPanel } from './AlertsPanel';

const alerts: MatchAlert[] = [
  {
    id: 'a1',
    type: 'ALERT_TYPE_CANDIDATE_FOR_ROLE',
    roleId: 'r1',
    candidateId: 'c1',
    message: 'New strong candidate for "Backend Engineer": Ama matches 85% on the rubric.',
  },
  {
    id: 'a2',
    type: 'ALERT_TYPE_ROLE_FOR_CANDIDATE',
    roleId: 'r2',
    candidateId: 'c2',
    message: 'New role fits Kofi: "Senior Backend Engineer" (92% match).',
  },
];

describe('AlertsPanel', () => {
  it('renders alert messages with type badges', () => {
    render(<AlertsPanel alerts={alerts} />);
    expect(screen.getByText(alerts[0].message)).toBeInTheDocument();
    expect(screen.getByText(alerts[1].message)).toBeInTheDocument();
    expect(screen.getByText('Candidate for role')).toBeInTheDocument();
    expect(screen.getByText('Role for candidate')).toBeInTheDocument();
  });

  it('shows an empty state when there are no alerts', () => {
    render(<AlertsPanel alerts={[]} />);
    expect(screen.getByText('No alerts yet.')).toBeInTheDocument();
  });

  it('renders pagination controls when there is more than one page', () => {
    render(<AlertsPanel alerts={alerts} page={1} pageCount={3} onPageChange={vi.fn()} />);
    expect(screen.getByRole('navigation')).toBeInTheDocument();
  });

  it('hides pagination controls when there is only one page', () => {
    render(<AlertsPanel alerts={alerts} page={1} pageCount={1} onPageChange={vi.fn()} />);
    expect(screen.queryByRole('navigation')).not.toBeInTheDocument();
  });
});
