import { Box, Card, CardContent, Chip, Stack, Typography } from '@mui/material';

import type { Contest } from '../../api/types';
import { contestStatusColor, contestStatusLabel } from '../../lib/format';

const SUBJECT_LABEL: Record<Contest['subject'], string> = {
  CONTEST_SUBJECT_UNSPECIFIED: 'Assessment',
  CONTEST_SUBJECT_MATCH: 'Shortlist result',
  CONTEST_SUBJECT_REPORT_CARD: 'Report card',
};

export function MyContestsList({ contests }: { contests: Contest[] }) {
  if (contests.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary">
        You have not disputed any assessments.
      </Typography>
    );
  }
  return (
    <Stack spacing={2}>
      {contests.map((c) => (
        <Card key={c.id} variant="outlined">
          <CardContent>
            <Stack spacing={1}>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                <Chip size="small" variant="outlined" label={SUBJECT_LABEL[c.subject]} />
                <Box sx={{ flexGrow: 1 }} />
                <Chip size="small" color={contestStatusColor[c.status]} label={contestStatusLabel[c.status]} />
              </Stack>
              <Typography variant="body2">{c.reason}</Typography>
              {c.resolution && (
                <Typography variant="caption" color="text.secondary">
                  Reviewer note: {c.resolution}
                </Typography>
              )}
            </Stack>
          </CardContent>
        </Card>
      ))}
    </Stack>
  );
}
