import { apiFetch } from './client';
import type { ContestSubject, ListMyContestsResponse, RaiseContestResponse } from './types';

export const contestApi = {
  // The candidate is taken from the auth context server-side, never the body.
  raise: (subject: ContestSubject, subjectId: string, reason: string) =>
    apiFetch<RaiseContestResponse>('/v1/contests', {
      method: 'POST',
      body: { subject, subject_id: subjectId, reason },
    }),
  listMine: (page = 1, pageSize = 20) =>
    apiFetch<ListMyContestsResponse>(`/v1/contests?page.page=${page}&page.page_size=${pageSize}`),
};
