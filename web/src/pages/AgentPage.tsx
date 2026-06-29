import { Alert, Box, Divider, Stack, Typography } from '@mui/material';

import { ApiError } from '../api/types';
import { ApplicationsList } from '../components/agent/ApplicationsList';
import { CardListSkeleton } from '../components/Skeletons';
import { DotsButton } from '../components/DotsButton';
import { WakeUpCard } from '../components/agent/WakeUpCard';
import { useApplications, useTimeAdvance } from '../query/agent';
import { useAuthStore } from '../stores/auth';

function errorMessage(err: unknown): string {
  if (err instanceof ApiError && err.status === 501) {
    return 'The agent needs the configured environment (database + your verified profile) to run.';
  }
  return err instanceof Error ? err.message : 'Something went wrong.';
}

export function AgentPage() {
  const candidateId = useAuthStore((s) => s.user?.id);
  const advance = useTimeAdvance(candidateId);
  const applications = useApplications(candidateId);

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
        <Typography variant="h6">Applications</Typography>
        {applications.isPending && candidateId ? (
          <CardListSkeleton count={2} />
        ) : applications.isError ? (
          <Alert severity="info">{errorMessage(applications.error)}</Alert>
        ) : (
          <ApplicationsList applications={applications.data?.applications ?? []} />
        )}
      </Stack>
    </Stack>
  );
}
