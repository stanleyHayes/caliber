import { Box, Card, CardContent, Chip, LinearProgress, Stack, Typography } from '@mui/material';

import type { TalentProfile } from '../../api/types';
import { passportColor, passportLabel } from '../../lib/format';

export function ProfileView({ profile }: { profile: TalentProfile }) {
  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={2.5}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', rowGap: 0.5 }}>
            <Typography variant="h6" component="h2" sx={{ flexGrow: 1 }}>
              Your Talent Passport
            </Typography>
            <Chip color={passportColor[profile.passportStatus]} label={passportLabel[profile.passportStatus]} />
          </Stack>
          {profile.summary && (
            <Typography variant="body2" color="text.secondary">
              {profile.summary}
            </Typography>
          )}
          <Stack spacing={1.75}>
            {profile.competencies.map((c, i) => (
              <Box key={i}>
                <Stack direction="row" sx={{ justifyContent: 'space-between', mb: 0.5 }}>
                  <Typography variant="body2">{c.name}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    {c.level.toFixed(1)} / 5
                  </Typography>
                </Stack>
                <LinearProgress variant="determinate" value={(c.level / 5) * 100} sx={{ height: 6, borderRadius: 1 }} />
                {c.evidenceQuote && (
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                    “{c.evidenceQuote}”
                  </Typography>
                )}
              </Box>
            ))}
          </Stack>
        </Stack>
      </CardContent>
    </Card>
  );
}
