import {
  Alert,
  Box,
  IconButton,
  InputAdornment,
  Link as MuiLink,
  MenuItem,
  Stack,
  TextField,
  Typography,
} from '@mui/material';
import PersonAddAlt1Outlined from '@mui/icons-material/PersonAddAlt1Outlined';
import Visibility from '@mui/icons-material/Visibility';
import VisibilityOff from '@mui/icons-material/VisibilityOff';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useNavigate } from 'react-router-dom';

import type { UserRole } from '../api/types';
import { DotsButton } from '../components/DotsButton';
import { PublicAuthFrame } from '../components/PublicAuthFrame';
import { useRegister } from '../query/auth';
import { fonts } from '../theme/tokens';

export function RegisterPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const register = useRegister();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [role, setRole] = useState<UserRole>('USER_ROLE_EMPLOYER');

  const roles: { value: UserRole; label: string }[] = [
    { value: 'USER_ROLE_EMPLOYER', label: t('auth.roleEmployer') },
    { value: 'USER_ROLE_RECRUITER', label: t('auth.roleRecruiter') },
    { value: 'USER_ROLE_CANDIDATE', label: t('auth.roleCandidate') },
  ];

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    register.mutate({ name, email, password, role }, { onSuccess: () => navigate('/app', { replace: true }) });
  };

  return (
    <Box sx={{ py: { xs: 1, md: 3 } }}>
      <PublicAuthFrame eyebrow={t('auth.registerEyebrow')} proofTitle={t('auth.proofTitle')} proofBody={t('auth.proofBody')}>
        <Stack spacing={3} component="form" onSubmit={onSubmit}>
          <Stack spacing={1.25}>
            <Box
              sx={{
                display: 'grid',
                placeItems: 'center',
                width: 48,
                height: 48,
                borderRadius: 2,
                bgcolor: 'rgba(245, 158, 11, 0.14)',
                color: '#B45309',
              }}
            >
              <PersonAddAlt1Outlined />
            </Box>
            <Typography component="h1" sx={{ fontFamily: fonts.title, fontSize: { xs: 36, sm: 44 }, lineHeight: 1.06, fontWeight: 650 }}>
              {t('auth.createAccount')}
            </Typography>
            <Typography color="text.secondary" sx={{ fontSize: 17, lineHeight: 1.65 }}>
              {t('auth.passwordHint')}
            </Typography>
          </Stack>
          {register.isError && <Alert severity="error">{register.error.message}</Alert>}
          <Stack spacing={2.25}>
            <TextField label={t('auth.fullName')} value={name} onChange={(e) => setName(e.target.value)} required fullWidth />
            <TextField
              label={t('auth.email')}
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              required
              fullWidth
            />
            <TextField
              label={t('auth.password')}
              type={showPassword ? 'text' : 'password'}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="new-password"
              required
              fullWidth
              slotProps={{
                input: {
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton
                        aria-label={showPassword ? t('auth.hidePassword') : t('auth.showPassword')}
                        onClick={() => setShowPassword((s) => !s)}
                        onMouseDown={(e) => e.preventDefault()}
                        edge="end"
                      >
                        {showPassword ? <VisibilityOff fontSize="small" /> : <Visibility fontSize="small" />}
                      </IconButton>
                    </InputAdornment>
                  ),
                },
              }}
            />
            <TextField
              select
              label={t('auth.role')}
              value={role}
              onChange={(e) => setRole(e.target.value as UserRole)}
              fullWidth
            >
              {roles.map((r) => (
                <MenuItem key={r.value} value={r.value}>
                  {r.label}
                </MenuItem>
              ))}
            </TextField>
          </Stack>
          <DotsButton type="submit" variant="contained" size="large" loading={register.isPending}>
            {t('auth.submitCreate')}
          </DotsButton>
          <Typography variant="body2" color="text.secondary">
            {t('auth.hasAccount')}{' '}
            <MuiLink component={Link} to="/login">
              {t('auth.signInLink')}
            </MuiLink>
          </Typography>
        </Stack>
      </PublicAuthFrame>
    </Box>
  );
}
