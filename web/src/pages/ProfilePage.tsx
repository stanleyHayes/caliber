import { Alert, Box, Button, Card, CardContent, Skeleton, Stack, TextField, Typography } from '@mui/material';
import { useState } from 'react';

import { ApiError } from '../api/types';
import { MyContestsList } from '../components/contest/MyContestsList';
import { DeleteAccount } from '../components/privacy/DeleteAccount';
import { DotsButton } from '../components/DotsButton';
import { PageControls } from '../components/PageControls';
import { ProfileView } from '../components/talent/ProfileView';
import { downloadTextFile } from '../lib/download';
import { useMyContests } from '../query/contest';
import { useExportMyData } from '../query/privacy';
import { useCreateProfile, useProfile } from '../query/talent';
import { useAuthStore } from '../stores/auth';

const CONTESTS_PAGE_SIZE = 20;

function splitCommaList(value: string): string[] {
  return value.split(',').map((v) => v.trim()).filter(Boolean);
}

function splitLineList(value: string): string[] {
  return value.split(/\r?\n/).map((v) => v.trim()).filter(Boolean);
}

async function fileToBase64(file: File): Promise<string> {
  const bytes = new Uint8Array(await file.arrayBuffer());
  let binary = '';
  const chunkSize = 0x8000;
  for (let i = 0; i < bytes.length; i += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunkSize));
  }
  return btoa(binary);
}

export function ProfilePage() {
  const candidateId = useAuthStore((s) => s.user?.id);
  const profile = useProfile(candidateId);
  const create = useCreateProfile(candidateId);
  const [contestsPage, setContestsPage] = useState(1);
  const contests = useMyContests(Boolean(candidateId), contestsPage, CONTESTS_PAGE_SIZE);
  const dataExport = useExportMyData();
  const [cv, setCv] = useState('');
  const [cvFile, setCvFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState('');
  const [readingFile, setReadingFile] = useState(false);
  const [location, setLocation] = useState('');
  const [targetTitles, setTargetTitles] = useState('');
  const [salaryFloor, setSalaryFloor] = useState('');
  const [dealBreakers, setDealBreakers] = useState('');

  const submit = async () => {
    if (!cvFile && cv.trim().length === 0) {
      return;
    }
    setFileError('');
    setReadingFile(true);
    try {
      const cvFilePayload = cvFile ? await fileToBase64(cvFile) : undefined;
      create.mutate({
        cvText: cvFile ? '' : cv,
        cvFile: cvFilePayload,
        cvFilename: cvFile?.name,
        intake: {
          location,
          targetTitles: splitCommaList(targetTitles),
          salaryFloor: Number(salaryFloor) || 0,
          dealBreakers: splitLineList(dealBreakers),
        },
      });
    } catch {
      setFileError('Could not read that CV file.');
    } finally {
      setReadingFile(false);
    }
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
          Upload or paste your CV. Caliber extracts an evidence-linked profile — every competency cites its source, and
          your job-search agent only ever uses what is here.
        </Typography>
      </Stack>

      {profile.isPending && candidateId && <Skeleton variant="rounded" height={180} />}
      {existing && <ProfileView profile={existing} />}

      {(notFound || existing) && (
        <Card variant="outlined">
          <CardContent>
            <Stack spacing={2}>
              <Typography variant="h6" component="h2">{existing ? 'Update from a new CV' : 'Create your profile'}</Typography>
              {create.isError && (
                <Alert severity="error">{create.error instanceof Error ? create.error.message : 'Failed.'}</Alert>
              )}
              {fileError && <Alert severity="error">{fileError}</Alert>}
              <Stack direction="row" spacing={1.5} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
                <Button component="label" variant="outlined">
                  Upload CV file
                  <input
                    aria-label="Upload CV file"
                    hidden
                    type="file"
                    accept=".pdf,.docx,.txt,.text,.md,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain,text/markdown"
                    onChange={(e) => setCvFile(e.target.files?.[0] ?? null)}
                  />
                </Button>
                {cvFile && (
                  <Typography variant="body2" color="text.secondary">
                    {cvFile.name}
                  </Typography>
                )}
              </Stack>
              <TextField
                value={cv}
                onChange={(e) => setCv(e.target.value)}
                placeholder={cvFile ? 'File selected. You can leave pasted text empty.' : 'Paste your CV text...'}
                multiline
                minRows={5}
                fullWidth
              />
              <TextField label="Location" value={location} onChange={(e) => setLocation(e.target.value)} fullWidth />
              <TextField
                label="Target titles"
                value={targetTitles}
                onChange={(e) => setTargetTitles(e.target.value)}
                placeholder="Backend Engineer, Platform Engineer"
                fullWidth
              />
              <TextField
                label="Salary floor"
                type="number"
                value={salaryFloor}
                onChange={(e) => setSalaryFloor(e.target.value)}
                slotProps={{ htmlInput: { min: 0 } }}
                fullWidth
              />
              <TextField
                label="Deal-breakers"
                value={dealBreakers}
                onChange={(e) => setDealBreakers(e.target.value)}
                placeholder="One per line"
                multiline
                minRows={2}
                fullWidth
              />
              <Box>
                <DotsButton
                  variant="contained"
                  loading={create.isPending || readingFile}
                  disabled={!cvFile && cv.trim().length === 0}
                  onClick={submit}
                >
                  {existing ? 'Re-extract profile' : 'Build my profile'}
                </DotsButton>
              </Box>
            </Stack>
          </CardContent>
        </Card>
      )}

      {(contests.data?.contests.length ?? 0) > 0 && (
        <Stack spacing={2}>
          <Typography variant="h6" component="h2">Your disputes</Typography>
          <MyContestsList contests={contests.data?.contests ?? []} />
          {contests.data?.page && (
            <PageControls
              page={contests.data.page.page || contestsPage}
              pageCount={contests.data.page.totalPages}
              onChange={setContestsPage}
            />
          )}
        </Stack>
      )}

      <Stack spacing={1.5} sx={{ alignItems: 'flex-start' }}>
        <Typography variant="h6" component="h2">Your data</Typography>
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
