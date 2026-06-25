import { Box, Button, Chip, Container, Paper, Stack, Typography, useColorScheme } from '@mui/material';

import { fonts } from './theme/tokens';

function ModeToggle() {
  const { mode, setMode } = useColorScheme();
  const next = mode === 'dark' ? 'light' : 'dark';
  return (
    <Button variant="outlined" onClick={() => setMode(next)}>
      Switch to {next} mode
    </Button>
  );
}

export function App() {
  return (
    <Container maxWidth="md" sx={{ py: { xs: 6, md: 10 } }}>
      <Stack spacing={4}>
        <Stack spacing={1}>
          <Chip label="POC" color="primary" size="small" sx={{ alignSelf: 'flex-start' }} />
          <Typography variant="h1" sx={{ fontSize: { xs: 40, md: 64 } }}>
            Project Caliber
          </Typography>
          <Typography variant="h5" color="text.secondary">
            Explainable, bias-safe talent intelligence.
          </Typography>
        </Stack>

        <Typography color="text.secondary">
          The design system is live: Fraunces for titles, Outfit for body, and JetBrains Mono for
          statuses. Colors and typography come from centralized tokens, themed for light and dark.
        </Typography>

        <Paper variant="outlined" sx={{ p: 2 }}>
          <Box sx={{ fontFamily: fonts.mono, fontSize: 14, letterSpacing: '0.02em' }}>
            STATUS: SCAFFOLD_READY · API: /v1 · AUTH: JWT
          </Box>
        </Paper>

        <ModeToggle />
      </Stack>
    </Container>
  );
}
