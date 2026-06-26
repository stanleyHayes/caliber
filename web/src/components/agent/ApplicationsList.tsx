import { Box, Card, CardContent, Chip, Stack, Typography } from '@mui/material';

import type { Application } from '../../api/types';
import { applicationStatusColor, applicationStatusLabel, shortId } from '../../lib/format';

export function ApplicationsList({ applications }: { applications: Application[] }) {
  if (applications.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary">
        No applications yet. Run your agent to apply on your behalf.
      </Typography>
    );
  }
  return (
    <Stack spacing={2}>
      {applications.map((a) => (
        <Card key={a.id} variant="outlined">
          <CardContent>
            <Stack spacing={1}>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                <Chip size="small" color={applicationStatusColor[a.status]} label={applicationStatusLabel[a.status]} />
                {a.source === 'APPLICATION_SOURCE_AGENT' && <Chip size="small" variant="outlined" label="by your agent" />}
                <Box sx={{ flexGrow: 1 }} />
                <Typography variant="caption" color="text.secondary">
                  role {shortId(a.roleId)}
                </Typography>
              </Stack>
              <Typography variant="body2">{a.tailoredSummary}</Typography>
            </Stack>
          </CardContent>
        </Card>
      ))}
    </Stack>
  );
}
