import { AppBar, Box, Button, Container, Toolbar, Typography } from '@mui/material';
import { Link, Outlet } from 'react-router-dom';

import { useLogout } from '../query/auth';
import { useAuthStore } from '../stores/auth';
import { ModeToggle } from './ModeToggle';

export function AppShell() {
  const user = useAuthStore((s) => s.user);
  const accessToken = useAuthStore((s) => s.accessToken);
  const logout = useLogout();
  return (
    <Box sx={{ minHeight: '100dvh', bgcolor: 'background.default' }}>
      <AppBar position="sticky" color="default" elevation={0} sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Toolbar>
          <Typography
            variant="h6"
            component={Link}
            to="/"
            sx={{ flexGrow: 1, textDecoration: 'none', color: 'text.primary', fontFamily: (t) => t.typography.h6.fontFamily }}
          >
            Caliber
          </Typography>
          {accessToken && (
            <Button component={Link} to="/radar" color="inherit" sx={{ mr: 1 }}>
              Radar
            </Button>
          )}
          <ModeToggle />
          {accessToken && (
            <Button onClick={() => logout.mutate()} sx={{ ml: 1 }} color="inherit">
              Sign out{user ? ` (${user.name})` : ''}
            </Button>
          )}
        </Toolbar>
      </AppBar>
      <Container maxWidth="lg" sx={{ py: { xs: 3, md: 5 } }}>
        <Outlet />
      </Container>
    </Box>
  );
}
