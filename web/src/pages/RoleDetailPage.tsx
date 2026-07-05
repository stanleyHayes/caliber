import DeleteOutlineRounded from '@mui/icons-material/DeleteOutlineRounded';
import EditRounded from '@mui/icons-material/EditRounded';
import {
  Alert,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Stack,
  Typography,
} from '@mui/material';
import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import type { Role } from '../api/types';
import { DotsButton } from '../components/DotsButton';
import { PageBackButton } from '../components/PageBackButton';
import { PermissionRedirect } from '../components/PermissionRedirect';
import { CardListSkeleton } from '../components/Skeletons';
import { RoleEditor } from '../components/flow/RoleEditor';
import { RoleSpecCard } from '../components/flow/RoleSpecCard';
import { RubricCard } from '../components/flow/RubricCard';
import { ShortlistSection } from '../components/flow/ShortlistSection';
import { roleStatusLabel, seniorityLabel } from '../lib/format';
import { canManageRoles } from '../lib/permissions';
import { useDeleteRole, useRole } from '../query/flow';
import { useAuthStore } from '../stores/auth';
import { fonts } from '../theme/tokens';

const detailChipSx = {
  borderRadius: '999px',
  fontFamily: fonts.body,
  fontWeight: 750,
  height: 30,
  '& .MuiChip-label': { px: 1.25 },
};

function roleTitle(role: Role) {
  return role.title || role.spec.title || 'Untitled role';
}

function formatCreatedAt(value: string) {
  if (!value) {
    return 'Created date unavailable';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return 'Created date unavailable';
  }
  return `Created ${new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' }).format(date)}`;
}

export function RoleDetailPage() {
  const { roleId } = useParams();
  const navigate = useNavigate();
  const userRole = useAuthStore((s) => s.user?.role);
  const userId = useAuthStore((s) => s.user?.id);
  const canUseRoles = canManageRoles(userRole);
  const roleQuery = useRole(roleId, canUseRoles && Boolean(roleId));
  const deleteRole = useDeleteRole();
  const [editing, setEditing] = useState(false);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [roleVersion, setRoleVersion] = useState(0);

  if (!canUseRoles) {
    return <PermissionRedirect />;
  }

  const role = roleQuery.data?.role;
  // Only the owning employer may edit/delete — mirrors the backend ownership
  // guard (grpc/role.go). A reviewer viewing another employer's role sees it read-only.
  const isOwner = Boolean(role) && role?.employerId === userId;
  const title = role ? roleTitle(role) : 'Role details';

  const onDelete = () => {
    if (!role) {
      return;
    }
    deleteRole.mutate(role.id, {
      onSuccess: () => navigate('/roles', { replace: true }),
    });
  };

  return (
    <Stack spacing={4} sx={{ maxWidth: 1040, mx: 'auto' }}>
      <PageBackButton fallbackTo="/roles" label="Back to roles" alwaysFallback />

      <Stack spacing={2.5}>
        <Stack
          direction={{ xs: 'column', md: 'row' }}
          spacing={2}
          useFlexGap
          sx={{ alignItems: { xs: 'flex-start', md: 'flex-end' }, justifyContent: 'space-between' }}
        >
          <Stack spacing={1.25} sx={{ minWidth: 0 }}>
            <Typography variant="h1" component="h1" sx={{ fontSize: { xs: 38, md: 52 }, lineHeight: 1.02 }}>
              {title}
            </Typography>
            {role && (
              <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap' }}>
                <Chip
                  size="small"
                  label={roleStatusLabel[role.status]}
                  color={role.status === 'ROLE_STATUS_OPEN' ? 'primary' : 'default'}
                  variant={role.status === 'ROLE_STATUS_OPEN' ? 'filled' : 'outlined'}
                  sx={detailChipSx}
                />
                <Chip size="small" label={seniorityLabel[role.spec.seniority]} variant="outlined" sx={detailChipSx} />
                {role.spec.location && <Chip size="small" label={role.spec.location} variant="outlined" sx={detailChipSx} />}
                {role.spec.availability && <Chip size="small" label={role.spec.availability} variant="outlined" sx={detailChipSx} />}
                <Chip
                  size="small"
                  label={`${role.rubric.competencies.length} ${
                    role.rubric.competencies.length === 1 ? 'competency' : 'competencies'
                  }`}
                  variant="outlined"
                  sx={detailChipSx}
                />
              </Stack>
            )}
          </Stack>

          {role && isOwner && !editing && (
            <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap', width: { xs: '100%', sm: 'auto' } }}>
              <Button
                variant="outlined"
                startIcon={<EditRounded fontSize="small" />}
                onClick={() => setEditing(true)}
                sx={{ minHeight: 44, borderRadius: '8px', fontWeight: 800 }}
              >
                Refine role
              </Button>
              <Button
                variant="outlined"
                color="error"
                startIcon={<DeleteOutlineRounded fontSize="small" />}
                onClick={() => {
                  deleteRole.reset();
                  setConfirmingDelete(true);
                }}
                sx={{ minHeight: 44, borderRadius: '8px', fontWeight: 800 }}
              >
                Delete role
              </Button>
            </Stack>
          )}
        </Stack>

        {role && (
          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: { xs: '1fr', sm: 'repeat(3, minmax(0, 1fr))' },
              gap: 1,
              p: 1,
              border: '1px solid',
              borderColor: 'divider',
              borderRadius: '8px',
              bgcolor: 'background.paper',
            }}
          >
            {[
              ['Status', roleStatusLabel[role.status]],
              ['Seniority', seniorityLabel[role.spec.seniority]],
              ['Created', formatCreatedAt(role.createdAt).replace(/^Created /, '')],
            ].map(([label, value]) => (
              <Box key={label} sx={{ p: 1.5, minWidth: 0 }}>
                <Typography variant="caption" color="text.secondary" sx={{ fontFamily: fonts.mono, fontWeight: 800, textTransform: 'uppercase' }}>
                  {label}
                </Typography>
                <Typography sx={{ mt: 0.5, fontWeight: 850, overflowWrap: 'anywhere' }}>{value}</Typography>
              </Box>
            ))}
          </Box>
        )}
      </Stack>

      {!roleId ? (
        <Alert severity="info">Pick a role from your roles list first.</Alert>
      ) : roleQuery.isPending ? (
        <CardListSkeleton count={3} />
      ) : roleQuery.isError ? (
        <Alert severity="error">{roleQuery.error instanceof Error ? roleQuery.error.message : 'Could not load this role.'}</Alert>
      ) : !role ? (
        <Alert severity="info">This role could not be found.</Alert>
      ) : editing ? (
        <RoleEditor
          role={role}
          onSaved={() => {
            setRoleVersion((v) => v + 1);
            setEditing(false);
          }}
          onCancel={() => setEditing(false)}
        />
      ) : (
        <Stack spacing={3}>
          <RoleSpecCard spec={role.spec} />
          <RubricCard rubric={role.rubric} />
          <Divider />
          <ShortlistSection roleId={role.id} version={roleVersion} />
        </Stack>
      )}

      <Dialog
        open={confirmingDelete}
        onClose={() => {
          if (!deleteRole.isPending) {
            setConfirmingDelete(false);
          }
        }}
        fullWidth
        maxWidth="xs"
      >
        <DialogTitle>Delete role?</DialogTitle>
        <DialogContent>
          <Stack spacing={1.5}>
            <Typography>
              Delete {role ? roleTitle(role) : 'this role'} and its role-scoped shortlist, interviews, and applications.
            </Typography>
            {deleteRole.isError && (
              <Alert severity="error">
                {deleteRole.error instanceof Error ? deleteRole.error.message : 'Could not delete this role.'}
              </Alert>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmingDelete(false)} disabled={deleteRole.isPending}>
            Cancel
          </Button>
          <DotsButton color="error" variant="contained" loading={deleteRole.isPending} onClick={onDelete}>
            Delete role
          </DotsButton>
        </DialogActions>
      </Dialog>
    </Stack>
  );
}
