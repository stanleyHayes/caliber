import { Box, Skeleton, Stack } from '@mui/material';
import { useEffect, useState, type ReactNode } from 'react';

import { authApi } from '../api/auth';
import { tryRefresh } from '../api/client';
import { useAuthStore } from '../stores/auth';

// A session is immediately ready when there is already an access token or no
// refresh token to restore from; otherwise we refresh + load the user once.
function initiallyReady(): boolean {
  const { accessToken, refreshToken } = useAuthStore.getState();
  return Boolean(accessToken) || !refreshToken;
}

export function SessionBootstrap({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(initiallyReady);

  useEffect(() => {
    if (ready) {
      return;
    }
    let cancelled = false;
    void (async () => {
      if (await tryRefresh()) {
        try {
          const { user } = await authApi.me();
          if (!cancelled) {
            useAuthStore.getState().setUser(user);
          }
        } catch {
          // a failed /me leaves the session token-only; the UI still works
        }
      }
      if (!cancelled) {
        setReady(true);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [ready]);

  if (!ready) {
    return (
      <Box sx={{ maxWidth: 880, mx: 'auto', p: 4 }}>
        <Stack spacing={2}>
          <Skeleton variant="rounded" height={56} />
          <Skeleton width="55%" height={32} />
          <Skeleton variant="rounded" height={180} />
        </Stack>
      </Box>
    );
  }
  return <>{children}</>;
}
