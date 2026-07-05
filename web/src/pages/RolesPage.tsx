import ArrowForwardRounded from '@mui/icons-material/ArrowForwardRounded';
import { Alert, Box, Button, Card, CardContent, Chip, Stack, Typography } from '@mui/material';
import { useState } from 'react';
import { Link } from 'react-router-dom';

import { PageControls } from '../components/PageControls';
import { PageBackButton } from '../components/PageBackButton';
import { PermissionRedirect } from '../components/PermissionRedirect';
import { CardListSkeleton } from '../components/Skeletons';
import { roleStatusLabel, seniorityLabel } from '../lib/format';
import { canManageRoles, canScreenSelf } from '../lib/permissions';
import { useRoles } from '../query/flow';
import { useAuthStore } from '../stores/auth';
import { fonts } from '../theme/tokens';

const PAGE_SIZE = 20;
const roleChipSx = {
  borderRadius: '999px',
  fontFamily: fonts.body,
  fontWeight: 700,
  height: 30,
  '& .MuiChip-label': { px: 1.25 },
};

export function RolesPage() {
  const employerId = useAuthStore((s) => s.user?.id);
  const role = useAuthStore((s) => s.user?.role);
  const canUseRoles = canManageRoles(role);
  const showInterviewAction = canScreenSelf(role);
  const [page, setPage] = useState(1);
  const roles = useRoles(canUseRoles ? employerId : undefined, page, PAGE_SIZE);

  if (!canUseRoles) {
    return <PermissionRedirect />;
  }

  return (
    <Stack spacing={4} sx={{ maxWidth: 960, mx: 'auto' }}>
      <PageBackButton />
      <Stack direction="row" spacing={2} useFlexGap sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
        <Stack spacing={1} sx={{ flexGrow: 1 }}>
          <Typography variant="h1" component="h1" sx={{ fontSize: 42, lineHeight: 1.05 }}>
            Your roles
          </Typography>
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
          {(roles.data?.roles ?? []).map((r) => {
            const title = r.title || r.spec.title;
            const competencyCount = r.rubric.competencies.length;

            return (
              <Card
                key={r.id}
                variant="outlined"
                sx={{
                  borderRadius: '8px',
                  overflow: 'hidden',
                  bgcolor: 'background.paper',
                  boxShadow: '0 18px 45px rgba(17, 24, 39, 0.05)',
                  transition: 'border-color 160ms ease, box-shadow 160ms ease, transform 160ms ease',
                  '&:hover': {
                    borderColor: 'primary.main',
                    boxShadow: '0 22px 55px rgba(0, 102, 204, 0.12)',
                    transform: 'translateY(-1px)',
                  },
                }}
              >
                <CardContent sx={{ p: { xs: 2.25, sm: 3 }, '&:last-child': { pb: { xs: 2.25, sm: 3 } } }}>
                  <Stack direction={{ xs: 'column', sm: 'row' }} spacing={{ xs: 2, sm: 3 }} sx={{ alignItems: { xs: 'stretch', sm: 'center' } }}>
                    <Box
                      aria-hidden
                      sx={{
                        display: { xs: 'none', sm: 'block' },
                        width: 4,
                        alignSelf: 'stretch',
                        borderRadius: '999px',
                        bgcolor: r.status === 'ROLE_STATUS_OPEN' ? 'primary.main' : 'divider',
                      }}
                    />
                    <Stack spacing={1.35} sx={{ flexGrow: 1, minWidth: 0 }}>
                      <Chip
                        size="small"
                        label={roleStatusLabel[r.status]}
                        color={r.status === 'ROLE_STATUS_OPEN' ? 'primary' : 'default'}
                        variant={r.status === 'ROLE_STATUS_OPEN' ? 'filled' : 'outlined'}
                        sx={{ ...roleChipSx, alignSelf: 'flex-start', fontWeight: 800, height: 26 }}
                      />
                      <Typography
                        variant="h5"
                        component="h2"
                        sx={{
                          fontFamily: fonts.body,
                          fontSize: 24,
                          fontWeight: 800,
                          lineHeight: 1.15,
                          letterSpacing: 0,
                          color: 'text.primary',
                        }}
                      >
                        {title}
                      </Typography>
                      <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap' }}>
                        <Chip size="small" label={seniorityLabel[r.spec.seniority]} sx={roleChipSx} />
                        {r.spec.location && <Chip size="small" variant="outlined" label={r.spec.location} sx={roleChipSx} />}
                        {r.spec.availability && <Chip size="small" variant="outlined" label={r.spec.availability} sx={roleChipSx} />}
                        <Chip
                          size="small"
                          variant="outlined"
                          label={`${competencyCount} ${competencyCount === 1 ? 'competency' : 'competencies'}`}
                          sx={roleChipSx}
                        />
                      </Stack>
                    </Stack>
                    {showInterviewAction && (
                      <Button
                        component={Link}
                        to={`/interview?roleId=${r.id}`}
                        variant="outlined"
                        size="medium"
                        endIcon={<ArrowForwardRounded fontSize="small" />}
                        sx={{ alignSelf: { xs: 'stretch', sm: 'center' }, minHeight: 44, borderRadius: '8px', px: 2.25, fontWeight: 800 }}
                      >
                        Interview
                      </Button>
                    )}
                  </Stack>
                </CardContent>
              </Card>
            );
          })}
          {roles.data?.page && (
            <PageControls page={roles.data.page.page || page} pageCount={roles.data.page.totalPages} onChange={setPage} />
          )}
        </Stack>
      )}
    </Stack>
  );
}
