import ArrowForwardRounded from '@mui/icons-material/ArrowForwardRounded';
import FactCheckOutlined from '@mui/icons-material/FactCheckOutlined';
import InsightsOutlined from '@mui/icons-material/InsightsOutlined';
import LoginRounded from '@mui/icons-material/LoginRounded';
import PsychologyAltOutlined from '@mui/icons-material/PsychologyAltOutlined';
import ShieldOutlined from '@mui/icons-material/ShieldOutlined';
import WorkHistoryOutlined from '@mui/icons-material/WorkHistoryOutlined';
import { Box, Button, Chip, Container, Stack, Typography } from '@mui/material';
import { motion, useReducedMotion } from 'motion/react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';

import { fonts } from '../theme/tokens';

const evidenceRoomImage = '/media/caliber-evidence-room.png';

export function LandingPage() {
  const { t } = useTranslation();
  const reduceMotion = useReducedMotion();

  const features = [
    { title: t('landing.feature1Title'), body: t('landing.feature1Body'), Icon: FactCheckOutlined, accent: '#2DD4BF' },
    { title: t('landing.feature2Title'), body: t('landing.feature2Body'), Icon: PsychologyAltOutlined, accent: '#F59E0B' },
    { title: t('landing.feature3Title'), body: t('landing.feature3Body'), Icon: WorkHistoryOutlined, accent: '#60A5FA' },
  ];
  const stats = [
    { value: '3', label: t('landing.stat1') },
    { value: '0', label: t('landing.stat2') },
    { value: '100%', label: t('landing.stat3') },
  ];

  return (
    <Box sx={{ position: 'relative' }}>
      <Box
        component="section"
        sx={{
          mx: 'calc(50% - 50vw)',
          mt: { xs: -3, md: -5 },
          minHeight: { xs: 'min(680px, calc(100svh - 92px))', md: 'min(760px, calc(100svh - 132px))' },
          display: 'flex',
          alignItems: 'center',
          position: 'relative',
          overflow: 'hidden',
          backgroundImage: `linear-gradient(90deg, rgba(7, 12, 22, 0.94) 0%, rgba(7, 12, 22, 0.78) 47%, rgba(7, 12, 22, 0.28) 100%), url(${evidenceRoomImage})`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
          color: '#fff',
        }}
      >
        <Container maxWidth="lg" sx={{ py: { xs: 7, md: 9 }, position: 'relative' }}>
          <Stack spacing={{ xs: 3, md: 4 }} sx={{ alignItems: 'flex-start', maxWidth: 780 }}>
            <Chip
              icon={<ShieldOutlined />}
              label={t('landing.eyebrow')}
              sx={{
                color: '#DFFCF8',
                bgcolor: 'rgba(45, 212, 191, 0.14)',
                border: '1px solid rgba(45, 212, 191, 0.32)',
                '.MuiChip-icon': { color: '#2DD4BF' },
              }}
            />
            <motion.div initial={reduceMotion ? false : { opacity: 0, y: 24 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5 }}>
              <Typography
                component="h1"
                sx={{
                  fontFamily: fonts.title,
                  fontWeight: 720,
                  fontSize: { xs: 44, sm: 58, md: 82 },
                  lineHeight: 1,
                  maxWidth: 800,
                }}
              >
                {t('landing.headline')}
              </Typography>
            </motion.div>
            <motion.div
              initial={reduceMotion ? false : { opacity: 0, y: 24 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.08 }}
            >
              <Typography component="p" variant="h6" sx={{ maxWidth: 670, color: 'rgba(255,255,255,0.78)', fontWeight: 400, lineHeight: 1.65 }}>
                {t('landing.subheadline')}
              </Typography>
            </motion.div>
            <Stack direction="row" spacing={1.5} useFlexGap sx={{ flexWrap: 'wrap' }}>
              <Button component={Link} to="/register" variant="contained" size="large" endIcon={<ArrowForwardRounded />}>
                {t('landing.ctaPrimary')}
              </Button>
              <Button
                component={Link}
                to="/login"
                variant="outlined"
                size="large"
                startIcon={<LoginRounded />}
                sx={{ color: '#fff', borderColor: 'rgba(255,255,255,0.48)', '&:hover': { borderColor: '#fff', bgcolor: 'rgba(255,255,255,0.08)' } }}
              >
                {t('landing.ctaSecondary')}
              </Button>
            </Stack>
            <Stack direction="row" spacing={{ xs: 2, md: 4 }} useFlexGap sx={{ flexWrap: 'wrap', pt: 1 }}>
              {stats.map((s) => (
                <Box key={s.label}>
                  <Typography sx={{ fontFamily: fonts.mono, fontSize: { xs: 28, md: 36 }, lineHeight: 1, color: '#FBBF24' }}>
                    {s.value}
                  </Typography>
                  <Typography variant="body2" sx={{ color: 'rgba(255,255,255,0.68)', mt: 0.75 }}>
                    {s.label}
                  </Typography>
                </Box>
              ))}
            </Stack>
          </Stack>
        </Container>
      </Box>

      <Container maxWidth="lg" sx={{ position: 'relative', pt: { xs: 6, md: 8 }, pb: { xs: 3, md: 6 } }}>
        <Stack spacing={2.5} sx={{ maxWidth: 760, mb: { xs: 4, md: 5 } }}>
          <Typography sx={{ fontFamily: fonts.mono, fontSize: 12, letterSpacing: 0, color: 'primary.main', textTransform: 'uppercase' }}>
            {t('landing.systemLabel')}
          </Typography>
          <Typography component="h2" sx={{ fontFamily: fonts.title, fontWeight: 650, fontSize: { xs: 34, md: 52 }, lineHeight: 1.05 }}>
            {t('landing.systemTitle')}
          </Typography>
          <Typography color="text.secondary" sx={{ fontSize: { xs: 16, md: 18 }, lineHeight: 1.75 }}>
            {t('landing.systemBody')}
          </Typography>
        </Stack>

        <Box component="section" sx={{ display: 'grid', gap: 2.5, gridTemplateColumns: { xs: '1fr', md: '1fr 1fr 1fr' } }}>
          {features.map(({ Icon, ...f }, i) => (
            <motion.div
              key={f.title}
              initial={reduceMotion ? false : { opacity: 0, y: 28 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, amount: 0.4 }}
              transition={{ duration: 0.5, delay: i * 0.08 }}
            >
              <Box
                sx={{
                  p: 3,
                  height: '100%',
                  minHeight: 260,
                  border: 1,
                  borderColor: 'divider',
                  borderRadius: 2,
                  bgcolor: 'background.paper',
                  boxShadow: '0 18px 52px rgba(17, 20, 24, 0.06)',
                }}
              >
                <Box
                  sx={{
                    display: 'grid',
                    placeItems: 'center',
                    width: 44,
                    height: 44,
                    borderRadius: 1.5,
                    mb: 2.5,
                    color: f.accent,
                    bgcolor: `${f.accent}18`,
                  }}
                >
                  <Icon />
                </Box>
                <Typography variant="h5" component="h3" gutterBottom>
                  {f.title}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {f.body}
                </Typography>
              </Box>
            </motion.div>
          ))}
        </Box>

        <Box
          component="section"
          sx={{
            mt: { xs: 3, md: 4 },
            p: { xs: 3, md: 4 },
            display: 'grid',
            gap: 2,
            gridTemplateColumns: { xs: '1fr', md: '1fr auto' },
            alignItems: 'center',
            borderRadius: 2,
            border: 1,
            borderColor: 'divider',
            bgcolor: 'background.paper',
          }}
        >
          <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
            <Box sx={{ display: 'grid', placeItems: 'center', width: 48, height: 48, borderRadius: 1.5, bgcolor: 'rgba(245, 158, 11, 0.14)', color: '#B45309' }}>
              <InsightsOutlined />
            </Box>
            <Box>
              <Typography component="h2" variant="h5">
                {t('landing.closeTitle')}
              </Typography>
              <Typography color="text.secondary">{t('landing.closeBody')}</Typography>
            </Box>
          </Stack>
          <Button component={Link} to="/register" variant="contained" endIcon={<ArrowForwardRounded />} sx={{ justifySelf: { xs: 'stretch', md: 'end' } }}>
            {t('landing.closeCta')}
          </Button>
        </Box>
      </Container>
    </Box>
  );
}
