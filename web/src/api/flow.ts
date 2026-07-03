import { apiFetch } from './client';
import type {
  GenerateRoleResponse,
  ListRolesResponse,
  RecordRejectionResponse,
  Role,
  RoleSpec,
  Rubric,
  ShortlistResponse,
} from './types';

export const flowApi = {
  generateRole: (employerId: string, freeText: string) =>
    apiFetch<GenerateRoleResponse>('/v1/roles:generate', {
      method: 'POST',
      body: { employer_id: employerId, free_text: freeText },
    }),
  listRoles: (employerId: string, page = 1, pageSize = 20) =>
    apiFetch<ListRolesResponse>(
      `/v1/roles?employer_id=${encodeURIComponent(employerId)}&page.page=${page}&page.page_size=${pageSize}`,
    ),
  updateRole: (roleId: string, spec: RoleSpec, rubric: Rubric) =>
    apiFetch<{ role: Role }>(`/v1/roles/${encodeURIComponent(roleId)}`, {
      method: 'PATCH',
      body: { role_id: roleId, spec, rubric },
    }),
  // The backend produces a ranked, explainable shortlist with page metadata.
  shortlist: (roleId: string, page: number, pageSize: number) =>
    apiFetch<ShortlistResponse>(
      `/v1/roles/${encodeURIComponent(roleId)}/shortlist?page.page=${page}&page.page_size=${pageSize}`,
    ),
  // A decline is never automatic: human_approved must be true and the approving
  // human is taken from the auth context server-side, not this body.
  recordRejection: (roleId: string, candidateId: string, reason: string, humanApproved: boolean) =>
    apiFetch<RecordRejectionResponse>(`/v1/roles/${encodeURIComponent(roleId)}/rejections`, {
      method: 'POST',
      body: { role_id: roleId, candidate_id: candidateId, reason, human_approved: humanApproved },
    }),
};
