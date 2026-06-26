import { apiFetch } from './client';
import type { GenerateRoleResponse, Role, RoleSpec, Rubric, ShortlistResponse } from './types';

export const flowApi = {
  generateRole: (employerId: string, freeText: string) =>
    apiFetch<GenerateRoleResponse>('/v1/roles:generate', {
      method: 'POST',
      body: { employer_id: employerId, free_text: freeText },
    }),
  updateRole: (roleId: string, spec: RoleSpec, rubric: Rubric) =>
    apiFetch<{ role: Role }>(`/v1/roles/${encodeURIComponent(roleId)}`, {
      method: 'PATCH',
      body: { role_id: roleId, spec, rubric },
    }),
  // The backend produces a ranked top-N shortlist; pageSize bounds the pool size.
  shortlist: (roleId: string, pageSize: number) =>
    apiFetch<ShortlistResponse>(
      `/v1/roles/${encodeURIComponent(roleId)}/shortlist?page.page=1&page.page_size=${pageSize}`,
    ),
};
