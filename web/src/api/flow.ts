import { apiFetch } from './client';
import type { GenerateRoleResponse, ShortlistResponse } from './types';

export const flowApi = {
  generateRole: (employerId: string, freeText: string) =>
    apiFetch<GenerateRoleResponse>('/v1/roles:generate', {
      method: 'POST',
      body: { employer_id: employerId, free_text: freeText },
    }),
  // The backend produces a ranked top-N shortlist; pageSize bounds the pool size.
  shortlist: (roleId: string, pageSize: number) =>
    apiFetch<ShortlistResponse>(
      `/v1/roles/${encodeURIComponent(roleId)}/shortlist?page.page=1&page.page_size=${pageSize}`,
    ),
};
