import { Alert, Button, Card, CardContent, Chip, Stack, Typography } from '@mui/material';
import { Link } from 'react-router-dom';

import { CardListSkeleton } from '../components/Skeletons';
import { seniorityLabel } from '../lib/format';
import { useRoles } from '../query/flow';
import { useAuthStore } from '../stores/auth';

export function RolesPage() {
  const employerId = useAuthStore((s) => s.user?.id);
  const roles = useRoles(employerId);

  return (
    <Stack spacing={4} sx={{ maxWidth: 820, mx: 'auto' }}>
      <Stack direction="row" spacing={2} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <Stack spacing={1} sx={{ flexGrow: 1 }}>
          <Typography variant="h3" component="h1">Your roles</Typography>
          <Typography color="text.secondary">Every role you have described, with its spec and rubric.</Typography>
        </Stack>
        <Button component={Link} to="/roles/new" variant="contained">
          Describe a role
        </Button>
      </Stack>

      {roles.isPending && employerId ? (
        <CardListSkeleton count={2} />
      ) : roles.isError ? (
        <Alert severity="info">{roles.error instanceof Error ? roles.error.message : 'Could not load roles.'}</Alert>
      ) : (roles.data?.roles ?? []).length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          No roles yet. Describe one to get started.
        </Typography>
      ) : (
        <Stack spacing={2}>
          {(roles.data?.roles ?? []).map((r) => (
            <Card key={r.id} variant="outlined">
              <CardContent>
                <Stack direction="row" spacing={2} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
                  <Stack spacing={0.5} sx={{ flexGrow: 1 }}>
                    <Typography variant="h6" component="h2">{r.title || r.spec.title}</Typography>
                    <Stack direction="row" spacing={1}>
                      <Chip size="small" label={seniorityLabel[r.spec.seniority]} />
                      {r.spec.location && <Chip size="small" variant="outlined" label={r.spec.location} />}
                      <Chip size="small" variant="outlined" label={`${r.rubric.competencies.length} competencies`} />
                    </Stack>
                  </Stack>
                  <Button component={Link} to={`/interview?roleId=${r.id}`} variant="outlined" size="small">
                    Interview
                  </Button>
                </Stack>
              </CardContent>
            </Card>
          ))}
        </Stack>
      )}
    </Stack>
  );
}
