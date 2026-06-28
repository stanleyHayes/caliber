import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { RoleSpec } from '../../api/types';
import { RoleSpecCard } from './RoleSpecCard';

const spec: RoleSpec = {
  title: 'Senior Go Engineer',
  location: 'Accra',
  seniority: 'SENIORITY_SENIOR',
  availability: 'Full-time',
  responsibilities: ['Lead the payments platform'],
  mustHaves: ['Go', 'PostgreSQL'],
  niceToHaves: ['gRPC'],
  salaryBand: { currency: 'GHS', low: 1000, high: 5000 },
};

describe('RoleSpecCard', () => {
  it('renders the structured spec: title, seniority, location, availability, and salary band', () => {
    render(<RoleSpecCard spec={spec} />);
    expect(screen.getByText('Senior Go Engineer')).toBeInTheDocument();
    expect(screen.getByText('Senior')).toBeInTheDocument();
    expect(screen.getByText('Accra')).toBeInTheDocument();
    expect(screen.getByText('Full-time')).toBeInTheDocument();
    expect(screen.getByText('GHS 1000–5000')).toBeInTheDocument();
  });

  it('renders responsibilities, must-haves, and nice-to-haves as labelled groups', () => {
    render(<RoleSpecCard spec={spec} />);
    expect(screen.getByText('Responsibilities')).toBeInTheDocument();
    expect(screen.getByText('Lead the payments platform')).toBeInTheDocument();
    expect(screen.getByText('Must-haves')).toBeInTheDocument();
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('PostgreSQL')).toBeInTheDocument();
    expect(screen.getByText('Nice-to-haves')).toBeInTheDocument();
    expect(screen.getByText('gRPC')).toBeInTheDocument();
  });

  it('omits an empty group and shows "Not specified" when no salary band is set', () => {
    render(
      <RoleSpecCard
        spec={{ ...spec, niceToHaves: [], salaryBand: { currency: 'GHS', low: 0, high: 0 } }}
      />,
    );
    expect(screen.queryByText('Nice-to-haves')).not.toBeInTheDocument();
    expect(screen.getByText('Not specified')).toBeInTheDocument();
  });
});
