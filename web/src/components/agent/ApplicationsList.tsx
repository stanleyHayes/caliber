import AutoAwesomeRounded from '@mui/icons-material/AutoAwesomeRounded';
import CheckCircleRounded from '@mui/icons-material/CheckCircleRounded';
import DraftsRounded from '@mui/icons-material/DraftsRounded';
import HourglassTopRounded from '@mui/icons-material/HourglassTopRounded';
import SendRounded from '@mui/icons-material/SendRounded';
import WorkOutlineRounded from '@mui/icons-material/WorkOutlineRounded';
import { Box, Card, CardContent, Chip, LinearProgress, Stack, Typography } from '@mui/material';

import type { Application, ApplicationStatus } from '../../api/types';
import { applicationStatusColor, applicationStatusLabel, shortId } from '../../lib/format';
import { fonts } from '../../theme/tokens';

const statusMeta: Record<ApplicationStatus, { progress: number; detail: string; Icon: typeof DraftsRounded }> = {
  APPLICATION_STATUS_UNSPECIFIED: { progress: 0, detail: 'Status pending', Icon: WorkOutlineRounded },
  APPLICATION_STATUS_DRAFTED: { progress: 25, detail: 'Draft ready', Icon: DraftsRounded },
  APPLICATION_STATUS_SUBMITTED: { progress: 55, detail: 'Submitted to employer', Icon: SendRounded },
  APPLICATION_STATUS_SCREENING: { progress: 78, detail: 'Screening in progress', Icon: HourglassTopRounded },
  APPLICATION_STATUS_SCREENED: { progress: 100, detail: 'Screening complete', Icon: CheckCircleRounded },
};

const fallbackStatus: ApplicationStatus = 'APPLICATION_STATUS_UNSPECIFIED';

function getStatusMeta(status: Application['status']) {
  return statusMeta[status] ?? statusMeta[fallbackStatus];
}

function getStatusLabel(status: Application['status']) {
  return applicationStatusLabel[status] ?? applicationStatusLabel[fallbackStatus];
}

function getStatusColor(status: Application['status']) {
  return applicationStatusColor[status] ?? applicationStatusColor[fallbackStatus];
}

function sourceLabel(source: Application['source']) {
  switch (source) {
    case 'APPLICATION_SOURCE_AGENT':
      return 'by your agent';
    case 'APPLICATION_SOURCE_MANUAL':
      return 'manual';
    default:
      return 'source pending';
  }
}

const chipSx = {
  borderRadius: '999px',
  fontFamily: fonts.body,
  fontWeight: 750,
  height: 30,
  '& .MuiChip-label': { px: 1.25 },
};

export function ApplicationsList({ applications }: { applications: Application[] }) {
  if (applications.length === 0) {
    return (
      <Box
        role="status"
        sx={{
          display: 'grid',
          gridTemplateColumns: { xs: '1fr', sm: 'auto 1fr' },
          gap: 1.5,
          alignItems: 'center',
          p: { xs: 2, sm: 2.25 },
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: '8px',
          bgcolor: 'background.paper',
        }}
      >
        <Box
          sx={{
            display: 'grid',
            placeItems: 'center',
            width: 44,
            height: 44,
            borderRadius: '8px',
            bgcolor: 'rgba(0, 102, 204, 0.1)',
            color: 'primary.main',
          }}
        >
          <WorkOutlineRounded aria-hidden="true" />
        </Box>
        <Box sx={{ minWidth: 0 }}>
          <Typography sx={{ fontWeight: 850, lineHeight: 1.2 }}>No applications yet</Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.4 }}>
            Run your agent to prepare the first one.
          </Typography>
        </Box>
      </Box>
    );
  }
  return (
    <Stack spacing={2}>
      {applications.map((a) => {
        const meta = getStatusMeta(a.status);
        const Icon = meta.Icon;
        const isAgent = a.source === 'APPLICATION_SOURCE_AGENT';

        return (
          <Card
            key={a.id}
            variant="outlined"
            sx={{
              borderRadius: '8px',
              overflow: 'hidden',
              bgcolor: 'background.paper',
              boxShadow: '0 18px 45px rgba(0, 0, 0, 0.08)',
            }}
          >
            <CardContent sx={{ p: { xs: 2, sm: 2.5 }, '&:last-child': { pb: { xs: 2, sm: 2.5 } } }}>
              <Stack
                direction={{ xs: 'column', md: 'row' }}
                spacing={{ xs: 2, md: 2.5 }}
                sx={{ alignItems: { xs: 'stretch', md: 'flex-start' } }}
              >
                <Box
                  sx={{
                    display: 'grid',
                    placeItems: 'center',
                    width: 46,
                    height: 46,
                    borderRadius: '8px',
                    bgcolor: isAgent ? 'rgba(0, 102, 204, 0.12)' : 'rgba(107, 114, 128, 0.12)',
                    color: isAgent ? 'primary.main' : 'text.secondary',
                    flex: '0 0 auto',
                  }}
                >
                  {isAgent ? <AutoAwesomeRounded aria-hidden="true" /> : <Icon aria-hidden="true" />}
                </Box>

                <Stack spacing={1.5} sx={{ flexGrow: 1, minWidth: 0 }}>
                  <Stack
                    direction={{ xs: 'column', sm: 'row' }}
                    spacing={1}
                    useFlexGap
                    sx={{ alignItems: { xs: 'flex-start', sm: 'center' }, justifyContent: 'space-between' }}
                  >
                    <Stack direction="row" spacing={1} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
                      <Chip
                        size="small"
                        color={getStatusColor(a.status)}
                        label={getStatusLabel(a.status)}
                        sx={chipSx}
                      />
                      <Chip size="small" variant="outlined" label={sourceLabel(a.source)} sx={chipSx} />
                    </Stack>
                    <Box
                      sx={{
                        px: 1.25,
                        py: 0.65,
                        border: '1px solid',
                        borderColor: 'divider',
                        borderRadius: '8px',
                        fontFamily: fonts.mono,
                        fontSize: 13,
                        fontWeight: 800,
                        color: 'text.secondary',
                        bgcolor: 'rgba(107, 114, 128, 0.06)',
                      }}
                    >
                      Role ref {shortId(a.roleId)}
                    </Box>
                  </Stack>

                  <Stack spacing={0.75}>
                    <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', gap: 1 }}>
                      <Typography variant="body2" sx={{ fontWeight: 850 }}>
                        {meta.detail}
                      </Typography>
                      <Typography variant="caption" color="text.secondary" sx={{ fontFamily: fonts.mono, fontWeight: 800 }}>
                        {meta.progress}%
                      </Typography>
                    </Stack>
                    <LinearProgress
                      variant="determinate"
                      value={meta.progress}
                      sx={{
                        height: 7,
                        borderRadius: '999px',
                        bgcolor: 'rgba(107, 114, 128, 0.2)',
                        '& .MuiLinearProgress-bar': { borderRadius: '999px' },
                      }}
                    />
                  </Stack>

                  <Box
                    sx={{
                      p: 1.5,
                      border: '1px solid',
                      borderColor: 'divider',
                      borderRadius: '8px',
                      bgcolor: 'rgba(107, 114, 128, 0.06)',
                    }}
                  >
                    <Typography
                      variant="caption"
                      color="text.secondary"
                      sx={{ display: 'block', mb: 0.5, fontFamily: fonts.mono, fontWeight: 800, textTransform: 'uppercase' }}
                    >
                      Summary
                    </Typography>
                    <Typography variant="body2" sx={{ lineHeight: 1.55, overflowWrap: 'anywhere' }}>
                      {a.tailoredSummary || 'No summary recorded yet.'}
                    </Typography>
                  </Box>
                </Stack>
              </Stack>
            </CardContent>
          </Card>
        );
      })}
    </Stack>
  );
}
