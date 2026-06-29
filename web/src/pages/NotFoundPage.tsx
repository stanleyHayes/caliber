import { Button, Stack, Typography } from '@mui/material';
import { Link } from 'react-router-dom';

export function NotFoundPage() {
  return (
    <Stack spacing={2} sx={{ py: 6, alignItems: 'flex-start' }}>
      <Typography variant="h3" component="h1">Not found</Typography>
      <Typography color="text.secondary">That page does not exist.</Typography>
      <Button component={Link} to="/" variant="contained">
        Back home
      </Button>
    </Stack>
  );
}
