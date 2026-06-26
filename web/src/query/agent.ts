import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { agentApi } from '../api/agent';

export function useApplications(candidateId: string | undefined) {
  return useQuery({
    queryKey: ['applications', candidateId],
    queryFn: () => agentApi.listApplications(candidateId as string),
    enabled: Boolean(candidateId),
    retry: 0,
  });
}

export function useTimeAdvance(candidateId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => agentApi.timeAdvance(candidateId as string),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['applications', candidateId] }),
  });
}
