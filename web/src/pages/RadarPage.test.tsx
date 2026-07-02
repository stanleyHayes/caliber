import { render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  ApiError,
  type GetAlertsResponse,
  type GetPoolResponse,
  type SupplyDemandItem,
  type TimeToShortlistMetric,
} from '../api/types';
import { RadarPage } from './RadarPage';

type Q<T> = { isPending: boolean; isError: boolean; error: unknown; data?: T };
let ttslResult: Q<{ metric: TimeToShortlistMetric }>;
let supplyResult: Q<{ items: SupplyDemandItem[] }>;
let poolResult: Q<GetPoolResponse>;
let alertsResult: Q<GetAlertsResponse>;

vi.mock('../query/radar', () => ({
  useTimeToShortlist: () => ttslResult,
  useSupplyDemand: () => supplyResult,
  usePool: () => poolResult,
  useAlerts: () => alertsResult,
}));

const ok = <T,>(data: T): Q<T> => ({ isPending: false, isError: false, error: null, data });

beforeEach(() => {
  ttslResult = ok({ metric: { baselineHours: 40, currentHours: 4, improvementFactor: 10 } });
  supplyResult = ok({ items: [{ roleFamily: 'Backend', openRoles: 3, availableCandidates: 12, gap: -9 }] });
  poolResult = ok({
    candidates: [{ candidateId: 'c1', name: 'Ama Mensah', passportStatus: 'PASSPORT_STATUS_SCREENED', headlineScore: 0.9 }],
    page: { page: 1, pageSize: 20, totalItems: 1, totalPages: 1 },
  });
  alertsResult = ok({
    alerts: [
      {
        id: 'a1',
        type: 'ALERT_TYPE_CANDIDATE_FOR_ROLE',
        roleId: 'r1',
        candidateId: 'c1',
        message: 'New strong candidate for "Backend Engineer": Ama matches 90% on the rubric.',
      },
    ],
    page: { page: 1, pageSize: 20, totalItems: 1, totalPages: 1 },
  });
});
afterEach(() => vi.clearAllMocks());

describe('RadarPage', () => {
  it('renders the four radar panels from their data', () => {
    render(<RadarPage />);
    expect(screen.getByRole('heading', { name: 'Talent Radar' })).toBeInTheDocument();
    expect(screen.getByText('Backend')).toBeInTheDocument();
    expect(screen.getByText('Ama Mensah')).toBeInTheDocument();
    expect(screen.getByText('Match alerts')).toBeInTheDocument();
    expect(screen.getByText(alertsResult.data!.alerts[0].message)).toBeInTheDocument();
  });

  it('explains each panel when the environment is not configured (501)', () => {
    const err: Q<never> = { isPending: false, isError: true, error: new ApiError(501, 'unimplemented') };
    ttslResult = err;
    supplyResult = err;
    poolResult = err;
    alertsResult = err;
    render(<RadarPage />);
    expect(screen.getAllByText(/Talent Radar needs the configured environment/i)).toHaveLength(4);
  });
});
