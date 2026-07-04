import ArrowBackRounded from '@mui/icons-material/ArrowBackRounded';
import AutoAwesomeOutlined from '@mui/icons-material/AutoAwesomeOutlined';
import ShieldOutlined from '@mui/icons-material/ShieldOutlined';
import { Box, Button, Paper, Stack, Typography } from '@mui/material';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';

import { fonts } from '../theme/tokens';

const evidenceRoomImage = '/media/caliber-evidence-room.png';

interface PublicAuthFrameProps {
  children: ReactNode;
  eyebrow: string;
  proofTitle: string;
  proofBody: string;
}

export function PublicAuthFrame({ children, eyebrow, proofTitle, proofBody }: PublicAuthFrameProps) {
  const { t } = useTranslation();

  return (
    <Box
      sx={{
        mx: 'auto',
        maxWidth: 1120,
        display: 'grid',
        gridTemplateColumns: { xs: '1fr', md: 'minmax(0, 0.9fr) minmax(420px, 0.8fr)' },
        gap: { xs: 3, md: 4 },
        alignItems: 'stretch',
      }}
    >
      <Paper
        variant="outlined"
        sx={{
          p: { xs: 3, sm: 4, md: 5 },
          borderRadius: 2,
          borderColor: 'rgba(17, 20, 24, 0.12)',
          boxShadow: '0 24px 70px rgba(17, 20, 24, 0.08)',
          bgcolor: 'background.paper',
        }}
      >
        <Stack spacing={3.5}>
          <Button
            component={Link}
            to="/"
            variant="text"
            startIcon={<ArrowBackRounded />}
            sx={{ alignSelf: 'flex-start', px: 0 }}
          >
            {t('auth.backOverview')}
          </Button>
          {children}
        </Stack>
      </Paper>

      <Box
        sx={{
          position: 'relative',
          minHeight: { xs: 360, md: '100%' },
          overflow: 'hidden',
          borderRadius: 2,
          border: 1,
          borderColor: 'rgba(255,255,255,0.2)',
          backgroundImage: `linear-gradient(180deg, rgba(10, 17, 30, 0.18), rgba(10, 17, 30, 0.9)), url(${evidenceRoomImage})`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          color: '#fff',
          boxShadow: '0 24px 80px rgba(6, 11, 20, 0.24)',
        }}
      >
        <Stack
          sx={{
            minHeight: '100%',
            justifyContent: 'space-between',
            p: { xs: 3, md: 4 },
            gap: 4,
          }}
        >
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
            <Box
              sx={{
                display: 'grid',
                placeItems: 'center',
                width: 36,
                height: 36,
                borderRadius: 1.5,
                bgcolor: 'rgba(45, 212, 191, 0.16)',
                border: '1px solid rgba(45, 212, 191, 0.34)',
              }}
            >
              <ShieldOutlined fontSize="small" />
            </Box>
            <Typography sx={{ fontFamily: fonts.mono, fontSize: 12, letterSpacing: 0, textTransform: 'uppercase' }}>
              {eyebrow}
            </Typography>
          </Stack>

          <Stack spacing={2.5} sx={{ maxWidth: 430 }}>
            <Typography
              component="p"
              sx={{
                fontFamily: fonts.title,
                fontSize: { xs: 34, md: 46 },
                lineHeight: 1.04,
                fontWeight: 650,
              }}
            >
              {proofTitle}
            </Typography>
            <Typography component="p" sx={{ color: 'rgba(255,255,255,0.78)', fontSize: 16, lineHeight: 1.7 }}>
              {proofBody}
            </Typography>
          </Stack>

          <Stack
            direction={{ xs: 'column', sm: 'row', md: 'column', lg: 'row' }}
            spacing={1.5}
            sx={{ alignItems: { xs: 'stretch', sm: 'center', md: 'stretch', lg: 'center' } }}
          >
            {[
              [t('auth.proofBadge1Title'), t('auth.proofBadge1Body')],
              [t('auth.proofBadge2Title'), t('auth.proofBadge2Body')],
            ].map(([label, body]) => (
              <Box
                key={label}
                sx={{
                  p: 2,
                  flex: 1,
                  borderRadius: 2,
                  bgcolor: 'rgba(255,255,255,0.1)',
                  border: '1px solid rgba(255,255,255,0.16)',
                  backdropFilter: 'blur(12px)',
                }}
              >
                <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 0.75 }}>
                  <AutoAwesomeOutlined sx={{ fontSize: 18, color: '#FBBF24' }} />
                  <Typography sx={{ fontFamily: fonts.mono, fontSize: 12, letterSpacing: 0 }}>{label}</Typography>
                </Stack>
                <Typography variant="body2" sx={{ color: 'rgba(255,255,255,0.72)' }}>
                  {body}
                </Typography>
              </Box>
            ))}
          </Stack>
        </Stack>
      </Box>
    </Box>
  );
}
