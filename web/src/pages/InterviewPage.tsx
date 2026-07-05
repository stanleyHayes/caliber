import {
  Alert,
  Button,
  Box,
  Card,
  CardActionArea,
  CardContent,
  Chip,
  Collapse,
  Divider,
  Skeleton,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import ArrowForwardRounded from '@mui/icons-material/ArrowForwardRounded';
import { useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { ApiError } from '../api/types';
import { ContestAssessment } from '../components/contest/ContestAssessment';
import { DotsButton } from '../components/DotsButton';
import { PageBackButton } from '../components/PageBackButton';
import { PageControls } from '../components/PageControls';
import { PermissionRedirect } from '../components/PermissionRedirect';
import { QuestionPanel } from '../components/interview/QuestionPanel';
import { ReportCardView } from '../components/interview/ReportCardView';
import { TranscriptList } from '../components/interview/TranscriptList';
import { useInterview } from '../hooks/useInterview';
import { roleStatusLabel, seniorityLabel } from '../lib/format';
import { canScreenSelf } from '../lib/permissions';
import { useOpenRoles } from '../query/flow';
import { useAuthStore } from '../stores/auth';
import { fonts } from '../theme/tokens';

const ROLE_PAGE_SIZE = 8;

export function InterviewPage() {
  const [params] = useSearchParams();
  const user = useAuthStore((s) => s.user);
  const interview = useInterview();
  const [rolesPage, setRolesPage] = useState(1);
  const [roleId, setRoleId] = useState(params.get('roleId') ?? '');
  const [showRoleIdEntry, setShowRoleIdEntry] = useState(Boolean(params.get('roleId')));
  const canUseInterview = canScreenSelf(user?.role);

  const roles = useOpenRoles(rolesPage, ROLE_PAGE_SIZE, canUseInterview);
  const roleList = roles.data?.roles ?? [];
  const selectedRole = roleList.find((r) => r.id === roleId);
  const roleListDenied = roles.error instanceof ApiError && roles.error.status === 403;

  const begin = (nextRoleId = roleId) => {
    const id = nextRoleId.trim();
    if (id.length === 0) {
      return;
    }
    setRoleId(id);
    interview.start(id, user?.id ?? 'demo-candidate');
  };

  if (!canUseInterview || roleListDenied) {
    return <PermissionRedirect />;
  }

  return (
    <Stack spacing={4} sx={{ maxWidth: 760, mx: 'auto' }}>
      <PageBackButton />
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
            <Stack spacing={2.5}>
              <Stack spacing={0.25}>
                <Typography variant="overline" color="text.secondary">Choose a role</Typography>
                <Typography variant="body2" color="text.secondary">
                  Pick a role below. The interview starts against that role's rubric.
                </Typography>
              </Stack>

              {roles.isPending ? (
                <Stack spacing={1}>
                  <Skeleton variant="rounded" height={88} />
                  <Skeleton variant="rounded" height={88} />
                  <Skeleton variant="rounded" height={88} />
                </Stack>
              ) : roles.isError ? (
                <Alert severity="info">
                  {roles.error instanceof Error ? roles.error.message : 'Could not load interview roles.'}
                </Alert>
              ) : roleList.length > 0 ? (
                <Stack spacing={1.25} role="list" aria-label="Interview roles">
                  {roleList.map((r) => {
                    const title = r.title || r.spec.title;
                    const competencyCount = r.rubric.competencies.length;
                    const selected = selectedRole?.id === r.id;

                    return (
                      <Card
                        key={r.id}
                        variant="outlined"
                        sx={{
                          borderRadius: '8px',
                          borderColor: selected ? 'primary.main' : 'divider',
                          boxShadow: selected ? (t) => `inset 0 0 0 1px ${t.palette.primary.main}` : 'none',
                          bgcolor: selected ? 'action.selected' : 'background.paper',
                          overflow: 'hidden',
                        }}
                      >
                        <CardActionArea
                          aria-label={`Start interview for ${title}`}
                          onClick={() => begin(r.id)}
                          sx={{
                            p: { xs: 1.75, sm: 2 },
                            textAlign: 'left',
                            transition: 'background-color 160ms ease',
                          }}
                        >
                          <Stack
                            direction={{ xs: 'column', sm: 'row' }}
                            spacing={1.5}
                            sx={{ alignItems: { xs: 'stretch', sm: 'center' } }}
                          >
                            <Stack spacing={1} sx={{ flexGrow: 1, minWidth: 0 }}>
                              <Stack direction="row" spacing={1} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
                                <Chip
                                  size="small"
                                  label={roleStatusLabel[r.status]}
                                  color={r.status === 'ROLE_STATUS_OPEN' ? 'primary' : 'default'}
                                  variant={r.status === 'ROLE_STATUS_OPEN' ? 'filled' : 'outlined'}
                                  sx={{ borderRadius: '999px', fontFamily: fonts.body, fontWeight: 800 }}
                                />
                                <Chip size="small" label={seniorityLabel[r.spec.seniority]} sx={{ borderRadius: '999px', fontFamily: fonts.body }} />
                                {r.spec.location && (
                                  <Chip
                                    size="small"
                                    variant="outlined"
                                    label={r.spec.location}
                                    sx={{ borderRadius: '999px', fontFamily: fonts.body }}
                                  />
                                )}
                                {r.spec.availability && (
                                  <Chip
                                    size="small"
                                    variant="outlined"
                                    label={r.spec.availability}
                                    sx={{ borderRadius: '999px', fontFamily: fonts.body }}
                                  />
                                )}
                                <Chip
                                  size="small"
                                  variant="outlined"
                                  label={`${competencyCount} ${competencyCount === 1 ? 'competency' : 'competencies'}`}
                                  sx={{ borderRadius: '999px', fontFamily: fonts.body }}
                                />
                              </Stack>
                              <Typography sx={{ fontWeight: 800, fontSize: 20, lineHeight: 1.2 }} noWrap>
                                {title}
                              </Typography>
                              {r.spec.mustHaves.length > 0 && (
                                <Typography variant="body2" color="text.secondary" noWrap>
                                  {r.spec.mustHaves.slice(0, 3).join(' · ')}
                                </Typography>
                              )}
                            </Stack>

                            <Stack
                              direction="row"
                              spacing={0.75}
                              sx={{
                                alignItems: 'center',
                                justifyContent: { xs: 'flex-start', sm: 'flex-end' },
                                color: 'primary.main',
                                fontWeight: 800,
                                whiteSpace: 'nowrap',
                              }}
                            >
                              <Typography component="span" sx={{ fontWeight: 800 }}>
                                Start interview
                              </Typography>
                              <ArrowForwardRounded fontSize="small" />
                            </Stack>
                          </Stack>
                        </CardActionArea>
                      </Card>
                    );
                  })}
                  {roles.data?.page && (
                    <PageControls
                      page={roles.data.page.page || rolesPage}
                      pageCount={roles.data.page.totalPages}
                      onChange={setRolesPage}
                    />
                  )}
                </Stack>
              ) : (
                <Typography variant="body2" color="text.secondary">
                  No interview roles are available yet.
                </Typography>
              )}

              <Divider />

              <Button
                variant="text"
                onClick={() => setShowRoleIdEntry((current) => !current)}
                sx={{ alignSelf: 'flex-start', px: 0, fontWeight: 800 }}
              >
                {showRoleIdEntry ? 'Hide role ID field' : 'Use a role ID instead'}
              </Button>

              <Collapse in={showRoleIdEntry} unmountOnExit>
                <Stack spacing={1.5} sx={{ pt: 0.5, alignItems: 'flex-start' }}>
                  <TextField
                    label="Role ID"
                    value={roleId}
                    onChange={(e) => setRoleId(e.target.value)}
                    placeholder="Paste a role id"
                    size="small"
                    fullWidth
                  />
                  <DotsButton
                    variant="outlined"
                    size="large"
                    onClick={() => begin()}
                    disabled={roleId.trim().length === 0}
                  >
                    Start interview
                  </DotsButton>
                </Stack>
              </Collapse>
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
