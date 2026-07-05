import ArrowBackRounded from '@mui/icons-material/ArrowBackRounded';
import LoginRounded from '@mui/icons-material/LoginRounded';
import { Box, Button, Stack, Typography } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { motion, useReducedMotion } from 'motion/react';
import { useTranslation } from 'react-i18next';
import { Link, useLocation } from 'react-router-dom';

import { fonts } from '../theme/tokens';

// The 404 in Caliber's own vocabulary: a record with no source — a break in the
// evidence chain. Signature element is the oversized Fraunces "404" printed with
// a teal misregistration ghost, as if the record never cleanly resolved.
export function NotFoundPage() {
  const { t } = useTranslation();
  const reduceMotion = useReducedMotion();
  // NotFoundRedirect (App.tsx) forwards the originally-requested path here as
  // navigation state. Absent on a direct /404 hit — then we hide the route line.
  const { state } = useLocation();
  const attemptedRoute = (state as { from?: string } | null)?.from ?? '';

  return (
    <Box sx={{ display: 'flex', justifyContent: 'center', px: 2, py: { xs: 6, md: 10 } }}>
      <motion.div
        initial={reduceMotion ? false : { opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, ease: [0.16, 1, 0.3, 1] }}
        style={{ width: '100%', maxWidth: 560 }}
      >
        <Stack spacing={3}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
            <Box
              aria-hidden
              sx={{
                width: 8,
                height: 8,
                borderRadius: '50%',
                bgcolor: '#2DD4BF',
                boxShadow: `0 0 0 4px ${alpha('#2DD4BF', 0.18)}`,
              }}
            />
            <Typography
              component="p"
              sx={{
                fontFamily: fonts.mono,
                fontSize: 12,
                textTransform: 'uppercase',
                letterSpacing: '0.14em',
                color: 'text.secondary',
              }}
            >
              {t('notFound.eyebrow')}
            </Typography>
          </Stack>

          <Box aria-hidden sx={{ position: 'relative', lineHeight: 0.82 }}>
            <Typography
              component="span"
              sx={{
                position: 'absolute',
                inset: 0,
                fontFamily: fonts.title,
                fontWeight: 600,
                fontSize: { xs: 120, sm: 168, md: 196 },
                letterSpacing: '-0.03em',
                color: 'rgba(45, 212, 191, 0.55)',
                transform: 'translate(0.09em, 0.09em)',
                userSelect: 'none',
              }}
            >
              404
            </Typography>
            <Typography
              component="span"
              sx={{
                position: 'relative',
                display: 'block',
                fontFamily: fonts.title,
                fontWeight: 600,
                fontSize: { xs: 120, sm: 168, md: 196 },
                letterSpacing: '-0.03em',
                color: 'text.primary',
              }}
            >
              404
            </Typography>
          </Box>

          <Stack spacing={1.5}>
            <Typography
              component="h1"
              sx={{
                fontFamily: fonts.title,
                fontWeight: 600,
                fontSize: { xs: 26, sm: 31 },
                letterSpacing: '-0.01em',
                lineHeight: 1.14,
              }}
            >
              {t('notFound.heading')}
            </Typography>
            <Typography sx={{ color: 'text.secondary', fontSize: 17, lineHeight: 1.7, maxWidth: 468 }}>
              {t('notFound.message')}
            </Typography>
          </Stack>

          {attemptedRoute && (
            <Box
              sx={(theme) => ({
                display: 'flex',
                gap: 1.25,
                alignItems: 'baseline',
                flexWrap: 'wrap',
                px: 1.75,
                py: 1.25,
                borderRadius: 1.5,
                border: 1,
                borderColor: 'divider',
                bgcolor: alpha(theme.palette.text.primary, 0.02),
              })}
            >
              <Typography
                component="span"
                sx={{
                  fontFamily: fonts.mono,
                  fontSize: 11,
                  textTransform: 'uppercase',
                  letterSpacing: '0.12em',
                  color: 'text.secondary',
                }}
              >
                {t('notFound.routeLabel')}
              </Typography>
              <Typography
                component="span"
                sx={{ fontFamily: fonts.mono, fontSize: 13, color: 'text.primary', wordBreak: 'break-all' }}
              >
                {attemptedRoute}
              </Typography>
            </Box>
          )}

          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} sx={{ pt: 0.5 }}>
            <Button component={Link} to="/" variant="contained" size="large" startIcon={<ArrowBackRounded />}>
              {t('notFound.backHome')}
            </Button>
            <Button component={Link} to="/login" variant="text" size="large" startIcon={<LoginRounded />}>
              {t('notFound.signIn')}
            </Button>
          </Stack>
        </Stack>
      </motion.div>
    </Box>
  );
}
