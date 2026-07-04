import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { talentApi, type CreateProfileInput } from '../api/talent';

export function useProfile(candidateId: string | undefined) {
  return useQuery({
    queryKey: ['profile', candidateId],
    queryFn: () => talentApi.getProfile(candidateId as string),
    enabled: Boolean(candidateId),
    retry: 0,
  });
}

export function useCreateProfile(candidateId: string | undefined) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateProfileInput) => talentApi.createProfile(candidateId as string, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['profile', candidateId] }),
  });
}
