import DonutSmallOutlined from '@mui/icons-material/DonutSmallOutlined';
import WorkOutlineRounded from '@mui/icons-material/WorkOutlineRounded';
import { Box, Card, CardContent, Chip, LinearProgress, Stack, Typography } from '@mui/material';
import { alpha } from '@mui/material/styles';

import type { SupplyDemandItem } from '../../api/types';
import { fonts } from '../../theme/tokens';

export function SupplyDemandPanel({ items }: { items: SupplyDemandItem[] }) {
  const max = Math.max(1, ...items.map((i) => Math.max(i.openRoles, i.availableCandidates)));
  return (
    <Card variant="outlined" sx={{ borderRadius: '8px', bgcolor: 'background.paper', overflow: 'hidden', height: '100%' }}>
      <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
        <Box
          sx={(theme) => ({
            p: { xs: 2.25, md: 2.75 },
            borderBottom: '1px solid',
            borderColor: 'divider',
            background: `linear-gradient(135deg, ${alpha(theme.palette.warning.main, 0.13)}, transparent 52%)`,
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
                bgcolor: 'rgba(245, 158, 11, 0.14)',
                color: '#F59E0B',
                flex: '0 0 auto',
              }}
            >
              <DonutSmallOutlined />
            </Box>
            <Box>
              <Typography component="h2" sx={{ fontFamily: fonts.body, fontWeight: 850, fontSize: { xs: 22, md: 24 }, lineHeight: 1.1 }}>
                Supply &amp; demand
              </Typography>
              <Typography sx={{ mt: 0.5, color: 'text.secondary', fontSize: 14, lineHeight: 1.45 }}>
                Open role pressure against available verified candidates.
              </Typography>
            </Box>
          </Stack>
        </Box>

        <Stack spacing={1.2} sx={{ p: { xs: 1.5, md: 2 } }}>
          {items.length === 0 && (
            <Box sx={{ border: '1px dashed', borderColor: 'divider', borderRadius: '8px', p: 2, bgcolor: 'rgba(255, 255, 255, 0.03)' }}>
              <Typography variant="body2" color="text.secondary">
                No open roles yet.
              </Typography>
            </Box>
          )}
          {items.map((it) => (
            <Box
              key={it.roleFamily}
              sx={(theme) => ({
                border: '1px solid',
                borderColor: 'divider',
                borderRadius: '8px',
                p: 1.5,
                bgcolor: alpha(theme.palette.background.default, 0.34),
              })}
            >
              <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 1.2, flexWrap: 'wrap', gap: 1 }}>
                <Stack direction="row" spacing={1.1} sx={{ alignItems: 'center', minWidth: 0 }}>
                  <Box
                    sx={{
                      width: 34,
                      height: 34,
                      borderRadius: '8px',
                      display: 'grid',
                      placeItems: 'center',
                      bgcolor: 'rgba(77, 155, 255, 0.12)',
                      color: 'primary.main',
                      flex: '0 0 auto',
                    }}
                  >
                    <WorkOutlineRounded fontSize="small" />
                  </Box>
                  <Box sx={{ minWidth: 0 }}>
                    <Typography variant="body2" sx={{ fontFamily: fonts.body, fontWeight: 850, textTransform: 'capitalize' }}>
                      {it.roleFamily}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {it.openRoles} open · {it.availableCandidates} candidates · gap {it.gap}
                    </Typography>
                  </Box>
                </Stack>
                <Chip
                  size="small"
                  color={it.gap > 0 ? 'warning' : 'success'}
                  label={it.gap > 0 ? 'Talent gap' : 'Covered'}
                  sx={{ borderRadius: '999px', fontFamily: fonts.body, fontWeight: 800 }}
                />
              </Stack>
              <Stack spacing={0.8}>
                <Box>
                  <Stack direction="row" sx={{ justifyContent: 'space-between', mb: 0.35 }}>
                    <Typography sx={{ color: 'text.secondary', fontSize: 11, fontWeight: 800 }}>Open roles</Typography>
                    <Typography sx={{ fontFamily: fonts.mono, fontSize: 12 }}>{it.openRoles}</Typography>
                  </Stack>
                  <LinearProgress variant="determinate" color="warning" value={(it.openRoles / max) * 100} sx={{ height: 7, borderRadius: '999px' }} />
                </Box>
                <Box>
                  <Stack direction="row" sx={{ justifyContent: 'space-between', mb: 0.35 }}>
                    <Typography sx={{ color: 'text.secondary', fontSize: 11, fontWeight: 800 }}>Available candidates</Typography>
                    <Typography sx={{ fontFamily: fonts.mono, fontSize: 12 }}>{it.availableCandidates}</Typography>
                  </Stack>
                  <LinearProgress
                    variant="determinate"
                    color="primary"
                    value={(it.availableCandidates / max) * 100}
                    sx={{ height: 7, borderRadius: '999px' }}
                  />
                </Box>
              </Stack>
            </Box>
          ))}
        </Stack>
      </CardContent>
    </Card>
  );
}
