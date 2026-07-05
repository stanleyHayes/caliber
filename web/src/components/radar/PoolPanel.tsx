import FactCheckOutlined from '@mui/icons-material/FactCheckOutlined';
import PersonSearchOutlined from '@mui/icons-material/PersonSearchOutlined';
import { Avatar, Box, Card, CardContent, Chip, LinearProgress, Stack, Typography } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { AnimatePresence, motion } from 'motion/react';

import type { PoolCandidate } from '../../api/types';
import { PageControls } from '../PageControls';
import { passportColor, passportLabel, pct, shortId } from '../../lib/format';
import { fonts } from '../../theme/tokens';

type ScoreColor = 'success' | 'primary' | 'warning' | 'error';

function initials(name: string, fallback: string) {
  const value = name.trim() || fallback;
  return value
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase())
    .join('');
}

function scoreColor(score: number): ScoreColor {
  if (score >= 0.85) return 'success';
  if (score >= 0.7) return 'primary';
  if (score >= 0.5) return 'warning';
  return 'error';
}

export function PoolPanel({
  candidates,
  page,
  pageCount,
  onPageChange,
}: {
  candidates: PoolCandidate[];
  page?: number;
  pageCount?: number;
  onPageChange?: (page: number) => void;
}) {
  return (
    <Card
      variant="outlined"
      sx={{
        borderRadius: '8px',
        bgcolor: 'background.paper',
        overflow: 'hidden',
        boxShadow: '0 18px 54px rgba(0, 0, 0, 0.16)',
      }}
    >
      <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
        <Box
          sx={(theme) => ({
            p: { xs: 2.25, md: 2.75 },
            borderBottom: '1px solid',
            borderColor: 'divider',
            background: `linear-gradient(135deg, ${alpha(theme.palette.primary.main, 0.14)}, transparent 48%)`,
          })}
        >
          <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
            <Box
              sx={{
                width: 44,
                height: 44,
                borderRadius: '8px',
                display: 'grid',
                placeItems: 'center',
                bgcolor: 'rgba(77, 155, 255, 0.14)',
                color: 'primary.main',
                flex: '0 0 auto',
              }}
            >
              <PersonSearchOutlined />
            </Box>
            <Box sx={{ minWidth: 0, flex: 1 }}>
              <Typography component="h2" sx={{ fontFamily: fonts.body, fontWeight: 850, fontSize: { xs: 22, md: 24 }, lineHeight: 1.1 }}>
                Live talent pool
              </Typography>
              <Typography sx={{ mt: 0.5, color: 'text.secondary', fontSize: 14, lineHeight: 1.45 }}>
                Ranked by current passport signal and headline match strength.
              </Typography>
            </Box>
            <Chip
              icon={<FactCheckOutlined />}
              label={`${candidates.length} visible`}
              size="small"
              variant="outlined"
              sx={{ display: { xs: 'none', sm: 'inline-flex' }, borderRadius: '999px', fontFamily: fonts.body, fontWeight: 800 }}
            />
          </Stack>
        </Box>

        <Stack spacing={1.1} sx={{ p: { xs: 1.5, md: 2 } }}>
          {candidates.length === 0 && (
            <Box
              sx={{
                border: '1px dashed',
                borderColor: 'divider',
                borderRadius: '8px',
                p: 2,
                bgcolor: 'rgba(255, 255, 255, 0.03)',
              }}
            >
              <Typography variant="body2" color="text.secondary">
                No candidates in the pool yet.
              </Typography>
            </Box>
          )}
          <AnimatePresence mode="popLayout">
            {candidates.map((c, index) => {
              const displayName = c.name || shortId(c.candidateId);
              const progress = Math.max(0, Math.min(100, c.headlineScore * 100));
              const tone = scoreColor(c.headlineScore);

              return (
                <motion.div
                  key={c.candidateId}
                  layout
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -8 }}
                  transition={{ duration: 0.18, ease: 'easeOut' }}
                >
                  <Box
                    sx={(theme) => ({
                      display: 'flex',
                      gap: 1.5,
                      alignItems: 'center',
                      border: '1px solid',
                      borderColor: 'divider',
                      borderRadius: '8px',
                      p: { xs: 1.25, sm: 1.5 },
                      bgcolor: alpha(theme.palette.background.default, 0.34),
                      transition: 'border-color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease',
                      '&:hover': {
                        borderColor: alpha(theme.palette.primary.main, 0.52),
                        transform: 'translateY(-2px)',
                        boxShadow: `0 16px 42px ${alpha(theme.palette.common.black, 0.18)}`,
                      },
                    })}
                  >
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, minWidth: 0, flex: '1 1 auto' }}>
                      <Typography
                        sx={{
                          width: 28,
                          flex: '0 0 auto',
                          fontFamily: fonts.mono,
                          color: 'text.secondary',
                          fontSize: 12,
                          fontWeight: 800,
                        }}
                      >
                        {String(index + 1).padStart(2, '0')}
                      </Typography>
                      <Avatar
                        variant="rounded"
                        sx={{
                          width: 42,
                          height: 42,
                          borderRadius: '8px',
                          bgcolor: 'rgba(45, 212, 191, 0.16)',
                          color: '#2DD4BF',
                          fontFamily: fonts.body,
                          fontSize: 14,
                          fontWeight: 850,
                        }}
                      >
                        {initials(c.name, shortId(c.candidateId))}
                      </Avatar>
                      <Box sx={{ minWidth: 0 }}>
                        <Typography sx={{ fontFamily: fonts.body, fontWeight: 850, fontSize: 16, lineHeight: 1.25 }} noWrap>
                          {displayName}
                        </Typography>
                        <Typography sx={{ mt: 0.25, fontFamily: fonts.mono, color: 'text.secondary', fontSize: 11 }}>
                          ID {shortId(c.candidateId)}
                        </Typography>
                      </Box>
                    </Box>

                    <Stack
                      direction={{ xs: 'column', sm: 'row' }}
                      spacing={1}
                      sx={{ alignItems: { xs: 'flex-end', sm: 'center' }, flex: '0 0 auto' }}
                    >
                      <Chip
                        size="small"
                        color={passportColor[c.passportStatus]}
                        label={passportLabel[c.passportStatus]}
                        sx={{ borderRadius: '999px', fontFamily: fonts.body, fontWeight: 800 }}
                      />
                      <Box sx={{ width: { xs: 86, sm: 118 } }}>
                        <Stack direction="row" sx={{ alignItems: 'baseline', justifyContent: 'space-between', mb: 0.45 }}>
                          <Typography sx={{ fontSize: 11, color: 'text.secondary', fontWeight: 800 }}>Signal</Typography>
                          <Typography sx={{ fontFamily: fonts.mono, fontSize: 13, fontWeight: 850 }}>{pct(c.headlineScore)}</Typography>
                        </Stack>
                        <LinearProgress
                          aria-label={`${displayName} headline score`}
                          variant="determinate"
                          color={tone}
                          value={progress}
                          sx={{ height: 7, borderRadius: '999px', bgcolor: 'rgba(148, 163, 184, 0.18)' }}
                        />
                      </Box>
                    </Stack>
                  </Box>
                </motion.div>
              );
            })}
          </AnimatePresence>
          {pageCount !== undefined && pageCount > 1 && page !== undefined && onPageChange !== undefined && (
            <PageControls page={page} pageCount={pageCount} onChange={onPageChange} />
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}
