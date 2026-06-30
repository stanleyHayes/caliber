import {
  Box,
  Card,
  CardContent,
  Divider,
  FormControlLabel,
  MenuItem,
  Slider,
  Stack,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import { useState } from 'react';

import type { Competency, Role, RoleSpec, Seniority } from '../../api/types';
import { useUpdateRole } from '../../query/flow';
import { pct } from '../../lib/format';
import { DotsButton } from '../DotsButton';

const SENIORITIES: { value: Seniority; label: string }[] = [
  { value: 'SENIORITY_JUNIOR', label: 'Junior' },
  { value: 'SENIORITY_MID', label: 'Mid' },
  { value: 'SENIORITY_SENIOR', label: 'Senior' },
  { value: 'SENIORITY_LEAD', label: 'Lead' },
];

export function RoleEditor({
  role,
  onSaved,
  onCancel,
}: {
  role: Role;
  onSaved: (role: Role) => void;
  onCancel: () => void;
}) {
  const update = useUpdateRole();
  // Hold the full spec so untouched fields (responsibilities, must-haves…) survive the save.
  const [spec, setSpec] = useState<RoleSpec>(role.spec);
  const [comps, setComps] = useState<Competency[]>(role.rubric.competencies);

  const patchSpec = (p: Partial<RoleSpec>) => setSpec((s) => ({ ...s, ...p }));
  const patchComp = (i: number, p: Partial<Competency>) =>
    setComps((cs) => cs.map((c, j) => (j === i ? { ...c, ...p } : c)));

  const total = comps.reduce((sum, c) => sum + c.weight, 0);

  const save = () => {
    update.mutate(
      { roleId: role.id, spec, rubric: { competencies: comps } },
      { onSuccess: (data) => onSaved(data.role) },
    );
  };

  return (
    <Card variant="outlined">
      <CardContent>
        <Stack spacing={3}>
          <Typography variant="h6">Refine spec &amp; rubric</Typography>

          <Stack spacing={2}>
            <TextField
              label="Title"
              value={spec.title}
              onChange={(e) => patchSpec({ title: e.target.value })}
              fullWidth
            />
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
              <TextField
                label="Location"
                value={spec.location}
                onChange={(e) => patchSpec({ location: e.target.value })}
                fullWidth
              />
              <TextField
                select
                label="Seniority"
                value={spec.seniority}
                onChange={(e) => patchSpec({ seniority: e.target.value as Seniority })}
                sx={{ width: { xs: '100%', sm: 160 } }}
              >
                {SENIORITIES.map((o) => (
                  <MenuItem key={o.value} value={o.value}>
                    {o.label}
                  </MenuItem>
                ))}
              </TextField>
            </Stack>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
              <TextField
                label="Currency"
                value={spec.salaryBand.currency}
                onChange={(e) => patchSpec({ salaryBand: { ...spec.salaryBand, currency: e.target.value } })}
                sx={{ width: { xs: '100%', sm: 120 } }}
              />
              <TextField
                label="Salary low"
                type="number"
                value={spec.salaryBand.low}
                onChange={(e) => patchSpec({ salaryBand: { ...spec.salaryBand, low: Number(e.target.value) } })}
                fullWidth
              />
              <TextField
                label="Salary high"
                type="number"
                value={spec.salaryBand.high}
                onChange={(e) => patchSpec({ salaryBand: { ...spec.salaryBand, high: Number(e.target.value) } })}
                fullWidth
              />
            </Stack>
          </Stack>

          <Divider />

          <Stack spacing={2}>
            <Stack direction="row" sx={{ justifyContent: 'space-between', alignItems: 'baseline' }}>
              <Typography variant="subtitle2">Rubric weights</Typography>
              <Typography variant="caption" color="text.secondary">
                weights are re-normalized on save (now {pct(total)})
              </Typography>
            </Stack>
            {comps.map((c, i) => (
              <Box key={i}>
                <Stack
                direction={{ xs: 'column', sm: 'row' }}
                spacing={2}
                sx={{ alignItems: { xs: 'flex-start', sm: 'center' } }}
              >
                <Typography variant="body2" sx={{ flexGrow: 1, minWidth: 0 }}>
                  {c.name}
                </Typography>
                <Slider
                  value={c.weight}
                  onChange={(_, v) => patchComp(i, { weight: v as number })}
                  min={0}
                  max={1}
                  step={0.05}
                  valueLabelDisplay="auto"
                  valueLabelFormat={(v) => pct(v)}
                  sx={{ width: { xs: '100%', sm: 200 }, minWidth: { sm: 120 } }}
                />
                <FormControlLabel
                  control={
                    <Switch checked={c.mustHave} onChange={(e) => patchComp(i, { mustHave: e.target.checked })} />
                  }
                  label="must-have"
                />
              </Stack>
              </Box>
            ))}
          </Stack>

          <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap' }}>
            <DotsButton variant="contained" loading={update.isPending} onClick={save} disabled={total === 0}>
              Save changes
            </DotsButton>
            <DotsButton variant="text" onClick={onCancel} disabled={update.isPending}>
              Cancel
            </DotsButton>
          </Stack>
        </Stack>
      </CardContent>
    </Card>
  );
}
