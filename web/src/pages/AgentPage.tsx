import { Alert, Box, Chip, Divider, Stack, Typography } from '@mui/material';
import { useState } from 'react';

import { ApiError } from '../api/types';
import { ApplicationsList } from '../components/agent/ApplicationsList';
import { CardListSkeleton } from '../components/Skeletons';
import { DotsButton } from '../components/DotsButton';
import { PageControls } from '../components/PageControls';
import { PageBackButton } from '../components/PageBackButton';
import { PermissionRedirect } from '../components/PermissionRedirect';
import { WakeUpCard } from '../components/agent/WakeUpCard';
import { canRunAgent } from '../lib/permissions';
import { useApplications, useTimeAdvance } from '../query/agent';
import { useAuthStore } from '../stores/auth';
import { fonts } from '../theme/tokens';

const APPLICATIONS_PAGE_SIZE = 20;

function errorMessage(err: unknown): string {
  if (err instanceof ApiError && err.status === 501) {
    return 'The agent needs the configured environment (database + your verified profile) to run.';
  }
  return err instanceof Error ? err.message : 'Something went wrong.';
}

export function AgentPage() {
  const user = useAuthStore((s) => s.user);
  const canUseAgent = canRunAgent(user?.role);
  const candidateId = canUseAgent ? user?.id : undefined;
  const [applicationsPage, setApplicationsPage] = useState(1);
  const advance = useTimeAdvance(candidateId);
  const applications = useApplications(candidateId, applicationsPage, APPLICATIONS_PAGE_SIZE);
  const applicationTotal = applications.data?.page?.totalItems ?? applications.data?.applications.length ?? 0;

  if (!canUseAgent) {
    return <PermissionRedirect />;
  }

  return (
    <Stack spacing={4} sx={{ maxWidth: 760, mx: 'auto' }}>
      <PageBackButton />
      <Stack spacing={1}>
        <Typography variant="h3" component="h1">Your job-search agent</Typography>
        <Typography color="text.secondary">
          It works while you sleep — honestly. It only applies where your verified profile already qualifies you,
          and every application draws on your real evidence.
        </Typography>
      </Stack>

      <Box>
        <DotsButton variant="contained" size="large" loading={advance.isPending} onClick={() => advance.mutate()}>
          Run overnight
        </DotsButton>
      </Box>

      {advance.isError && <Alert severity="info">{errorMessage(advance.error)}</Alert>}
      {advance.data && <WakeUpCard wakeUp={advance.data.wakeUp} />}

      <Divider />

      <Stack spacing={2.25}>
        <Stack
          direction={{ xs: 'column', sm: 'row' }}
          spacing={1.5}
          useFlexGap
          sx={{ alignItems: { xs: 'flex-start', sm: 'center' }, justifyContent: 'space-between' }}
        >
          <Box sx={{ minWidth: 0 }}>
            <Typography
              variant="h5"
              component="h2"
              sx={{ fontFamily: fonts.body, fontWeight: 850, letterSpacing: 0, lineHeight: 1.12 }}
            >
              Applications
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
              Submissions and drafts prepared from your verified profile.
            </Typography>
          </Box>
          <Chip
            size="small"
            label={`${applicationTotal} tracked`}
            variant="outlined"
            sx={{ borderRadius: '999px', fontFamily: fonts.body, fontWeight: 800, height: 30 }}
          />
        </Stack>
        {applications.isPending && candidateId ? (
          <CardListSkeleton count={2} />
        ) : applications.isError ? (
          <Alert severity="info">{errorMessage(applications.error)}</Alert>
        ) : (
          <>
            <ApplicationsList applications={applications.data?.applications ?? []} />
            {applications.data?.page && (
              <PageControls
                page={applications.data.page.page || applicationsPage}
                pageCount={applications.data.page.totalPages}
                onChange={setApplicationsPage}
              />
            )}
          </>
        )}
      </Stack>
    </Stack>
  );
}
