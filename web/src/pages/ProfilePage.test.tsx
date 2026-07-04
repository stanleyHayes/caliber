import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ApiError, type Contest, type TalentProfile, type User } from '../api/types';
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

let contestsResult: {
  data?: { contests: Contest[]; page?: { page: number; pageSize: number; totalItems: number; totalPages: number } };
};
const useMyContestsMock = vi.hoisted(() => vi.fn());
vi.mock('../query/contest', () => ({ useMyContests: (...args: unknown[]) => useMyContestsMock(...args) }));

const exportMutate = vi.fn();
let exportResult: { mutate: ReturnType<typeof vi.fn>; isPending: boolean; isError: boolean; error: Error | null };
vi.mock('../query/privacy', () => ({ useExportMyData: () => exportResult }));
// DeleteAccount is covered by its own test; stub it here so this page test does
// not pull in its router/mutation dependencies.
vi.mock('../components/privacy/DeleteAccount', () => ({ DeleteAccount: () => null }));

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

const contest: Contest = {
  id: 'contest-1',
  candidateId: 'cand-1',
  subject: 'CONTEST_SUBJECT_MATCH',
  subjectId: 'match-1',
  reason: 'The evidence quote missed my real Go work.',
  status: 'CONTEST_STATUS_OPEN',
  resolution: '',
};

beforeEach(() => {
  useAuthStore.setState({ user });
  mutate.mockReset();
  useMyContestsMock.mockReset();
  exportMutate.mockReset();
  profileResult = { isPending: false, error: new ApiError(404, 'not found') };
  createResult = { mutate, isPending: false, isError: false, error: null };
  contestsResult = { data: { contests: [] } };
  useMyContestsMock.mockImplementation(() => contestsResult);
  exportResult = { mutate: exportMutate, isPending: false, isError: false, error: null };
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
    fireEvent.change(screen.getByPlaceholderText('Paste your CV text...'), {
      target: { value: 'I built payment services in Go.' },
    });
    fireEvent.change(screen.getByLabelText('Location'), { target: { value: 'Accra' } });
    fireEvent.change(screen.getByLabelText('Target titles'), { target: { value: 'Backend Engineer, Platform Engineer' } });
    fireEvent.change(screen.getByLabelText('Salary floor'), { target: { value: '120000' } });
    fireEvent.change(screen.getByLabelText('Deal-breakers'), { target: { value: 'No relocation\nNo unpaid take-home' } });
    const build = screen.getByRole('button', { name: 'Build my profile' });
    expect(build).toBeEnabled();
    fireEvent.click(build);

    expect(mutate).toHaveBeenCalledWith({
      cvText: 'I built payment services in Go.',
      cvFile: undefined,
      cvFilename: undefined,
      intake: {
        location: 'Accra',
        targetTitles: ['Backend Engineer', 'Platform Engineer'],
        salaryFloor: 120000,
        dealBreakers: ['No relocation', 'No unpaid take-home'],
      },
    });
  });

  it('submits an uploaded CV file instead of pasted text', async () => {
    render(<ProfilePage />);
    const file = new File(['Senior Go engineer'], 'resume.pdf', { type: 'application/pdf' });
    fireEvent.change(screen.getByLabelText('Upload CV file'), { target: { files: [file] } });
    fireEvent.change(screen.getByLabelText('Location'), { target: { value: 'Kumasi' } });
    fireEvent.click(screen.getByRole('button', { name: 'Build my profile' }));

    await waitFor(() => expect(mutate).toHaveBeenCalledTimes(1));
    expect(mutate).toHaveBeenCalledWith({
      cvText: '',
      cvFile: btoa('Senior Go engineer'),
      cvFilename: 'resume.pdf',
      intake: { location: 'Kumasi', targetTitles: [], salaryFloor: 0, dealBreakers: [] },
    });
    expect(screen.getByText('resume.pdf')).toBeInTheDocument();
  });

  it('shows the existing passport with a re-extract action', () => {
    profileResult = { isPending: false, error: null, data: { profile } };
    render(<ProfilePage />);
    expect(screen.getByText('Your Talent Passport')).toBeInTheDocument();
    expect(screen.getByText('Update from a new CV')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Re-extract profile' })).toBeInTheDocument();
  });

  it('lets the candidate download a full copy of their data (DSAR right of access)', () => {
    render(<ProfilePage />);
    expect(screen.getByText('Your data')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Download my data' }));
    expect(exportMutate).toHaveBeenCalledTimes(1);
  });

  it('requests the selected server page when paginating disputes', () => {
    contestsResult = {
      data: { contests: [contest], page: { page: 1, pageSize: 20, totalItems: 44, totalPages: 3 } },
    };
    render(<ProfilePage />);
    expect(useMyContestsMock).toHaveBeenCalledWith(true, 1, 20);
    fireEvent.click(screen.getByRole('button', { name: /go to page 2/i }));
    expect(useMyContestsMock).toHaveBeenLastCalledWith(true, 2, 20);
  });
});
