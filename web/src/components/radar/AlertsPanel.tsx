import FactCheckOutlined from '@mui/icons-material/FactCheckOutlined';
import NotificationsActiveOutlined from '@mui/icons-material/NotificationsActiveOutlined';
import PersonSearchOutlined from '@mui/icons-material/PersonSearchOutlined';
import WorkOutlineRounded from '@mui/icons-material/WorkOutlineRounded';
import { Box, Card, CardContent, Chip, LinearProgress, Stack, Typography } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { AnimatePresence, motion } from 'motion/react';

import type { AlertType, MatchAlert } from '../../api/types';
import { shortId } from '../../lib/format';
import { PageControls } from '../PageControls';
import { fonts } from '../../theme/tokens';

const alertLabels: Record<AlertType, string> = {
  ALERT_TYPE_UNSPECIFIED: 'Alert',
  ALERT_TYPE_CANDIDATE_FOR_ROLE: 'Candidate for role',
  ALERT_TYPE_ROLE_FOR_CANDIDATE: 'Role for candidate',
};

function alertIcon(type: AlertType) {
  if (type === 'ALERT_TYPE_CANDIDATE_FOR_ROLE') return <PersonSearchOutlined fontSize="small" />;
  if (type === 'ALERT_TYPE_ROLE_FOR_CANDIDATE') return <WorkOutlineRounded fontSize="small" />;
  return <NotificationsActiveOutlined fontSize="small" />;
}

function extractPercent(message: string) {
  const match = message.match(/(\d{1,3})%/);
  if (!match) return null;
  return Math.max(0, Math.min(100, Number(match[1])));
}

export function AlertsPanel({
  alerts,
  page,
  pageCount,
  onPageChange,
}: {
  alerts: MatchAlert[];
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
            background: `linear-gradient(135deg, ${alpha(theme.palette.info.main, 0.16)}, transparent 50%)`,
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
                bgcolor: 'rgba(45, 212, 191, 0.16)',
                color: '#2DD4BF',
                flex: '0 0 auto',
              }}
            >
              <NotificationsActiveOutlined />
            </Box>
            <Box sx={{ minWidth: 0, flex: 1 }}>
              <Typography component="h2" sx={{ fontFamily: fonts.body, fontWeight: 850, fontSize: { xs: 22, md: 24 }, lineHeight: 1.1 }}>
                Match alerts
              </Typography>
              <Typography sx={{ mt: 0.5, color: 'text.secondary', fontSize: 14, lineHeight: 1.45 }}>
                New role-candidate movements with their match evidence kept visible.
              </Typography>
            </Box>
            <Chip
              icon={<FactCheckOutlined />}
              label={`${alerts.length} signals`}
              size="small"
              variant="outlined"
              sx={{ display: { xs: 'none', sm: 'inline-flex' }, borderRadius: '999px', fontFamily: fonts.body, fontWeight: 800 }}
            />
          </Stack>
        </Box>

        <Stack spacing={1.1} sx={{ p: { xs: 1.5, md: 2 } }}>
          {alerts.length === 0 && (
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
                No alerts yet.
              </Typography>
            </Box>
          )}
          <AnimatePresence mode="popLayout">
            {alerts.map((alert) => {
              const score = extractPercent(alert.message);

              return (
                <motion.div
                  key={alert.id}
                  layout
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -8 }}
                  transition={{ duration: 0.18, ease: 'easeOut' }}
                >
                  <Box
                    sx={(theme) => ({
                      display: 'grid',
                      gridTemplateColumns: { xs: '42px minmax(0, 1fr)', md: '42px minmax(0, 1fr) minmax(104px, 132px)' },
                      gap: 1.5,
                      alignItems: 'center',
                      border: '1px solid',
                      borderColor: 'divider',
                      borderRadius: '8px',
                      p: { xs: 1.25, sm: 1.5 },
                      bgcolor: alpha(theme.palette.background.default, 0.34),
                      transition: 'border-color 0.18s ease, transform 0.18s ease, box-shadow 0.18s ease',
                      '&:hover': {
                        borderColor: alpha(theme.palette.info.main, 0.54),
                        transform: 'translateY(-2px)',
                        boxShadow: `0 16px 42px ${alpha(theme.palette.common.black, 0.18)}`,
                      },
                    })}
                  >
                    <Box
                      sx={{
                        width: 42,
                        height: 42,
                        borderRadius: '8px',
                        display: 'grid',
                        placeItems: 'center',
                        color: 'primary.main',
                        bgcolor: 'rgba(77, 155, 255, 0.14)',
                      }}
                    >
                      {alertIcon(alert.type)}
                    </Box>

                    <Box sx={{ minWidth: 0 }}>
                      <Stack direction="row" spacing={1} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap', mb: 0.75 }}>
                        <Chip
                          size="small"
                          color="info"
                          label={alertLabels[alert.type] ?? alertLabels.ALERT_TYPE_UNSPECIFIED}
                          sx={{ borderRadius: '999px', fontFamily: fonts.body, fontWeight: 800 }}
                        />
                        <Typography sx={{ fontFamily: fonts.mono, color: 'text.secondary', fontSize: 11 }}>
                          {shortId(alert.id)}
                        </Typography>
                      </Stack>
                      <Typography sx={{ fontFamily: fonts.body, fontWeight: 750, lineHeight: 1.4 }}>{alert.message}</Typography>
                    </Box>

                    {score !== null && (
                      <Box sx={{ gridColumn: { xs: '2 / -1', md: 'auto' } }}>
                        <Stack direction="row" sx={{ justifyContent: 'space-between', mb: 0.45 }}>
                          <Typography sx={{ color: 'text.secondary', fontSize: 11, fontWeight: 800 }}>Match</Typography>
                          <Typography sx={{ fontFamily: fonts.mono, fontSize: 13, fontWeight: 850 }}>{score}%</Typography>
                        </Stack>
                        <LinearProgress
                          aria-label={`${alert.id} match score`}
                          variant="determinate"
                          color={score >= 85 ? 'success' : 'primary'}
                          value={score}
                          sx={{ height: 7, borderRadius: '999px', bgcolor: 'rgba(148, 163, 184, 0.18)' }}
                        />
                      </Box>
                    )}
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
