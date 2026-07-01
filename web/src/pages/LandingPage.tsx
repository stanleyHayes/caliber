import { Box, Button, Container, Stack, Typography } from '@mui/material';
import { motion, useScroll, useTransform } from 'motion/react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';

import { fonts } from '../theme/tokens';

const MotionBox = motion.create(Box);

export function LandingPage() {
  const { t } = useTranslation();
  const { scrollY } = useScroll();
  const blobY = useTransform(scrollY, [0, 600], [0, 160]);
  const blobY2 = useTransform(scrollY, [0, 600], [0, -120]);

  const features = [
    { title: t('landing.feature1Title'), body: t('landing.feature1Body') },
    { title: t('landing.feature2Title'), body: t('landing.feature2Body') },
    { title: t('landing.feature3Title'), body: t('landing.feature3Body') },
  ];

  return (
    <Box sx={{ position: 'relative', overflow: 'hidden' }}>
      <MotionBox
        aria-hidden
        style={{ y: blobY }}
        sx={{
          position: 'absolute', top: -120, right: -80, width: 360, height: 360, borderRadius: '50%',
          bgcolor: 'primary.main', opacity: 0.14, filter: 'blur(40px)', pointerEvents: 'none',
        }}
      />
      <MotionBox
        aria-hidden
        style={{ y: blobY2 }}
        sx={{
          position: 'absolute', top: 240, left: -120, width: 300, height: 300, borderRadius: '50%',
          bgcolor: 'secondary.main', opacity: 0.1, filter: 'blur(48px)', pointerEvents: 'none',
        }}
      />

      <Container maxWidth="md" sx={{ position: 'relative', py: { xs: 8, md: 14 } }}>
        <Stack component="section" spacing={4} sx={{ alignItems: 'flex-start' }}>
          <motion.div initial={{ opacity: 0, y: 24 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5 }}>
            <Typography component="h1" sx={{ fontFamily: fonts.title, fontWeight: 700, fontSize: { xs: 44, md: 72 }, lineHeight: 1.05 }}>
              {t('landing.headline')}
            </Typography>
          </motion.div>
          <motion.div initial={{ opacity: 0, y: 24 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5, delay: 0.1 }}>
            <Typography component="p" variant="h6" color="text.secondary" sx={{ maxWidth: 620, fontWeight: 400 }}>
              {t('landing.subheadline')}
            </Typography>
          </motion.div>
          <Stack direction="row" spacing={2} useFlexGap sx={{ flexWrap: 'wrap' }}>
            <Button component={Link} to="/register" variant="contained" size="large">
              {t('landing.ctaPrimary')}
            </Button>
            <Button component={Link} to="/login" variant="outlined" size="large">
              {t('landing.ctaSecondary')}
            </Button>
          </Stack>
        </Stack>

        <Box component="section" sx={{ mt: { xs: 8, md: 14 }, display: 'grid', gap: 3, gridTemplateColumns: { xs: '1fr', md: '1fr 1fr 1fr' } }}>
          {features.map((f, i) => (
            <motion.div
              key={f.title}
              initial={{ opacity: 0, y: 40, rotateX: -12 }}
              whileInView={{ opacity: 1, y: 0, rotateX: 0 }}
              viewport={{ once: true, amount: 0.4 }}
              transition={{ duration: 0.5, delay: i * 0.08 }}
              style={{ transformPerspective: 800 }}
            >
              <Box sx={{ p: 3, height: '100%', border: 1, borderColor: 'divider', borderRadius: 3, bgcolor: 'background.paper' }}>
                <Typography variant="h6" component="h2" gutterBottom>
                  {f.title}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {f.body}
                </Typography>
              </Box>
            </motion.div>
          ))}
        </Box>
      </Container>
    </Box>
  );
}
