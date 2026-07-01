import { Box, Card, CardContent, LinearProgress, Stack, Typography } from '@mui/material';

import type { SupplyDemandItem } from '../../api/types';

export function SupplyDemandPanel({ items }: { items: SupplyDemandItem[] }) {
  const max = Math.max(1, ...items.map((i) => Math.max(i.openRoles, i.availableCandidates)));
  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={2}>
          <Typography variant="h6" component="h2">Supply &amp; demand</Typography>
          {items.length === 0 && (
            <Typography variant="body2" color="text.secondary">
              No open roles yet.
            </Typography>
          )}
          {items.map((it) => (
            <Box key={it.roleFamily}>
              <Stack direction="row" sx={{ justifyContent: 'space-between', mb: 0.5, flexWrap: 'wrap', gap: 0.5 }}>
                <Typography variant="body2" sx={{ textTransform: 'capitalize' }}>
                  {it.roleFamily}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {it.openRoles} open · {it.availableCandidates} candidates · gap {it.gap}
                </Typography>
              </Stack>
              <LinearProgress
                variant="determinate"
                value={(it.openRoles / max) * 100}
                sx={{ height: 8, borderRadius: 1 }}
              />
            </Box>
          ))}
        </Stack>
      </CardContent>
    </Card>
  );
}
