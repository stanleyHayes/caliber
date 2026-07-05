import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { radarApi } from '../api/radar';

export function useTimeToShortlist(enabled = true) {
  return useQuery({ queryKey: ['radar', 'ttsl'], queryFn: radarApi.timeToShortlist, enabled, retry: 0 });
}

export function usePool(page = 1, pageSize = 20, enabled = true) {
  return useQuery({
    queryKey: ['radar', 'pool', page, pageSize],
    queryFn: () => radarApi.pool(page, pageSize),
    enabled,
    retry: 0,
    placeholderData: keepPreviousData,
  });
}

export function useSupplyDemand(enabled = true) {
  return useQuery({ queryKey: ['radar', 'supply-demand'], queryFn: radarApi.supplyDemand, enabled, retry: 0 });
}

export function useAlerts(page = 1, pageSize = 20, enabled = true) {
  return useQuery({
    queryKey: ['radar', 'alerts', page, pageSize],
    queryFn: () => radarApi.alerts(page, pageSize),
    enabled,
    retry: 0,
    placeholderData: keepPreviousData,
  });
}
