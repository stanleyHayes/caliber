import { useState } from 'react';
import { Alert, Stack, Typography } from '@mui/material';

import { ApiError } from '../api/types';
import { CardSkeleton } from '../components/Skeletons';
import { AlertsPanel } from '../components/radar/AlertsPanel';
import { PoolPanel } from '../components/radar/PoolPanel';
import { SupplyDemandPanel } from '../components/radar/SupplyDemandPanel';
import { TimeToShortlistHeadline } from '../components/radar/TimeToShortlistHeadline';
import { useAlerts, usePool, useSupplyDemand, useTimeToShortlist } from '../query/radar';

const pageSize = 20;

function unavailable(err: unknown): string {
  if (err instanceof ApiError && err.status === 501) {
    return 'Talent Radar needs the configured environment (database + seeded pool) to render.';
  }
  return err instanceof Error ? err.message : 'Could not load.';
}

export function RadarPage() {
  const [poolPage, setPoolPage] = useState(1);
  const [alertsPage, setAlertsPage] = useState(1);

  const ttsl = useTimeToShortlist();
  const supply = useSupplyDemand();
  const pool = usePool(poolPage, pageSize);
  const alerts = useAlerts(alertsPage, pageSize);

  return (
    <Stack spacing={4} sx={{ maxWidth: 900, mx: 'auto' }}>
      <Stack spacing={1}>
        <Typography variant="h3" component="h1">Talent Radar</Typography>
        <Typography color="text.secondary">The live god-view: pool, supply &amp; demand, alerts, and the headline metric.</Typography>
      </Stack>

      {ttsl.isPending ? (
        <CardSkeleton lines={3} />
      ) : ttsl.isError ? (
        <Alert severity="info">{unavailable(ttsl.error)}</Alert>
      ) : (
        ttsl.data && <TimeToShortlistHeadline metric={ttsl.data.metric} />
      )}

      {supply.isPending ? (
        <CardSkeleton lines={4} />
      ) : supply.isError ? (
        <Alert severity="info">{unavailable(supply.error)}</Alert>
      ) : (
        <SupplyDemandPanel items={supply.data?.items ?? []} />
      )}

      {pool.isPending ? (
        <CardSkeleton lines={5} />
      ) : pool.isError ? (
        <Alert severity="info">{unavailable(pool.error)}</Alert>
      ) : (
        <PoolPanel
          candidates={pool.data?.candidates ?? []}
          page={poolPage}
          pageCount={pool.data?.page?.totalPages ?? 1}
          onPageChange={setPoolPage}
        />
      )}

      {alerts.isPending ? (
        <CardSkeleton lines={4} />
      ) : alerts.isError ? (
        <Alert severity="info">{unavailable(alerts.error)}</Alert>
      ) : (
        <AlertsPanel
          alerts={alerts.data?.alerts ?? []}
          page={alertsPage}
          pageCount={alerts.data?.page?.totalPages ?? 1}
          onPageChange={setAlertsPage}
        />
      )}
    </Stack>
  );
}
