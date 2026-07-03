import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { agentApi } from '../api/agent';

export function useApplications(candidateId: string | undefined, page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ['applications', candidateId, page, pageSize],
    queryFn: () => agentApi.listApplications(candidateId as string, page, pageSize),
    enabled: Boolean(candidateId),
    retry: 0,
    placeholderData: keepPreviousData,
  });
}

export function useTimeAdvance(candidateId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => agentApi.timeAdvance(candidateId as string),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['applications', candidateId] }),
  });
}
