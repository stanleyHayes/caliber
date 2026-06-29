import { fireEvent, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { ContestAssessment } from './ContestAssessment';

const mutate = vi.fn();
const raiseState = { mutate, isPending: false, isError: false, isSuccess: false, error: null as Error | null };
vi.mock('../../query/contest', () => ({ useRaiseContest: () => raiseState }));

beforeEach(() => {
  mutate.mockReset();
  raiseState.isPending = false;
  raiseState.isError = false;
  raiseState.isSuccess = false;
  raiseState.error = null;
});
afterEach(() => vi.clearAllMocks());

describe('ContestAssessment', () => {
  it('opens a dialog and submits a dispute with the subject, id, and reason', () => {
    render(<ContestAssessment subject="CONTEST_SUBJECT_REPORT_CARD" subjectId="iv-1" />);

    fireEvent.click(screen.getByRole('button', { name: 'Dispute this report card' }));
    const submit = screen.getByRole('button', { name: 'Submit dispute' });
    expect(submit).toBeDisabled(); // gated until a reason is given

    fireEvent.change(screen.getByLabelText(/reason/i), { target: { value: 'The breakdown missed my Go work.' } });
    expect(submit).toBeEnabled();
    fireEvent.click(submit);

    expect(mutate).toHaveBeenCalledWith({
      subject: 'CONTEST_SUBJECT_REPORT_CARD',
      subjectId: 'iv-1',
      reason: 'The breakdown missed my Go work.',
    });
  });

  it('confirms once the dispute is submitted for human review', () => {
    raiseState.isSuccess = true;
    render(<ContestAssessment subject="CONTEST_SUBJECT_REPORT_CARD" subjectId="iv-1" />);
    expect(screen.getByText(/submitted for human review/i)).toBeInTheDocument();
    // The trigger button is gone — no double-submission.
    expect(screen.queryByRole('button', { name: /Dispute this/ })).not.toBeInTheDocument();
  });

  it('surfaces a submission error inside the dialog', () => {
    raiseState.isError = true;
    raiseState.error = new Error('Service unavailable');
    render(<ContestAssessment subject="CONTEST_SUBJECT_MATCH" subjectId="m-1" />);
    fireEvent.click(screen.getByRole('button', { name: 'Dispute this shortlist result' }));
    expect(screen.getByText('Service unavailable')).toBeInTheDocument();
  });
});
