import RadarRounded from '@mui/icons-material/RadarRounded';
import SpeedOutlined from '@mui/icons-material/SpeedOutlined';
import { Box, Card, CardContent, LinearProgress, Stack, Typography } from '@mui/material';
import { alpha } from '@mui/material/styles';

import type { TimeToShortlistMetric } from '../../api/types';
import { fonts } from '../../theme/tokens';

export function TimeToShortlistHeadline({ metric }: { metric: TimeToShortlistMetric }) {
  const days = Math.round(metric.baselineHours / 24);
  const currentShare = Math.max(4, Math.min(100, (metric.currentHours / Math.max(1, metric.baselineHours)) * 100));
  return (
    <Card
      variant="outlined"
      sx={(theme) => ({
        borderRadius: '8px',
        bgcolor: 'background.paper',
        overflow: 'hidden',
        height: '100%',
        borderColor: alpha(theme.palette.primary.main, 0.36),
        boxShadow: '0 18px 54px rgba(0, 0, 0, 0.16)',
      })}
    >
      <CardContent
        sx={(theme) => ({
          p: { xs: 2.25, md: 2.75 },
          height: '100%',
          background:
            `linear-gradient(135deg, ${alpha(theme.palette.primary.main, 0.2)}, transparent 44%), ` +
            `linear-gradient(180deg, ${alpha(theme.palette.success.main, 0.08)}, transparent 70%)`,
        })}
      >
        <Stack spacing={2.25} sx={{ height: '100%' }}>
          <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
            <Box
              sx={{
                width: 44,
                height: 44,
                borderRadius: '8px',
                display: 'grid',
                placeItems: 'center',
                bgcolor: 'rgba(77, 155, 255, 0.14)',
                color: 'primary.main',
              }}
            >
              <SpeedOutlined />
            </Box>
            <Box>
              <Typography component="h2" sx={{ fontFamily: fonts.body, fontWeight: 850, fontSize: { xs: 22, md: 24 }, lineHeight: 1.1 }}>
                Time to shortlist
              </Typography>
              <Typography sx={{ mt: 0.5, color: 'text.secondary', fontSize: 14 }}>Weeks-to-hours demo metric</Typography>
            </Box>
          </Stack>

          <Box>
            <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1 }}>
              <Typography component="p" sx={{ fontFamily: fonts.body, fontSize: { xs: 54, md: 72 }, fontWeight: 900, lineHeight: 0.9 }}>
              {Math.round(metric.improvementFactor)}×
              </Typography>
              <Typography component="span" sx={{ color: 'text.secondary', fontSize: 18, fontWeight: 850 }}>
                faster
              </Typography>
            </Box>
            <Typography sx={{ mt: 1, color: 'text.secondary', lineHeight: 1.5 }}>
              From ~{days} days to {metric.currentHours} hours — weeks collapse to hours.
            </Typography>
          </Box>

          <Stack spacing={1.1} sx={{ mt: 'auto' }}>
            <Box>
              <Stack direction="row" sx={{ justifyContent: 'space-between', mb: 0.45 }}>
                <Typography sx={{ color: 'text.secondary', fontSize: 11, fontWeight: 800 }}>Baseline</Typography>
                <Typography sx={{ fontFamily: fonts.mono, fontSize: 12 }}>~{days} days</Typography>
              </Stack>
              <LinearProgress
                aria-label="baseline shortlist time"
                variant="determinate"
                value={100}
                sx={{ height: 7, borderRadius: '999px', bgcolor: 'rgba(148, 163, 184, 0.18)' }}
              />
            </Box>
            <Box>
              <Stack direction="row" sx={{ justifyContent: 'space-between', mb: 0.45 }}>
                <Typography sx={{ color: 'text.secondary', fontSize: 11, fontWeight: 800 }}>Current</Typography>
                <Typography sx={{ fontFamily: fonts.mono, fontSize: 12 }}>{metric.currentHours}h</Typography>
              </Stack>
              <LinearProgress
                aria-label="current shortlist time"
                variant="determinate"
                color="success"
                value={currentShare}
                sx={{ height: 7, borderRadius: '999px', bgcolor: 'rgba(148, 163, 184, 0.18)' }}
              />
            </Box>
          </Stack>

          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', color: 'text.secondary' }}>
            <RadarRounded fontSize="small" />
            <Typography sx={{ fontSize: 13, fontWeight: 750 }}>Live shortlist acceleration</Typography>
          </Stack>
        </Stack>
      </CardContent>
    </Card>
  );
}
