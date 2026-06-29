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
import { useNavigate } from 'react-router-dom';

import { DotsButton } from '../DotsButton';
import { useDeleteMyData } from '../../query/privacy';
import { useAuthStore } from '../../stores/auth';

const CONFIRM_WORD = 'DELETE';

// DeleteAccount surfaces the right-to-erasure (CAL-118): a candidate can
// permanently delete their account and all their data. It is irreversible, so it
// requires an explicit typed confirmation; on success the session is cleared and
// the user is returned to the landing page.
export function DeleteAccount() {
  const [open, setOpen] = useState(false);
  const [typed, setTyped] = useState('');
  const del = useDeleteMyData();
  const navigate = useNavigate();
  const clearSession = useAuthStore((s) => s.clear);

  const close = () => {
    setOpen(false);
    setTyped('');
  };

  const confirm = () => {
    if (typed.trim().toUpperCase() !== CONFIRM_WORD) {
      return;
    }
    del.mutate(undefined, {
      onSuccess: () => {
        clearSession();
        navigate('/', { replace: true });
      },
    });
  };

  return (
    <>
      <Button variant="text" color="error" onClick={() => setOpen(true)} sx={{ alignSelf: 'flex-start' }}>
        Delete my account
      </Button>
      <Dialog open={open} onClose={close} fullWidth maxWidth="sm">
        <DialogTitle>Delete your account and data?</DialogTitle>
        <DialogContent>
          <DialogContentText sx={{ mb: 2 }}>
            This permanently erases your profile, applications, screenings, and disputes. It cannot be
            undone. Type <strong>{CONFIRM_WORD}</strong> to confirm.
          </DialogContentText>
          {del.isError && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {del.error instanceof Error ? del.error.message : 'Could not delete your account.'}
            </Alert>
          )}
          <TextField
            autoFocus
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            label={`Type ${CONFIRM_WORD}`}
            fullWidth
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={close} color="inherit">
            Cancel
          </Button>
          <DotsButton
            variant="contained"
            color="error"
            loading={del.isPending}
            disabled={typed.trim().toUpperCase() !== CONFIRM_WORD}
            onClick={confirm}
          >
            Delete everything
          </DotsButton>
        </DialogActions>
      </Dialog>
    </>
  );
}
