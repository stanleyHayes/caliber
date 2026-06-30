import {
  Alert,
  Box,
  Link as MuiLink,
  MenuItem,
  Paper,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';

import type { UserRole } from '../api/types';
import { DotsButton } from '../components/DotsButton';
import { useRegister } from '../query/auth';

const ROLES: { value: UserRole; label: string }[] = [
  { value: 'USER_ROLE_EMPLOYER', label: 'Employer' },
  { value: 'USER_ROLE_RECRUITER', label: 'Recruiter' },
  { value: 'USER_ROLE_CANDIDATE', label: 'Candidate' },
];

export function RegisterPage() {
  const navigate = useNavigate();
  const register = useRegister();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [role, setRole] = useState<UserRole>('USER_ROLE_EMPLOYER');

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    register.mutate({ name, email, password, role }, { onSuccess: () => navigate('/app', { replace: true }) });
  };

  return (
    <Box sx={{ maxWidth: 460, mx: 'auto', mt: { xs: 2, md: 5 } }}>
      <Paper variant="outlined" sx={{ p: { xs: 3, sm: 4 } }}>
        <Stack spacing={3} component="form" onSubmit={onSubmit}>
          <Stack spacing={0.5}>
            <Typography variant="h4" component="h1">Create your account</Typography>
            <Typography color="text.secondary">Passwords must be at least 12 characters.</Typography>
          </Stack>
          {register.isError && <Alert severity="error">{register.error.message}</Alert>}
          <TextField label="Full name" value={name} onChange={(e) => setName(e.target.value)} required fullWidth />
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
            autoComplete="new-password"
            required
            fullWidth
          />
          <TextField
            select
            label="I am a…"
            value={role}
            onChange={(e) => setRole(e.target.value as UserRole)}
            fullWidth
          >
            {ROLES.map((r) => (
              <MenuItem key={r.value} value={r.value}>
                {r.label}
              </MenuItem>
            ))}
          </TextField>
          <DotsButton type="submit" variant="contained" size="large" loading={register.isPending}>
            Create account
          </DotsButton>
          <Typography variant="body2" color="text.secondary">
            Already registered?{' '}
            <MuiLink component={Link} to="/login">
              Sign in
            </MuiLink>
          </Typography>
        </Stack>
      </Paper>
    </Box>
  );
}
