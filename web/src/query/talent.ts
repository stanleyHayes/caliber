import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { talentApi, type IntakeInput } from '../api/talent';

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
    mutationFn: ({ cvText, intake }: { cvText: string; intake: IntakeInput }) =>
      talentApi.createProfile(candidateId as string, cvText, intake),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['profile', candidateId] }),
  });
}
