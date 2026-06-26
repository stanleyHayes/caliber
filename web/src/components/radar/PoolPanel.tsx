import { Box, Card, CardContent, Chip, Stack, Typography } from '@mui/material';

import type { PoolCandidate } from '../../api/types';
import { passportColor, passportLabel, pct, shortId } from '../../lib/format';

export function PoolPanel({ candidates }: { candidates: PoolCandidate[] }) {
  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={1.5}>
          <Typography variant="h6">Live talent pool</Typography>
          {candidates.length === 0 && (
            <Typography variant="body2" color="text.secondary">
              No candidates in the pool yet.
            </Typography>
          )}
          {candidates.map((c) => (
            <Stack key={c.candidateId} direction="row" spacing={1} sx={{ alignItems: 'center' }}>
              <Typography variant="body2" sx={{ flexGrow: 1 }}>
                {c.name || shortId(c.candidateId)}
              </Typography>
              <Chip size="small" color={passportColor[c.passportStatus]} label={passportLabel[c.passportStatus]} />
              <Box sx={{ width: 48, textAlign: 'right' }}>
                <Typography variant="body2">{pct(c.headlineScore)}</Typography>
              </Box>
            </Stack>
          ))}
        </Stack>
      </CardContent>
    </Card>
  );
}
