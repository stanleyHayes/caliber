import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { Application } from '../../api/types';
import { ApplicationsList } from './ApplicationsList';

const agentApp: Application = {
  id: 'a1',
  roleId: 'role-abcdef123456',
  candidateId: 'c1',
  source: 'APPLICATION_SOURCE_AGENT',
  tailoredSummary: 'Tailored to the payments platform role.',
  status: 'APPLICATION_STATUS_SUBMITTED',
};

describe('ApplicationsList', () => {
  it('shows an empty-state prompt when there are no applications', () => {
    render(<ApplicationsList applications={[]} />);
    expect(screen.getByText(/No applications yet/i)).toBeInTheDocument();
  });

  it('renders an application with its status, tailored summary, and short role id', () => {
    render(<ApplicationsList applications={[agentApp]} />);
    expect(screen.getByText('Submitted')).toBeInTheDocument();
    expect(screen.getByText('Tailored to the payments platform role.')).toBeInTheDocument();
    expect(screen.getByText('role role-abc')).toBeInTheDocument();
  });

  it('marks agent-sourced applications and omits the badge for manual ones', () => {
    const { rerender } = render(<ApplicationsList applications={[agentApp]} />);
    expect(screen.getByText('by your agent')).toBeInTheDocument();

    rerender(<ApplicationsList applications={[{ ...agentApp, source: 'APPLICATION_SOURCE_MANUAL' }]} />);
    expect(screen.queryByText('by your agent')).not.toBeInTheDocument();
  });
});
