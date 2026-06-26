import { useQuery } from '@tanstack/react-query';

import { radarApi } from '../api/radar';

export function useTimeToShortlist() {
  return useQuery({ queryKey: ['radar', 'ttsl'], queryFn: radarApi.timeToShortlist, retry: 0 });
}
export function usePool() {
  return useQuery({ queryKey: ['radar', 'pool'], queryFn: () => radarApi.pool(), retry: 0 });
}
export function useSupplyDemand() {
  return useQuery({ queryKey: ['radar', 'supply-demand'], queryFn: radarApi.supplyDemand, retry: 0 });
}
