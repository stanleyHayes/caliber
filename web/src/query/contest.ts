import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { contestApi } from '../api/contest';
import type { ContestSubject } from '../api/types';

export function useMyContests(enabled = true, page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ['contests', 'mine', page, pageSize],
    queryFn: () => contestApi.listMine(page, pageSize),
    enabled,
    retry: 0,
    placeholderData: keepPreviousData,
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
