import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { Contest } from '../../api/types';
import { MyContestsList } from './MyContestsList';

const open: Contest = {
  id: 'k1',
  candidateId: 'c1',
  subject: 'CONTEST_SUBJECT_REPORT_CARD',
  subjectId: 'iv1',
  reason: 'The breakdown missed my Go work.',
  status: 'CONTEST_STATUS_OPEN',
  resolution: '',
};

const resolved: Contest = {
  id: 'k2',
  candidateId: 'c1',
  subject: 'CONTEST_SUBJECT_MATCH',
  subjectId: 'm1',
  reason: 'Location gate was wrong.',
  status: 'CONTEST_STATUS_UPHELD',
  resolution: 'Agreed; rescored.',
};

describe('MyContestsList', () => {
  it('shows an empty state when there are no disputes', () => {
    render(<MyContestsList contests={[]} />);
    expect(screen.getByText(/not disputed any assessments/i)).toBeInTheDocument();
  });

  it('renders an open dispute with its subject and status', () => {
    render(<MyContestsList contests={[open]} />);
    expect(screen.getByText('Report card')).toBeInTheDocument();
    expect(screen.getByText('Under review')).toBeInTheDocument();
    expect(screen.getByText('The breakdown missed my Go work.')).toBeInTheDocument();
  });

  it('shows the reviewer note and status once a dispute is resolved', () => {
    render(<MyContestsList contests={[resolved]} />);
    expect(screen.getByText('Shortlist result')).toBeInTheDocument();
    expect(screen.getByText('Upheld')).toBeInTheDocument();
    expect(screen.getByText('Reviewer note: Agreed; rescored.')).toBeInTheDocument();
  });
});
