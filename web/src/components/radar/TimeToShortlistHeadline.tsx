import { Box, Card, CardContent, Stack, Typography } from '@mui/material';

import type { TimeToShortlistMetric } from '../../api/types';
import { fonts } from '../../theme/tokens';

export function TimeToShortlistHeadline({ metric }: { metric: TimeToShortlistMetric }) {
  const days = Math.round(metric.baselineHours / 24);
  return (
    <Card variant="outlined" sx={{ bgcolor: 'primary.main', color: 'primary.contrastText' }}>
      <CardContent>
        <Stack spacing={1}>
          <Typography variant="overline" sx={{ opacity: 0.85 }}>
            Time to shortlist
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1 }}>
            <Typography sx={{ fontFamily: fonts.title, fontSize: { xs: 44, md: 64 }, fontWeight: 700, lineHeight: 1 }}>
              {Math.round(metric.improvementFactor)}×
            </Typography>
            <Typography variant="h6" sx={{ opacity: 0.9 }}>
              faster
            </Typography>
          </Box>
          <Typography sx={{ opacity: 0.9 }}>
            From ~{days} days to {metric.currentHours} hours — weeks collapse to hours.
          </Typography>
        </Stack>
      </CardContent>
    </Card>
  );
}
