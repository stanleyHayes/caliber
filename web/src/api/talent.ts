import { apiFetch } from './client';
import type { ProfileResponse } from './types';

export interface IntakeInput {
  location: string;
  targetTitles: string[];
  salaryFloor: number;
}

export const talentApi = {
  createProfile: (candidateId: string, cvText: string, intake: IntakeInput) =>
    apiFetch<ProfileResponse>(`/v1/candidates/${encodeURIComponent(candidateId)}/profile:fromCv`, {
      method: 'POST',
      body: {
        candidate_id: candidateId,
        cv_text: cvText,
        intake: { location: intake.location, target_titles: intake.targetTitles, salary_floor: intake.salaryFloor },
      },
    }),
  getProfile: (candidateId: string) =>
    apiFetch<ProfileResponse>(`/v1/candidates/${encodeURIComponent(candidateId)}/profile`),
};
