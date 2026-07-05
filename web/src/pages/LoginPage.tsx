import LockOutlined from '@mui/icons-material/LockOutlined';
import Visibility from '@mui/icons-material/Visibility';
import VisibilityOff from '@mui/icons-material/VisibilityOff';
import { Alert, Box, IconButton, InputAdornment, Link as MuiLink, Stack, TextField, Typography } from '@mui/material';
import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link, useNavigate } from 'react-router-dom';

import { DotsButton } from '../components/DotsButton';
import { PublicAuthFrame } from '../components/PublicAuthFrame';
import { useLogin } from '../query/auth';
import { fonts } from '../theme/tokens';

export function LoginPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const login = useLogin();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const passwordResetHref = `mailto:support@projectcaliber.app?subject=${encodeURIComponent(t('auth.passwordResetSubject'))}`;

  const onSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    login.mutate({ email, password }, { onSuccess: () => navigate('/app', { replace: true }) });
  };

  return (
    <Box sx={{ py: { xs: 1, md: 3 } }}>
      <PublicAuthFrame eyebrow={t('auth.loginEyebrow')} proofTitle={t('auth.proofTitle')} proofBody={t('auth.proofBody')}>
        <Stack spacing={3} component="form" onSubmit={onSubmit}>
          <Stack spacing={1.25}>
            <Box
              sx={{
                display: 'grid',
                placeItems: 'center',
                width: 48,
                height: 48,
                borderRadius: 2,
                bgcolor: 'rgba(0, 102, 204, 0.1)',
                color: 'primary.main',
              }}
            >
              <LockOutlined />
            </Box>
            <Typography component="h1" sx={{ fontFamily: fonts.title, fontSize: { xs: 36, sm: 44 }, lineHeight: 1.06, fontWeight: 650 }}>
              {t('auth.welcomeBack')}
            </Typography>
            <Typography color="text.secondary" sx={{ fontSize: 17, lineHeight: 1.65 }}>
              {t('auth.signInPrompt')}
            </Typography>
          </Stack>
          {login.isError && <Alert severity="error">{login.error.message}</Alert>}
          <Stack spacing={2.25}>
            <TextField
              label={t('auth.email')}
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete="email"
              required
              fullWidth
            />
            <Stack spacing={0.75}>
              <TextField
                label={t('auth.password')}
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                autoComplete="current-password"
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
              <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
                <MuiLink href={passwordResetHref} underline="hover" variant="body2" sx={{ fontWeight: 700 }}>
                  {t('auth.forgotPassword')}
                </MuiLink>
              </Box>
            </Stack>
          </Stack>
          <DotsButton type="submit" variant="contained" size="large" loading={login.isPending}>
            {t('auth.submitSignIn')}
          </DotsButton>
          <Typography variant="body2" color="text.secondary">
            {t('auth.noAccount')}{' '}
            <MuiLink component={Link} to="/register">
              {t('auth.createOne')}
            </MuiLink>
          </Typography>
        </Stack>
      </PublicAuthFrame>
    </Box>
  );
}
