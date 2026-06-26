import { Box, Chip, Stack, Typography } from '@mui/material';

import type { InterviewTurn } from '../../hooks/useInterview';

export function TranscriptList({ turns }: { turns: InterviewTurn[] }) {
  if (turns.length === 0) {
    return null;
  }
  return (
    <Stack spacing={2}>
      {turns.map((t) => (
        <Box key={t.ordinal}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 0.5 }}>
            <Chip size="small" variant="outlined" label={t.competencyTag || `Q${t.ordinal}`} />
            <Typography variant="body2" color="text.secondary">
              {t.question}
            </Typography>
          </Stack>
          <Typography variant="body1" sx={{ pl: 1, borderLeft: 2, borderColor: 'primary.main' }}>
            {t.answer}
          </Typography>
        </Box>
      ))}
    </Stack>
  );
}
