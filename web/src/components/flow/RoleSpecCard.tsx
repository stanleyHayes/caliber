import { Card, CardContent, Chip, Divider, Stack, Typography } from '@mui/material';

import type { RoleSpec } from '../../api/types';
import { seniorityLabel } from '../../lib/format';

function Chips({ label, items, color }: { label: string; items: string[]; color?: 'primary' | 'default' }) {
  if (items.length === 0) {
    return null;
  }
  return (
    <Stack spacing={0.75}>
      <Typography variant="overline" color="text.secondary">
        {label}
      </Typography>
      <Stack direction="row" useFlexGap spacing={0.75} sx={{ flexWrap: 'wrap' }}>
        {items.map((it, i) => (
          <Chip key={i} label={it} size="small" color={color ?? 'default'} variant={color ? 'filled' : 'outlined'} />
        ))}
      </Stack>
    </Stack>
  );
}

export function RoleSpecCard({ spec }: { spec: RoleSpec }) {
  const salary =
    spec.salaryBand.low || spec.salaryBand.high
      ? `${spec.salaryBand.currency} ${spec.salaryBand.low}–${spec.salaryBand.high}`
      : 'Not specified';
  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={2}>
          <Stack spacing={0.5}>
            <Typography variant="h5">{spec.title}</Typography>
            <Stack direction="row" useFlexGap spacing={1} sx={{ flexWrap: 'wrap' }}>
              <Chip size="small" label={seniorityLabel[spec.seniority]} />
              {spec.location && <Chip size="small" variant="outlined" label={spec.location} />}
              {spec.availability && <Chip size="small" variant="outlined" label={spec.availability} />}
              <Chip size="small" variant="outlined" label={salary} />
            </Stack>
          </Stack>
          <Divider />
          <Chips label="Responsibilities" items={spec.responsibilities} />
          <Chips label="Must-haves" items={spec.mustHaves} color="primary" />
          <Chips label="Nice-to-haves" items={spec.niceToHaves} />
        </Stack>
      </CardContent>
    </Card>
  );
}
