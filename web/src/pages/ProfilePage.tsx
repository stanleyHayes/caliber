import { Alert, Box, Card, CardContent, Skeleton, Stack, TextField, Typography } from '@mui/material';
import { useState } from 'react';

import { ApiError } from '../api/types';
import { MyContestsList } from '../components/contest/MyContestsList';
import { DeleteAccount } from '../components/privacy/DeleteAccount';
import { DotsButton } from '../components/DotsButton';
import { ProfileView } from '../components/talent/ProfileView';
import { downloadTextFile } from '../lib/download';
import { useMyContests } from '../query/contest';
import { useExportMyData } from '../query/privacy';
import { useCreateProfile, useProfile } from '../query/talent';
import { useAuthStore } from '../stores/auth';

export function ProfilePage() {
  const candidateId = useAuthStore((s) => s.user?.id);
  const profile = useProfile(candidateId);
  const create = useCreateProfile(candidateId);
  const contests = useMyContests(Boolean(candidateId));
  const dataExport = useExportMyData();
  const [cv, setCv] = useState('');
  const [location, setLocation] = useState('');

  const submit = () => {
    if (cv.trim().length === 0) {
      return;
    }
    create.mutate({ cvText: cv, intake: { location, targetTitles: [], salaryFloor: 0 } });
  };

  const downloadData = () => {
    dataExport.mutate(undefined, {
      onSuccess: (data) => downloadTextFile('my-caliber-data.json', data.document),
    });
  };

  const existing = profile.data?.profile ?? create.data?.profile;
  const notFound = profile.error instanceof ApiError && profile.error.status === 404;

  return (
    <Stack spacing={4} sx={{ maxWidth: 760, mx: 'auto' }}>
      <Stack spacing={1}>
        <Typography variant="h3" component="h1">Talent Passport</Typography>
        <Typography color="text.secondary">
          Paste your CV. Caliber extracts an evidence-linked profile — every competency cites its source, and
          your job-search agent only ever uses what is here.
        </Typography>
      </Stack>

      {profile.isPending && candidateId && <Skeleton variant="rounded" height={180} />}
      {existing && <ProfileView profile={existing} />}

      {(notFound || existing) && (
        <Card variant="outlined">
          <CardContent>
            <Stack spacing={2}>
              <Typography variant="h6">{existing ? 'Update from a new CV' : 'Create your profile'}</Typography>
              {create.isError && (
                <Alert severity="error">{create.error instanceof Error ? create.error.message : 'Failed.'}</Alert>
              )}
              <TextField
                value={cv}
                onChange={(e) => setCv(e.target.value)}
                placeholder="Paste your CV text…"
                multiline
                minRows={5}
                fullWidth
              />
              <TextField label="Location" value={location} onChange={(e) => setLocation(e.target.value)} fullWidth />
              <Box>
                <DotsButton variant="contained" loading={create.isPending} disabled={cv.trim().length === 0} onClick={submit}>
                  {existing ? 'Re-extract profile' : 'Build my profile'}
                </DotsButton>
              </Box>
            </Stack>
          </CardContent>
        </Card>
      )}

      {(contests.data?.contests.length ?? 0) > 0 && (
        <Stack spacing={2}>
          <Typography variant="h6">Your disputes</Typography>
          <MyContestsList contests={contests.data?.contests ?? []} />
        </Stack>
      )}

      <Stack spacing={1.5} sx={{ alignItems: 'flex-start' }}>
        <Typography variant="h6">Your data</Typography>
        <Typography variant="body2" color="text.secondary">
          Download a complete copy of everything Caliber holds about you — your profile, applications,
          screenings, and disputes (Ghana Data Protection Act, right of access).
        </Typography>
        {dataExport.isError && (
          <Alert severity="error">
            {dataExport.error instanceof Error ? dataExport.error.message : 'Could not export your data.'}
          </Alert>
        )}
        <DotsButton variant="outlined" loading={dataExport.isPending} onClick={downloadData}>
          Download my data
        </DotsButton>
        <DeleteAccount />
      </Stack>
    </Stack>
  );
}
