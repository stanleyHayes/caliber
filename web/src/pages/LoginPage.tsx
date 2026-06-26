import { Alert, Box, Link as MuiLink, Paper, Stack, TextField, Typography } from '@mui/material';
import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';

import { DotsButton } from '../components/DotsButton';
import { useLogin } from '../query/auth';

export function LoginPage() {
  const navigate = useNavigate();
  const login = useLogin();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    login.mutate({ email, password }, { onSuccess: () => navigate('/', { replace: true }) });
  };

  return (
    <Box sx={{ maxWidth: 420, mx: 'auto', mt: { xs: 2, md: 6 } }}>
      <Paper variant="outlined" sx={{ p: 4 }}>
        <Stack spacing={3} component="form" onSubmit={onSubmit}>
          <Stack spacing={0.5}>
            <Typography variant="h4">Welcome back</Typography>
            <Typography color="text.secondary">Sign in to Caliber.</Typography>
          </Stack>
          {login.isError && <Alert severity="error">{login.error.message}</Alert>}
          <TextField
            label="Email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            autoComplete="email"
            required
            fullWidth
          />
          <TextField
            label="Password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
            required
            fullWidth
          />
          <DotsButton type="submit" variant="contained" size="large" loading={login.isPending}>
            Sign in
          </DotsButton>
          <Typography variant="body2" color="text.secondary">
            No account?{' '}
            <MuiLink component={Link} to="/register">
              Create one
            </MuiLink>
          </Typography>
        </Stack>
      </Paper>
    </Box>
  );
}
