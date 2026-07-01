import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { flowApi } from '../api/flow';
import type { GenerateRoleResponse, ListRolesResponse, RecordRejectionResponse, Role, ShortlistResponse } from '../api/types';
import {
  useGenerateRole,
  useRecordRejection,
  useRoles,
  useShortlist,
  useUpdateRole,
} from './flow';

const role: Role = {
  id: 'r1',
  employerId: 'e1',
  title: 'Engineer',
  status: 'ROLE_STATUS_OPEN',
  createdAt: '2026-01-01T00:00:00Z',
  spec: {
    title: 'Engineer',
    location: 'Accra',
    seniority: 'SENIORITY_MID',
    availability: 'Full-time',
    responsibilities: [],
    mustHaves: ['Go'],
    niceToHaves: [],
    salaryBand: { currency: 'GHS', low: 0, high: 0 },
  },
  rubric: { competencies: [{ name: 'Go', weight: 1, mustHave: true }] },
};

vi.mock('../api/flow', () => ({
  flowApi: {
    recordRejection: vi.fn(),
    generateRole: vi.fn(),
    listRoles: vi.fn(),
    updateRole: vi.fn(),
    shortlist: vi.fn(),
  },
}));

function createWrapper() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

describe('useGenerateRole', () => {
  beforeEach(() => vi.clearAllMocks());

  it('generates a role from free text', async () => {
    const generated: GenerateRoleResponse = { role, availableMatches: 5 };
    vi.mocked(flowApi.generateRole).mockResolvedValue(generated);

    const { result } = renderHook(() => useGenerateRole(), { wrapper: createWrapper() });
    result.current.mutate({ employerId: 'e1', freeText: 'Go engineer' });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(generated);
  });
});

describe('useRecordRejection', () => {
  beforeEach(() => vi.clearAllMocks());

  it('records a human-approved rejection', async () => {
    const response: RecordRejectionResponse = { auditEntryId: 'audit-1' };
    vi.mocked(flowApi.recordRejection).mockResolvedValue(response);

    const { result } = renderHook(() => useRecordRejection(), { wrapper: createWrapper() });
    result.current.mutate({ roleId: 'r1', candidateId: 'c1', reason: 'not a fit', humanApproved: true });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(flowApi.recordRejection).toHaveBeenCalledWith('r1', 'c1', 'not a fit', true);
    expect(result.current.data).toEqual(response);
  });
});

describe('useRoles', () => {
  beforeEach(() => vi.clearAllMocks());

  it('lists roles for the employer', async () => {
    const response: ListRolesResponse = { roles: [role] };
    vi.mocked(flowApi.listRoles).mockResolvedValue(response);

    const { result } = renderHook(() => useRoles('e1'), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(response);
  });

  it('stays disabled when the employer id is missing', () => {
    const { result } = renderHook(() => useRoles(undefined), { wrapper: createWrapper() });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.fetchStatus).toBe('idle');
  });
});

describe('useUpdateRole', () => {
  beforeEach(() => vi.clearAllMocks());

  it('updates the role spec and rubric', async () => {
    vi.mocked(flowApi.updateRole).mockResolvedValue({ role });

    const { result } = renderHook(() => useUpdateRole(), { wrapper: createWrapper() });
    result.current.mutate({ roleId: 'r1', spec: role.spec, rubric: role.rubric });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(flowApi.updateRole).toHaveBeenCalledWith('r1', role.spec, role.rubric);
  });
});

describe('useShortlist', () => {
  beforeEach(() => vi.clearAllMocks());

  it('fetches a ranked shortlist when enabled', async () => {
    const response: ShortlistResponse = {
      shortlist: { matches: [], poolDepth: 10, exclusions: [] },
    };
    vi.mocked(flowApi.shortlist).mockResolvedValue(response);

    const { result } = renderHook(() => useShortlist('r1', 10, true, 0), { wrapper: createWrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual(response);
  });

  it('stays disabled when the role id is missing or enabled is false', () => {
    const { result } = renderHook(() => useShortlist(undefined, 10, true, 0), { wrapper: createWrapper() });
    expect(result.current.isLoading).toBe(false);
    expect(result.current.fetchStatus).toBe('idle');
  });
});
