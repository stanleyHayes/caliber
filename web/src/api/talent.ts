import { apiFetch } from './client';
import type { ProfileResponse } from './types';

export interface IntakeInput {
  location: string;
  targetTitles: string[];
  salaryFloor: number;
  dealBreakers: string[];
}

export interface CreateProfileInput {
  cvText: string;
  cvFile?: string;
  cvFilename?: string;
  intake: IntakeInput;
}

export const talentApi = {
  createProfile: (candidateId: string, input: CreateProfileInput) =>
    apiFetch<ProfileResponse>(`/v1/candidates/${encodeURIComponent(candidateId)}/profile:fromCv`, {
      method: 'POST',
      body: {
        candidate_id: candidateId,
        cv_text: input.cvText,
        cv_file: input.cvFile,
        cv_filename: input.cvFilename,
        intake: {
          location: input.intake.location,
          target_titles: input.intake.targetTitles,
          salary_floor: input.intake.salaryFloor,
          deal_breakers: input.intake.dealBreakers,
        },
      },
    }),
  getProfile: (candidateId: string) =>
    apiFetch<ProfileResponse>(`/v1/candidates/${encodeURIComponent(candidateId)}/profile`),
};
