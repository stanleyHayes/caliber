import { Alert, Skeleton, Stack, Typography } from '@mui/material';
import { useParams } from 'react-router-dom';

import { PageBackButton } from '../components/PageBackButton';
import { ProfileView } from '../components/talent/ProfileView';
import { shortId } from '../lib/format';
import { useProfile } from '../query/talent';

// Read-only view of another candidate's evidence-linked Talent Passport, reached
// by clicking a candidate in a shortlist. Employers/recruiters are authorized to
// view it (GetTalentProfile → requireSelfCandidateOrReviewer).
export function CandidateProfilePage() {
  const { candidateId } = useParams<{ candidateId: string }>();
  const profile = useProfile(candidateId);
  const data = profile.data?.profile;

  return (
    <Stack spacing={3} sx={{ maxWidth: 820, mx: 'auto' }}>
      <PageBackButton />
      <Stack spacing={1}>
        <Typography variant="h3" component="h1">Candidate profile</Typography>
        <Typography color="text.secondary">
          Evidence-linked Talent Passport{candidateId ? ` · ${shortId(candidateId)}` : ''} — every competency cites its source.
        </Typography>
      </Stack>

      {profile.isPending && candidateId && <Skeleton variant="rounded" height={220} />}
      {profile.isError && (
        <Alert severity="info">
          {profile.error instanceof Error ? profile.error.message : 'This candidate has no profile on file yet.'}
        </Alert>
      )}
      {data && <ProfileView profile={data} />}
    </Stack>
  );
}
