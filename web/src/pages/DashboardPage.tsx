import { Box, Card, CardContent, Chip, Skeleton, Stack, Typography } from '@mui/material';

import type { UserRole } from '../api/types';
import { useMe } from '../query/auth';
import { useAuthStore } from '../stores/auth';

const ROLE_LABEL: Record<UserRole, string> = {
  USER_ROLE_UNSPECIFIED: 'Member',
  USER_ROLE_EMPLOYER: 'Employer',
  USER_ROLE_RECRUITER: 'Recruiter',
  USER_ROLE_CANDIDATE: 'Candidate',
};

const NEXT_BY_ROLE: Record<UserRole, string> = {
  USER_ROLE_UNSPECIFIED: 'Your workspace is being set up.',
  USER_ROLE_EMPLOYER: 'Describe a role in plain language to generate an explainable shortlist.',
  USER_ROLE_RECRUITER: 'Describe a role in plain language to generate an explainable shortlist.',
  USER_ROLE_CANDIDATE: 'Complete your Talent Passport to get matched to roles.',
};

export function DashboardPage() {
  const user = useAuthStore((s) => s.user);
  const me = useMe();

  if (!user && me.isPending) {
    return (
      <Stack spacing={2} sx={{ maxWidth: 720 }}>
        <Skeleton width="40%" height={44} />
        <Skeleton variant="rounded" height={140} />
      </Stack>
    );
  }

  const role = user?.role ?? 'USER_ROLE_UNSPECIFIED';
  return (
    <Stack spacing={3} sx={{ maxWidth: 720 }}>
      <Box>
        <Chip label={ROLE_LABEL[role]} color="primary" size="small" sx={{ mb: 1 }} />
        <Typography variant="h3">Welcome{user ? `, ${user.name}` : ''}.</Typography>
      </Box>
      <Card variant="outlined">
        <CardContent>
          <Typography variant="overline" color="text.secondary">
            Next step
          </Typography>
          <Typography variant="h6" sx={{ mt: 0.5 }}>
            {NEXT_BY_ROLE[role]}
          </Typography>
        </CardContent>
      </Card>
    </Stack>
  );
}
