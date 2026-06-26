import { Box, Card, CardContent, Chip, Divider, LinearProgress, Stack, Typography } from '@mui/material';

import type { InterviewReportCard } from '../../api/types';
import { confidenceColor, confidenceLabel, verdictColor, verdictLabel } from '../../lib/format';

export function ReportCardView({ report }: { report: InterviewReportCard }) {
  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={2.5}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
            <Typography variant="h5" sx={{ flexGrow: 1 }}>
              Report card
            </Typography>
            <Chip color={verdictColor[report.verdict]} label={verdictLabel[report.verdict]} />
            <Chip variant="outlined" color={confidenceColor[report.confidence]} label={`${confidenceLabel[report.confidence]} confidence`} />
          </Stack>

          <Divider textAlign="left">
            <Typography variant="caption" color="text.secondary">
              Per-competency — every score cites a transcript quote
            </Typography>
          </Divider>

          {report.scores.length === 0 && (
            <Typography variant="body2" color="text.secondary">
              No per-competency scores were produced.
            </Typography>
          )}

          <Stack spacing={1.75}>
            {report.scores.map((sc, i) => (
              <Box key={i}>
                <Stack direction="row" sx={{ justifyContent: 'space-between', mb: 0.5 }}>
                  <Typography variant="body2">{sc.competency}</Typography>
                  <Typography variant="caption" color="text.secondary">
                    {sc.score.toFixed(1)} / 5
                  </Typography>
                </Stack>
                <LinearProgress variant="determinate" value={(sc.score / 5) * 100} sx={{ height: 6, borderRadius: 1 }} />
                {sc.evidence && (
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                    “{sc.evidence}”
                  </Typography>
                )}
              </Box>
            ))}
          </Stack>

          {report.recommendedNextStep && (
            <Box>
              <Typography variant="overline" color="text.secondary">
                Recommended next step
              </Typography>
              <Typography variant="body1">{report.recommendedNextStep}</Typography>
            </Box>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}
