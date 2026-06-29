import {
  Alert,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  FormControlLabel,
  TextField,
} from '@mui/material';
import { useState } from 'react';

import { useRecordRejection } from '../../query/flow';
import { DotsButton } from '../DotsButton';

// DeclineCandidate surfaces the human-approval gate (CAL-081/094): the AI never
// auto-rejects. A decline is a deliberate human decision — it requires a written
// reason AND an explicit human-approval confirmation before it can be recorded,
// and it is logged to the audit trail as the standing approval. The approving
// human's identity comes from the auth context, never this form.
export function DeclineCandidate({ roleId, candidateId }: { roleId: string; candidateId: string }) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState('');
  const [approved, setApproved] = useState(false);
  const reject = useRecordRejection();

  if (reject.isSuccess) {
    return (
      <Alert severity="success" sx={{ alignSelf: 'flex-start' }}>
        Decline recorded (human-approved &amp; logged).
      </Alert>
    );
  }

  const canSubmit = reason.trim().length > 0 && approved;
  const submit = () => {
    if (!canSubmit) {
      return;
    }
    reject.mutate({ roleId, candidateId, reason: reason.trim(), humanApproved: true });
  };

  return (
    <>
      <Button variant="text" color="error" onClick={() => setOpen(true)} sx={{ alignSelf: 'flex-start' }}>
        Decline candidate
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)} fullWidth maxWidth="sm">
        <DialogContent>
          <DialogContentText sx={{ mb: 2 }}>
            Caliber never declines a candidate on its own. This is your decision as the hiring human —
            give a reason and confirm it below. The decline is logged for the audit trail.
          </DialogContentText>
          {reject.isError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {reject.error instanceof Error ? reject.error.message : 'Could not record the decline.'}
            </Alert>
          )}
          <TextField
            autoFocus
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            label="Reason for declining"
            placeholder="e.g. Strong fit, but the role needs deeper distributed-systems depth."
            multiline
            minRows={3}
            fullWidth
          />
          <FormControlLabel
            sx={{ mt: 1 }}
            control={<Checkbox checked={approved} onChange={(e) => setApproved(e.target.checked)} />}
            label="I confirm this is my decision as the hiring human."
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)} color="inherit">
            Cancel
          </Button>
          <DotsButton variant="contained" color="error" loading={reject.isPending} disabled={!canSubmit} onClick={submit}>
            Record decline
          </DotsButton>
        </DialogActions>
      </Dialog>
    </>
  );
}
