import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { flowApi } from '../api/flow';
import type { RoleSpec, Rubric } from '../api/types';

export function useRecordRejection() {
  return useMutation({
    mutationFn: ({ roleId, candidateId, reason, humanApproved }: {
      roleId: string;
      candidateId: string;
      reason: string;
      humanApproved: boolean;
    }) => flowApi.recordRejection(roleId, candidateId, reason, humanApproved),
  });
}

export function useGenerateRole() {
  return useMutation({
    mutationFn: ({ employerId, freeText }: { employerId: string; freeText: string }) =>
      flowApi.generateRole(employerId, freeText),
  });
}

export function useRoles(employerId: string | undefined, page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ['roles', employerId, page, pageSize],
    queryFn: () => flowApi.listRoles(employerId as string, page, pageSize),
    enabled: Boolean(employerId),
    retry: 0,
    placeholderData: keepPreviousData,
  });
}

export function useOpenRoles(page = 1, pageSize = 20, enabled = true) {
  return useQuery({
    queryKey: ['openRoles', page, pageSize],
    queryFn: () => flowApi.listOpenRoles(page, pageSize),
    enabled,
    retry: 0,
    placeholderData: keepPreviousData,
  });
}

export function useRole(roleId: string | undefined, enabled = true) {
  return useQuery({
    queryKey: ['role', roleId],
    queryFn: () => flowApi.getRole(roleId as string),
    enabled: enabled && Boolean(roleId),
    retry: 0,
  });
}

export function useUpdateRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ roleId, spec, rubric }: { roleId: string; spec: RoleSpec; rubric: Rubric }) =>
      flowApi.updateRole(roleId, spec, rubric),
    onSuccess: (data, variables) => {
      queryClient.setQueryData(['role', variables.roleId], data);
      void queryClient.invalidateQueries({ queryKey: ['roles'] });
      void queryClient.invalidateQueries({ queryKey: ['openRoles'] });
    },
  });
}

export function useDeleteRole() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (roleId: string) => flowApi.deleteRole(roleId),
    onSuccess: (_data, roleId) => {
      queryClient.removeQueries({ queryKey: ['role', roleId] });
      void queryClient.invalidateQueries({ queryKey: ['roles'] });
      void queryClient.invalidateQueries({ queryKey: ['openRoles'] });
    },
  });
}

export function useShortlist(roleId: string | undefined, page: number, pageSize: number, enabled: boolean, version: number) {
  return useQuery({
    queryKey: ['shortlist', roleId, page, pageSize, version],
    queryFn: () => flowApi.shortlist(roleId as string, page, pageSize),
    enabled: enabled && Boolean(roleId),
    retry: 0, // a 501 (matching disabled without a DB) should not retry
    placeholderData: keepPreviousData, // keep the old ranking visible while re-ranking
  });
}
