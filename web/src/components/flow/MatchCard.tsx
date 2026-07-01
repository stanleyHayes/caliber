import WarningAmberOutlined from '@mui/icons-material/WarningAmberOutlined';
import {
  Box,
  Card,
  CardContent,
  Chip,
  Divider,
  LinearProgress,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';

import type { Match } from '../../api/types';
import { confidenceColor, confidenceLabel, pct, shortId } from '../../lib/format';
import { fonts } from '../../theme/tokens';
import { DeclineCandidate } from './DeclineCandidate';

export function MatchCard({ match, rank }: { match: Match; rank: number }) {
  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={2}>
          <Stack direction="row" sx={{ alignItems: 'flex-start', justifyContent: 'space-between', flexWrap: 'wrap', gap: 1 }}>
            <Stack spacing={0.5}>
              <Typography variant="overline" color="text.secondary">
                #{rank} · candidate
              </Typography>
              <Typography sx={{ fontFamily: fonts.mono }}>{shortId(match.candidateId)}</Typography>
            </Stack>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
              {match.thinEvidence && (
                <Tooltip title="Sparse evidence — recommend a screening interview">
                  <Chip
                    size="small"
                    color="warning"
                    variant="outlined"
                    icon={<WarningAmberOutlined aria-hidden="true" />}
                    label="thin evidence"
                  />
                </Tooltip>
              )}
              <Chip size="small" color={confidenceColor[match.confidence]} label={confidenceLabel[match.confidence]} />
              <Box sx={{ textAlign: 'right' }}>
                <Typography variant="h5" component="span" sx={{ lineHeight: 1 }}>
                  {pct(match.overallScore)}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  fit
                </Typography>
              </Box>
            </Stack>
          </Stack>

          {match.rationale && (
            <Typography variant="body2" color="text.secondary">
              {match.rationale}
            </Typography>
          )}

          <Divider textAlign="left">
            <Typography variant="caption" color="text.secondary">
              Per-competency
            </Typography>
          </Divider>
          <Stack spacing={1.25}>
            {match.breakdown.map((b, i) => (
              <Box key={i}>
                <Stack direction="row" sx={{ mb: 0.5, justifyContent: 'space-between' }}>
                  <Typography variant="body2">{b.competency}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    {b.score.toFixed(1)} / 5
                  </Typography>
                </Stack>
                <LinearProgress variant="determinate" value={(b.score / 5) * 100} sx={{ height: 6, borderRadius: 1 }} />
                {b.evidence && (
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                    “{b.evidence}”
                  </Typography>
                )}
              </Box>
            ))}
          </Stack>

          {match.watchOuts.length > 0 && (
            <Box>
              <Typography variant="overline" color="text.secondary">
                Watch-outs
              </Typography>
              <Stack component="ul" sx={{ m: 0, pl: 2.5 }} spacing={0.25}>
                {match.watchOuts.map((w, i) => (
                  <Typography key={i} component="li" variant="body2" color="text.secondary">
                    {w}
                  </Typography>
                ))}
              </Stack>
            </Box>
          )}

          <Divider />
          {/* The decline is human-only and audited — never an AI auto-reject (CAL-081/094). */}
          <DeclineCandidate roleId={match.roleId} candidateId={match.candidateId} />
        </Stack>
      </CardContent>
    </Card>
  );
}
