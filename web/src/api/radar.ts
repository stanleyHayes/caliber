import { apiFetch } from './client';
import type {
  GetAlertsResponse,
  GetPoolResponse,
  GetSupplyDemandResponse,
  GetTimeToShortlistResponse,
} from './types';

export const radarApi = {
  timeToShortlist: () => apiFetch<GetTimeToShortlistResponse>('/v1/radar/time-to-shortlist'),
  pool: (page = 1, pageSize = 20) =>
    apiFetch<GetPoolResponse>(`/v1/radar/pool?page.page=${page}&page.page_size=${pageSize}`),
  supplyDemand: () => apiFetch<GetSupplyDemandResponse>('/v1/radar/supply-demand'),
  alerts: (page = 1, pageSize = 20) =>
    apiFetch<GetAlertsResponse>(`/v1/radar/alerts?page.page=${page}&page.page_size=${pageSize}`),
};
