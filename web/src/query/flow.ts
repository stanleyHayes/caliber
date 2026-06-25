import { useMutation, useQuery } from '@tanstack/react-query';

import { flowApi } from '../api/flow';

export function useGenerateRole() {
  return useMutation({
    mutationFn: ({ employerId, freeText }: { employerId: string; freeText: string }) =>
      flowApi.generateRole(employerId, freeText),
  });
}

export function useShortlist(roleId: string | undefined, pageSize: number, enabled: boolean) {
  return useQuery({
    queryKey: ['shortlist', roleId, pageSize],
    queryFn: () => flowApi.shortlist(roleId as string, pageSize),
    enabled: enabled && Boolean(roleId),
    retry: 0, // a 501 (matching disabled without a DB) should not retry
  });
}
