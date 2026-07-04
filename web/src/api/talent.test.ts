import { beforeEach, describe, expect, it, vi } from 'vitest';

import { apiFetch } from './client';
import { talentApi } from './talent';

vi.mock('./client', () => ({ apiFetch: vi.fn() }));

describe('talentApi', () => {
  beforeEach(() => vi.clearAllMocks());

  it('creates a profile from uploaded CV bytes and guided intake', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ profile: { id: 'p1' } });

    await talentApi.createProfile('cand-1', {
      cvText: '',
      cvFile: 'U2VuaW9yIEdvIGVuZ2luZWVy',
      cvFilename: 'resume.pdf',
      intake: {
        location: 'Accra',
        targetTitles: ['Backend Engineer'],
        salaryFloor: 120000,
        dealBreakers: ['No relocation'],
      },
    });

    expect(apiFetch).toHaveBeenCalledWith('/v1/candidates/cand-1/profile:fromCv', {
      method: 'POST',
      body: {
        candidate_id: 'cand-1',
        cv_text: '',
        cv_file: 'U2VuaW9yIEdvIGVuZ2luZWVy',
        cv_filename: 'resume.pdf',
        intake: {
          location: 'Accra',
          target_titles: ['Backend Engineer'],
          salary_floor: 120000,
          deal_breakers: ['No relocation'],
        },
      },
    });
  });
});
