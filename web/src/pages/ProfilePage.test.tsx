import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiError, type TalentProfile, type User } from '../api/types';
import { useAuthStore } from '../stores/auth';
import { ProfilePage } from './ProfilePage';

type ProfileResult = { isPending: boolean; error: unknown; data?: { profile: TalentProfile } };
type CreateResult = { mutate: ReturnType<typeof vi.fn>; isPending: boolean; isError: boolean; error: Error | null; data?: { profile: TalentProfile } };

let profileResult: ProfileResult;
let createResult: CreateResult;
const mutate = vi.fn();
vi.mock('../query/talent', () => ({
  useProfile: () => profileResult,
  useCreateProfile: () => createResult,
}));

const user: User = {
  id: 'cand-1',
  email: 'ama@example.com',
  role: 'USER_ROLE_CANDIDATE',
  name: 'Ama',
  createdAt: '2026-01-01T00:00:00Z',
};

const profile: TalentProfile = {
  id: 'p1',
  candidateId: 'cand-1',
  summary: 'Backend engineer.',
  passportStatus: 'PASSPORT_STATUS_SCREENED',
  competencies: [{ name: 'Go', level: 4, evidenceQuote: 'built services in Go', sourceSpan: 'CV' }],
};

beforeEach(() => {
  useAuthStore.setState({ user });
  mutate.mockReset();
  profileResult = { isPending: false, error: new ApiError(404, 'not found') };
  createResult = { mutate, isPending: false, isError: false, error: null };
});
afterEach(() => {
  useAuthStore.getState().clear();
  localStorage.clear();
});

describe('ProfilePage', () => {
  it('offers to build a profile when the candidate has none yet', () => {
    render(<ProfilePage />);
    expect(screen.getByText('Create your profile')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Build my profile' })).toBeDisabled();
  });

  it('submits the pasted CV to extract a profile', () => {
    render(<ProfilePage />);
    fireEvent.change(screen.getByPlaceholderText('Paste your CV text…'), {
      target: { value: 'I built payment services in Go.' },
    });
    fireEvent.change(screen.getByLabelText('Location'), { target: { value: 'Accra' } });
    const build = screen.getByRole('button', { name: 'Build my profile' });
    expect(build).toBeEnabled();
    fireEvent.click(build);

    expect(mutate).toHaveBeenCalledWith({
      cvText: 'I built payment services in Go.',
      intake: { location: 'Accra', targetTitles: [], salaryFloor: 0 },
    });
  });

  it('shows the existing passport with a re-extract action', () => {
    profileResult = { isPending: false, error: null, data: { profile } };
    render(<ProfilePage />);
    expect(screen.getByText('Your Talent Passport')).toBeInTheDocument();
    expect(screen.getByText('Update from a new CV')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Re-extract profile' })).toBeInTheDocument();
  });
});
