import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  TextField,
} from '@mui/material';
import { useState } from 'react';

import type { ContestSubject } from '../../api/types';
import { useRaiseContest } from '../../query/contest';
import { DotsButton } from '../DotsButton';

const SUBJECT_NOUN: Record<ContestSubject, string> = {
  CONTEST_SUBJECT_UNSPECIFIED: 'assessment',
  CONTEST_SUBJECT_MATCH: 'shortlist result',
  CONTEST_SUBJECT_REPORT_CARD: 'report card',
};

// ContestAssessment lets a candidate dispute an assessment of them (a shortlist
// match or an interview report card). The dispute is a human-reviewed,
// non-destructive fairness control — it never alters the underlying assessment;
// it opens a record for a human to uphold or dismiss (CAL-083/094).
export function ContestAssessment({ subject, subjectId }: { subject: ContestSubject; subjectId: string }) {
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState('');
  const raise = useRaiseContest();
  const noun = SUBJECT_NOUN[subject];

  if (raise.isSuccess) {
    return (
      <Alert severity="success" sx={{ alignSelf: 'flex-start' }}>
        Your dispute was submitted for human review.
      </Alert>
    );
  }

  const submit = () => {
    if (reason.trim().length === 0) {
      return;
    }
    raise.mutate({ subject, subjectId, reason: reason.trim() });
  };

  return (
    <>
      <Button variant="text" color="inherit" onClick={() => setOpen(true)} sx={{ alignSelf: 'flex-start' }}>
        Dispute this {noun}
      </Button>
      <Dialog open={open} onClose={() => setOpen(false)} fullWidth maxWidth="sm">
        <DialogTitle>Dispute this {noun}</DialogTitle>
        <DialogContent>
          <DialogContentText sx={{ mb: 2 }}>
            Tell us what you think the assessment got wrong. A human reviewer will look at it — your
            {' '}
            {noun} is not changed automatically.
          </DialogContentText>
          {raise.isError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {raise.error instanceof Error ? raise.error.message : 'Could not submit your dispute.'}
            </Alert>
          )}
          <TextField
            autoFocus
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="e.g. The breakdown missed the Go services I shipped at my last role."
            label="Reason"
            multiline
            minRows={3}
            fullWidth
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)} color="inherit">
            Cancel
          </Button>
          <DotsButton variant="contained" loading={raise.isPending} disabled={reason.trim().length === 0} onClick={submit}>
            Submit dispute
          </DotsButton>
        </DialogActions>
      </Dialog>
    </>
  );
}
