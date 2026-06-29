import { Alert, Box, Card, CardContent, Skeleton, Stack, TextField, Typography } from '@mui/material';
import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { ContestAssessment } from '../components/contest/ContestAssessment';
import { DotsButton } from '../components/DotsButton';
import { QuestionPanel } from '../components/interview/QuestionPanel';
import { ReportCardView } from '../components/interview/ReportCardView';
import { TranscriptList } from '../components/interview/TranscriptList';
import { useInterview } from '../hooks/useInterview';
import { useAuthStore } from '../stores/auth';

export function InterviewPage() {
  const [params] = useSearchParams();
  const user = useAuthStore((s) => s.user);
  const interview = useInterview();
  const [roleId, setRoleId] = useState(params.get('roleId') ?? '');

  const begin = () => {
    if (roleId.trim().length === 0) {
      return;
    }
    interview.start(roleId.trim(), user?.id ?? 'demo-candidate');
  };

  return (
    <Stack spacing={4} sx={{ maxWidth: 760, mx: 'auto' }}>
      <Stack spacing={1}>
        <Typography variant="h3" component="h1">Screening interview</Typography>
        <Typography color="text.secondary">
          An adaptive AI interviewer probes each rubric competency, then scores your answers with
          evidence — no black box.
        </Typography>
      </Stack>

      {interview.status === 'idle' && (
        <Card variant="outlined">
          <CardContent>
            <Stack spacing={2}>
              <TextField
                label="Role ID"
                value={roleId}
                onChange={(e) => setRoleId(e.target.value)}
                placeholder="Paste a role id (generate one in Describe a role)"
                fullWidth
              />
              <DotsButton variant="contained" size="large" onClick={begin} disabled={roleId.trim().length === 0} sx={{ alignSelf: 'flex-start' }}>
                Start interview
              </DotsButton>
            </Stack>
          </CardContent>
        </Card>
      )}

      {interview.error && (
        <Alert
          severity="error"
          action={
            <DotsButton color="inherit" onClick={interview.reset}>
              Start over
            </DotsButton>
          }
        >
          {interview.error}
        </Alert>
      )}

      {interview.status === 'connecting' && (
        <Stack spacing={2}>
          <Skeleton variant="rounded" height={56} />
          <Skeleton variant="rounded" height={140} />
        </Stack>
      )}

      {interview.turns.length > 0 && <TranscriptList turns={interview.turns} />}

      {interview.status === 'asking' && interview.question && (
        <QuestionPanel question={interview.question} onAnswer={interview.answer} />
      )}

      {interview.status === 'submitting' && (
        <Box>
          <Typography variant="caption" color="text.secondary">
            Thinking about your next question…
          </Typography>
          <Skeleton variant="rounded" height={120} sx={{ mt: 1 }} />
        </Box>
      )}

      {interview.status === 'done' && interview.report && (
        <Stack spacing={2}>
          <ReportCardView report={interview.report} />
          <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap', alignItems: 'center' }}>
            <DotsButton variant="outlined" onClick={interview.reset}>
              Run another interview
            </DotsButton>
            <ContestAssessment subject="CONTEST_SUBJECT_REPORT_CARD" subjectId={interview.report.interviewId} />
          </Stack>
        </Stack>
      )}
    </Stack>
  );
}
