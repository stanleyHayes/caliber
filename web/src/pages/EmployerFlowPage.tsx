import { Alert, Box, Button, Card, CardContent, Divider, Stack, TextField, Typography } from '@mui/material';
import { useState } from 'react';

import type { GenerateRoleResponse } from '../api/types';
import { DotsButton } from '../components/DotsButton';
import { RoleEditor } from '../components/flow/RoleEditor';
import { RoleSpecCard } from '../components/flow/RoleSpecCard';
import { RubricCard } from '../components/flow/RubricCard';
import { ShortlistSection } from '../components/flow/ShortlistSection';
import { useGenerateRole } from '../query/flow';
import { useAuthStore } from '../stores/auth';

const PLACEHOLDER =
  'e.g. We need a senior Go backend engineer in Accra to own our matching services — must know Postgres and gRPC, ideally some Kubernetes. GHS 18k–25k, start within a month.';

export function EmployerFlowPage() {
  const user = useAuthStore((s) => s.user);
  const generate = useGenerateRole();
  const [text, setText] = useState('');
  const [result, setResult] = useState<GenerateRoleResponse | null>(null);
  const [editing, setEditing] = useState(false);

  const onGenerate = () => {
    if (!user || text.trim().length === 0) {
      return;
    }
    generate.mutate(
      { employerId: user.id, freeText: text },
      { onSuccess: (data) => setResult(data) },
    );
  };

  return (
    <Stack spacing={4} sx={{ maxWidth: 820, mx: 'auto' }}>
      <Stack spacing={1}>
        <Typography variant="h3">Describe the role</Typography>
        <Typography color="text.secondary">
          Write the hiring need in plain language. Caliber turns it into a structured spec and a
          weighted, bias-safe rubric — then ranks your pool with reasons.
        </Typography>
      </Stack>

      <Card variant="outlined">
        <CardContent>
          <Stack spacing={2}>
            <TextField
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder={PLACEHOLDER}
              multiline
              minRows={4}
              fullWidth
            />
            {generate.isError && (
              <Alert severity="error">
                {generate.error instanceof Error ? generate.error.message : 'Generation failed.'}
              </Alert>
            )}
            <Box>
              <DotsButton
                variant="contained"
                size="large"
                loading={generate.isPending}
                disabled={text.trim().length === 0}
                onClick={onGenerate}
              >
                Generate spec &amp; rubric
              </DotsButton>
            </Box>
          </Stack>
        </CardContent>
      </Card>

      {result && (
        <Stack spacing={3}>
          <Alert severity="success">
            {result.availableMatches > 0
              ? `${result.availableMatches} strong match${result.availableMatches > 1 ? 'es' : ''} already in your pool.`
              : 'Spec and rubric ready.'}
          </Alert>
          {editing ? (
            <RoleEditor
              role={result.role}
              onSaved={(role) => {
                setResult({ ...result, role });
                setEditing(false);
              }}
              onCancel={() => setEditing(false)}
            />
          ) : (
            <>
              <Stack direction="row" sx={{ justifyContent: 'flex-end' }}>
                <Button variant="outlined" onClick={() => setEditing(true)}>
                  Refine spec &amp; rubric
                </Button>
              </Stack>
              <RoleSpecCard spec={result.role.spec} />
              <RubricCard rubric={result.role.rubric} />
            </>
          )}
          <Divider />
          <ShortlistSection roleId={result.role.id} />
        </Stack>
      )}
    </Stack>
  );
}
