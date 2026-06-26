import { apiFetch } from './client';
import type { ListApplicationsResponse, TimeAdvanceResponse } from './types';

export const agentApi = {
  timeAdvance: (candidateId: string) =>
    apiFetch<TimeAdvanceResponse>(`/v1/candidates/${encodeURIComponent(candidateId)}/agent:timeAdvance`, {
      method: 'POST',
      body: { candidate_id: candidateId },
    }),
  listApplications: (candidateId: string, pageSize = 20) =>
    apiFetch<ListApplicationsResponse>(
      `/v1/candidates/${encodeURIComponent(candidateId)}/applications?page.page=1&page.page_size=${pageSize}`,
    ),
};
