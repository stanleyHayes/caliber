import { AppBar, Box, Button, Container, Toolbar, Typography } from '@mui/material';
import { AnimatePresence, motion } from 'motion/react';
import { Link, Outlet, useLocation } from 'react-router-dom';

import { useLogout } from '../query/auth';
import { useAuthStore } from '../stores/auth';
import { ModeToggle } from './ModeToggle';

export function AppShell() {
  const location = useLocation();
  const user = useAuthStore((s) => s.user);
  const accessToken = useAuthStore((s) => s.accessToken);
  const logout = useLogout();
  return (
    <Box sx={{ minHeight: '100dvh', bgcolor: 'background.default' }}>
      <Box
        component="a"
        href="#main-content"
        sx={{
          position: 'absolute',
          left: 8,
          top: -48,
          px: 2,
          py: 1,
          bgcolor: 'background.paper',
          color: 'text.primary',
          border: 1,
          borderColor: 'divider',
          borderRadius: 1,
          zIndex: (t) => t.zIndex.tooltip + 1,
          transition: 'top 0.15s ease',
          '&:focus-visible': { top: 8 },
        }}
      >
        Skip to main content
      </Box>
      <AppBar position="sticky" color="default" elevation={0} sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Toolbar>
          <Typography
            variant="h6"
            component={Link}
            to={accessToken ? '/app' : '/'}
            sx={{ flexGrow: 1, textDecoration: 'none', color: 'text.primary', fontFamily: (t) => t.typography.h6.fontFamily }}
          >
            Caliber
          </Typography>
          <Box
            component="nav"
            aria-label="Primary"
            sx={{
              display: 'flex',
              alignItems: 'center',
              flexWrap: 'wrap',
              justifyContent: 'flex-end',
              rowGap: 0.5,
              columnGap: 1,
            }}
          >
            {accessToken && (
              <Button component={Link} to="/radar" color="inherit" size="small">
                Radar
              </Button>
            )}
            <ModeToggle />
            {accessToken ? (
              <Button
                onClick={() => logout.mutate()}
                size="small"
                color="inherit"
                aria-label={user ? `Sign out (${user.name})` : 'Sign out'}
              >
                Sign out
                {user && (
                  <Box component="span" sx={{ display: { xs: 'none', sm: 'inline' }, ml: 0.5 }}>
                    ({user.name})
                  </Box>
                )}
              </Button>
            ) : (
              <>
                <Button component={Link} to="/login" color="inherit" size="small">
                  Sign in
                </Button>
                <Button component={Link} to="/register" variant="contained" size="small">
                  Get started
                </Button>
              </>
            )}
          </Box>
        </Toolbar>
      </AppBar>
      <Container component="main" id="main-content" maxWidth="lg" sx={{ py: { xs: 3, md: 5 } }}>
        <AnimatePresence mode="wait">
          <motion.div
            key={location.pathname}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -8 }}
            transition={{ duration: 0.2, ease: 'easeOut' }}
          >
            <Outlet />
          </motion.div>
        </AnimatePresence>
      </Container>
    </Box>
  );
}
