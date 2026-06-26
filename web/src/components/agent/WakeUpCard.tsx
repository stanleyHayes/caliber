import { Box, Card, CardContent, Stack, Typography } from '@mui/material';

import type { WakeUpView } from '../../api/types';

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <Box sx={{ minWidth: 120 }}>
      <Typography variant="h4">{value}</Typography>
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
    </Box>
  );
}

export function WakeUpCard({ wakeUp }: { wakeUp: WakeUpView }) {
  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={2.5}>
          <Typography variant="h5">While you were away</Typography>
          <Stack direction="row" useFlexGap sx={{ flexWrap: 'wrap', gap: 3 }}>
            <Stat label="New matches" value={wakeUp.newMatches} />
            <Stat label="Applications submitted" value={wakeUp.applicationsSubmitted} />
            <Stat label="Screenings completed" value={wakeUp.screeningsCompleted} />
            <Stat label="Employers interested" value={wakeUp.employersInterested} />
          </Stack>
          {wakeUp.highlights.length > 0 && (
            <Stack component="ul" spacing={0.5} sx={{ m: 0, pl: 2.5 }}>
              {wakeUp.highlights.map((h, i) => (
                <Typography key={i} component="li" variant="body2" color="text.secondary">
                  {h}
                </Typography>
              ))}
            </Stack>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}
