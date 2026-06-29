import { Box, Button, Container, Stack, Typography } from '@mui/material';
import { motion, useScroll, useTransform } from 'motion/react';
import { Link } from 'react-router-dom';

import { fonts } from '../theme/tokens';

const MotionBox = motion.create(Box);

const FEATURES = [
  {
    title: 'Explainable shortlists',
    body: 'Describe a role in plain language. Get a structured spec, a weighted rubric, and a ranked shortlist where every score traces back to evidence.',
  },
  {
    title: 'AI screening interviews',
    body: 'An adaptive interviewer probes each competency and produces an evidence-tagged report card — no black box, no fabrication.',
  },
  {
    title: 'An honest candidate agent',
    body: 'It works while you sleep, applying only where your verified profile already qualifies you — never inventing a thing.',
  },
];

export function LandingPage() {
  const { scrollY } = useScroll();
  const blobY = useTransform(scrollY, [0, 600], [0, 160]);
  const blobY2 = useTransform(scrollY, [0, 600], [0, -120]);

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
        <Stack spacing={4} sx={{ alignItems: 'flex-start' }}>
          <motion.div initial={{ opacity: 0, y: 24 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5 }}>
            <Typography component="h1" sx={{ fontFamily: fonts.title, fontWeight: 700, fontSize: { xs: 44, md: 72 }, lineHeight: 1.05 }}>
              Hire on evidence, not guesswork.
            </Typography>
          </motion.div>
          <motion.div initial={{ opacity: 0, y: 24 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.5, delay: 0.1 }}>
            <Typography variant="h6" color="text.secondary" sx={{ maxWidth: 620, fontWeight: 400 }}>
              Caliber is a talent-intelligence platform: explainable shortlisting, adaptive AI screening,
              and an honest candidate agent — bias-safe and human-in-the-loop.
            </Typography>
          </motion.div>
          <Stack direction="row" spacing={2}>
            <Button component={Link} to="/register" variant="contained" size="large">
              Get started
            </Button>
            <Button component={Link} to="/login" variant="outlined" size="large">
              Sign in
            </Button>
          </Stack>
        </Stack>

        <Box sx={{ mt: { xs: 8, md: 14 }, display: 'grid', gap: 3, gridTemplateColumns: { xs: '1fr', md: '1fr 1fr 1fr' } }}>
          {FEATURES.map((f, i) => (
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
