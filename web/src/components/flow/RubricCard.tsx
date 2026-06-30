import { Box, Card, CardContent, Chip, LinearProgress, Stack, Typography } from '@mui/material';

import type { Rubric } from '../../api/types';
import { pct } from '../../lib/format';

export function RubricCard({ rubric }: { rubric: Rubric }) {
  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={2}>
          <Typography variant="h6">Scoring rubric</Typography>
          {rubric.competencies.map((c, i) => (
            <Box key={i}>
              <Stack direction="row" sx={{ mb: 0.5, alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 0.5 }}>
                <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                  <Typography variant="body2">{c.name}</Typography>
                  {c.mustHave && <Chip size="small" color="primary" label="must-have" />}
                </Stack>
                <Typography variant="caption" color="text.secondary">
                  {pct(c.weight)}
                </Typography>
              </Stack>
              <LinearProgress variant="determinate" value={Math.min(100, c.weight * 100)} sx={{ borderRadius: 1, height: 6 }} />
            </Box>
          ))}
        </Stack>
      </CardContent>
    </Card>
  );
}
