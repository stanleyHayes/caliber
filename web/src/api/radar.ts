import { apiFetch } from './client';
import type { GetPoolResponse, GetSupplyDemandResponse, GetTimeToShortlistResponse } from './types';

export const radarApi = {
  timeToShortlist: () => apiFetch<GetTimeToShortlistResponse>('/v1/radar/time-to-shortlist'),
  pool: (pageSize = 20) => apiFetch<GetPoolResponse>(`/v1/radar/pool?page.page=1&page.page_size=${pageSize}`),
  supplyDemand: () => apiFetch<GetSupplyDemandResponse>('/v1/radar/supply-demand'),
};
