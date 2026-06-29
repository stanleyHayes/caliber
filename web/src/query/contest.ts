import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { contestApi } from '../api/contest';
import type { ContestSubject } from '../api/types';

export function useMyContests(enabled = true) {
  return useQuery({
    queryKey: ['contests', 'mine'],
    queryFn: () => contestApi.listMine(),
    enabled,
    retry: 0,
  });
}

export function useRaiseContest() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ subject, subjectId, reason }: { subject: ContestSubject; subjectId: string; reason: string }) =>
      contestApi.raise(subject, subjectId, reason),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['contests', 'mine'] }),
  });
}
