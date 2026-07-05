import { useState } from 'react';
import FactCheckOutlined from '@mui/icons-material/FactCheckOutlined';
import PersonSearchOutlined from '@mui/icons-material/PersonSearchOutlined';
import RadarRounded from '@mui/icons-material/RadarRounded';
import WorkOutlineRounded from '@mui/icons-material/WorkOutlineRounded';
import { Alert, Box, Chip, Stack, Typography } from '@mui/material';

import { ApiError } from '../api/types';
import { CardSkeleton } from '../components/Skeletons';
import { PageBackButton } from '../components/PageBackButton';
import { PermissionRedirect } from '../components/PermissionRedirect';
import { AlertsPanel } from '../components/radar/AlertsPanel';
import { PoolPanel } from '../components/radar/PoolPanel';
import { SupplyDemandPanel } from '../components/radar/SupplyDemandPanel';
import { TimeToShortlistHeadline } from '../components/radar/TimeToShortlistHeadline';
import { canViewDashboard } from '../lib/permissions';
import { useAlerts, usePool, useSupplyDemand, useTimeToShortlist } from '../query/radar';
import { useAuthStore } from '../stores/auth';
import { fonts } from '../theme/tokens';

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
  const userRole = useAuthStore((s) => s.user?.role);
  const canUseRadar = canViewDashboard(userRole);

  const ttsl = useTimeToShortlist(canUseRadar);
  const supply = useSupplyDemand(canUseRadar);
  const pool = usePool(poolPage, pageSize, canUseRadar);
  const alerts = useAlerts(alertsPage, pageSize, canUseRadar);
  const totalCandidates = pool.data?.page?.totalItems ?? pool.data?.candidates.length ?? 0;
  const totalAlerts = alerts.data?.page?.totalItems ?? alerts.data?.alerts.length ?? 0;
  const roleFamilies = supply.data?.items.length ?? 0;

  if (!canUseRadar) {
    return <PermissionRedirect />;
  }

  return (
    <Stack spacing={3} sx={{ maxWidth: 1180, mx: 'auto', fontFamily: fonts.body }}>
      <PageBackButton />
      <Box
        sx={{
          position: 'relative',
          overflow: 'hidden',
          borderRadius: '8px',
          border: '1px solid',
          borderColor: 'divider',
          bgcolor: 'background.paper',
          p: { xs: 2.5, md: 3.5 },
          boxShadow: '0 24px 80px rgba(0, 0, 0, 0.20)',
          '&::before': {
            content: '""',
            position: 'absolute',
            inset: 0,
            background:
              'linear-gradient(135deg, rgba(77, 155, 255, 0.18), transparent 42%), linear-gradient(180deg, rgba(45, 212, 191, 0.1), transparent 64%)',
            pointerEvents: 'none',
          },
        }}
      >
        <Box
          sx={{
            position: 'relative',
            display: 'grid',
            gap: { xs: 2.5, md: 3 },
            gridTemplateColumns: { xs: '1fr', md: 'minmax(0, 1fr) minmax(300px, 420px)' },
            alignItems: 'end',
          }}
        >
          <Stack spacing={2.25}>
            <Stack direction="row" spacing={1} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
              <Chip
                icon={<RadarRounded />}
                label="Live radar"
                color="primary"
                size="small"
                sx={{ borderRadius: '999px', fontFamily: fonts.body, fontWeight: 800 }}
              />
              <Chip
                icon={<FactCheckOutlined />}
                label="Evidence-linked signals"
                variant="outlined"
                size="small"
                sx={{ borderRadius: '999px', fontFamily: fonts.body, fontWeight: 700 }}
              />
            </Stack>

            <Box>
              <Typography
                component="h1"
                sx={{
                  fontFamily: fonts.body,
                  fontSize: { xs: 42, sm: 54, md: 68 },
                  fontWeight: 850,
                  letterSpacing: 0,
                  lineHeight: 0.96,
                }}
              >
                Talent Radar
              </Typography>
              <Typography sx={{ mt: 1.5, maxWidth: 700, color: 'text.secondary', fontSize: { xs: 16, md: 18 }, lineHeight: 1.55 }}>
                A command view for verified talent, role pressure, shortlist velocity, and explainable match movement.
              </Typography>
            </Box>
          </Stack>

          <Box
            sx={{
              display: 'grid',
              gap: 1,
              gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, minmax(0, 1fr))', md: '1fr' },
            }}
          >
            {[
              { label: 'Talent pool', value: totalCandidates, Icon: PersonSearchOutlined },
              { label: 'Role families', value: roleFamilies, Icon: WorkOutlineRounded },
              { label: 'Alerts', value: totalAlerts, Icon: FactCheckOutlined },
            ].map(({ label, value, Icon }) => (
              <Box
                key={label}
                sx={{
                  display: 'grid',
                  gridTemplateColumns: '40px 1fr auto',
                  gap: 1.25,
                  alignItems: 'center',
                  borderRadius: '8px',
                  border: '1px solid',
                  borderColor: 'divider',
                  bgcolor: 'rgba(255, 255, 255, 0.04)',
                  p: 1.25,
                }}
              >
                <Box
                  sx={{
                    width: 40,
                    height: 40,
                    borderRadius: '8px',
                    display: 'grid',
                    placeItems: 'center',
                    color: 'primary.main',
                    bgcolor: 'rgba(77, 155, 255, 0.14)',
                  }}
                >
                  <Icon fontSize="small" />
                </Box>
                <Typography sx={{ color: 'text.secondary', fontSize: 13, fontWeight: 800 }}>{label}</Typography>
                <Typography sx={{ fontFamily: fonts.mono, fontSize: 18, fontWeight: 800 }}>{value}</Typography>
              </Box>
            ))}
          </Box>
        </Box>
      </Box>

      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', lg: 'minmax(320px, 0.82fr) minmax(0, 1.18fr)' } }}>
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
      </Box>

      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', xl: 'minmax(360px, 0.82fr) minmax(0, 1.18fr)' } }}>
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
      </Box>
    </Stack>
  );
}
