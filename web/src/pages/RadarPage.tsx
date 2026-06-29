import { Alert, Skeleton, Stack, Typography } from '@mui/material';

import { ApiError } from '../api/types';
import { PoolPanel } from '../components/radar/PoolPanel';
import { SupplyDemandPanel } from '../components/radar/SupplyDemandPanel';
import { TimeToShortlistHeadline } from '../components/radar/TimeToShortlistHeadline';
import { usePool, useSupplyDemand, useTimeToShortlist } from '../query/radar';

function unavailable(err: unknown): string {
  if (err instanceof ApiError && err.status === 501) {
    return 'Talent Radar needs the configured environment (database + seeded pool) to render.';
  }
  return err instanceof Error ? err.message : 'Could not load.';
}

export function RadarPage() {
  const ttsl = useTimeToShortlist();
  const supply = useSupplyDemand();
  const pool = usePool();

  return (
    <Stack spacing={4} sx={{ maxWidth: 900, mx: 'auto' }}>
      <Stack spacing={1}>
        <Typography variant="h3" component="h1">Talent Radar</Typography>
        <Typography color="text.secondary">The live god-view: pool, supply &amp; demand, and the headline metric.</Typography>
      </Stack>

      {ttsl.isPending ? (
        <Skeleton variant="rounded" height={160} />
      ) : ttsl.isError ? (
        <Alert severity="info">{unavailable(ttsl.error)}</Alert>
      ) : (
        ttsl.data && <TimeToShortlistHeadline metric={ttsl.data.metric} />
      )}

      {supply.isPending ? (
        <Skeleton variant="rounded" height={180} />
      ) : supply.isError ? (
        <Alert severity="info">{unavailable(supply.error)}</Alert>
      ) : (
        <SupplyDemandPanel items={supply.data?.items ?? []} />
      )}

      {pool.isPending ? (
        <Skeleton variant="rounded" height={220} />
      ) : pool.isError ? (
        <Alert severity="info">{unavailable(pool.error)}</Alert>
      ) : (
        <PoolPanel candidates={pool.data?.candidates ?? []} />
      )}
    </Stack>
  );
}
