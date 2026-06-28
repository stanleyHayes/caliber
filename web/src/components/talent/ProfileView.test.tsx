import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { TalentProfile } from '../../api/types';
import { ProfileView } from './ProfileView';

const profile: TalentProfile = {
  id: 'p1',
  candidateId: 'c1',
  summary: 'Backend engineer with payments experience.',
  passportStatus: 'PASSPORT_STATUS_SCREENED',
  competencies: [
    { name: 'Go', level: 4.5, evidenceQuote: 'built payment services in Go', sourceSpan: 'CV §2' },
    { name: 'SQL', level: 3, evidenceQuote: '', sourceSpan: '' },
  ],
};

describe('ProfileView', () => {
  it('renders the passport status, summary, and per-competency levels', () => {
    render(<ProfileView profile={profile} />);
    expect(screen.getByText('Your Talent Passport')).toBeInTheDocument();
    expect(screen.getByText('Screened')).toBeInTheDocument();
    expect(screen.getByText('Backend engineer with payments experience.')).toBeInTheDocument();
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('4.5 / 5')).toBeInTheDocument();
    expect(screen.getByText('3.0 / 5')).toBeInTheDocument();
  });

  it('surfaces the evidence quote when present (no-fabrication: claims cite their source)', () => {
    render(<ProfileView profile={profile} />);
    expect(screen.getByText('“built payment services in Go”')).toBeInTheDocument();
  });

  it('omits the evidence line for a competency with no quote rather than inventing one', () => {
    render(<ProfileView profile={{ ...profile, competencies: [profile.competencies[1]] }} />);
    // SQL has an empty evidenceQuote — no quotation marks should be rendered.
    expect(screen.queryByText(/“.*”/)).not.toBeInTheDocument();
  });
});
