import { keepPreviousData, useQuery } from '@tanstack/react-query';

import { radarApi } from '../api/radar';

export function useTimeToShortlist() {
  return useQuery({ queryKey: ['radar', 'ttsl'], queryFn: radarApi.timeToShortlist, retry: 0 });
}

export function usePool(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ['radar', 'pool', page, pageSize],
    queryFn: () => radarApi.pool(page, pageSize),
    retry: 0,
    placeholderData: keepPreviousData,
  });
}

export function useSupplyDemand() {
  return useQuery({ queryKey: ['radar', 'supply-demand'], queryFn: radarApi.supplyDemand, retry: 0 });
}

export function useAlerts(page = 1, pageSize = 20) {
  return useQuery({
    queryKey: ['radar', 'alerts', page, pageSize],
    queryFn: () => radarApi.alerts(page, pageSize),
    retry: 0,
    placeholderData: keepPreviousData,
  });
}
