import { Card, CardContent, Chip, Stack, Typography } from '@mui/material';

import type { AlertType, MatchAlert } from '../../api/types';
import { PageControls } from '../PageControls';

const alertLabels: Record<AlertType, string> = {
  ALERT_TYPE_UNSPECIFIED: 'Alert',
  ALERT_TYPE_CANDIDATE_FOR_ROLE: 'Candidate for role',
  ALERT_TYPE_ROLE_FOR_CANDIDATE: 'Role for candidate',
};

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
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={1.5}>
          <Typography variant="h6" component="h2">
            Match alerts
          </Typography>
          {alerts.length === 0 && (
            <Typography variant="body2" color="text.secondary">
              No alerts yet.
            </Typography>
          )}
          {alerts.map((alert) => (
            <Stack
              key={alert.id}
              direction="row"
              spacing={1}
              useFlexGap
              sx={{ alignItems: 'flex-start', flexWrap: 'wrap', rowGap: 0.5 }}
            >
              <Chip size="small" color="info" label={alertLabels[alert.type] ?? alertLabels.ALERT_TYPE_UNSPECIFIED} />
              <Typography variant="body2" sx={{ flex: 1 }}>
                {alert.message}
              </Typography>
            </Stack>
          ))}
          {pageCount !== undefined && pageCount > 1 && page !== undefined && onPageChange !== undefined && (
            <PageControls page={page} pageCount={pageCount} onChange={onPageChange} />
          )}
        </Stack>
      </CardContent>
    </Card>
  );
}
