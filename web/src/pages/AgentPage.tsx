import { Alert, Box, Divider, Stack, Typography } from '@mui/material';
import { useState } from 'react';

import { ApiError } from '../api/types';
import { ApplicationsList } from '../components/agent/ApplicationsList';
import { CardListSkeleton } from '../components/Skeletons';
import { DotsButton } from '../components/DotsButton';
import { PageControls } from '../components/PageControls';
import { WakeUpCard } from '../components/agent/WakeUpCard';
import { useApplications, useTimeAdvance } from '../query/agent';
import { useAuthStore } from '../stores/auth';

const APPLICATIONS_PAGE_SIZE = 20;

function errorMessage(err: unknown): string {
  if (err instanceof ApiError && err.status === 501) {
    return 'The agent needs the configured environment (database + your verified profile) to run.';
  }
  return err instanceof Error ? err.message : 'Something went wrong.';
}

export function AgentPage() {
  const candidateId = useAuthStore((s) => s.user?.id);
  const [applicationsPage, setApplicationsPage] = useState(1);
  const advance = useTimeAdvance(candidateId);
  const applications = useApplications(candidateId, applicationsPage, APPLICATIONS_PAGE_SIZE);

  return (
    <Stack spacing={4} sx={{ maxWidth: 760, mx: 'auto' }}>
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

      <Stack spacing={2}>
        <Typography variant="h6" component="h2">Applications</Typography>
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
