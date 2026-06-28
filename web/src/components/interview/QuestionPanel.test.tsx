import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { InterviewQuestion } from '../../api/types';
import { QuestionPanel } from './QuestionPanel';

const question: InterviewQuestion = {
  interviewId: 'iv1',
  ordinal: 1,
  text: 'Walk me through a Go service you built.',
  competencyTag: 'Go',
};

describe('QuestionPanel', () => {
  it('shows the question + competency and disables submit until there is an answer', () => {
    render(<QuestionPanel question={question} onAnswer={() => {}} />);
    expect(screen.getByText('Walk me through a Go service you built.')).toBeInTheDocument();
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /submit answer/i })).toBeDisabled();
  });

  it('submits the trimmed answer and clears the field', () => {
    const onAnswer = vi.fn();
    render(<QuestionPanel question={question} onAnswer={onAnswer} />);
    const box = screen.getByRole('textbox');
    fireEvent.change(box, { target: { value: '  I built a payments service in Go.  ' } });

    const btn = screen.getByRole('button', { name: /submit answer/i });
    expect(btn).toBeEnabled();
    fireEvent.click(btn);

    expect(onAnswer).toHaveBeenCalledWith('I built a payments service in Go.');
    expect((box as HTMLTextAreaElement).value).toBe('');
  });

  it('ignores a whitespace-only answer', () => {
    const onAnswer = vi.fn();
    render(<QuestionPanel question={question} onAnswer={onAnswer} />);
    fireEvent.change(screen.getByRole('textbox'), { target: { value: '   ' } });
    expect(screen.getByRole('button', { name: /submit answer/i })).toBeDisabled();
    expect(onAnswer).not.toHaveBeenCalled();
  });
});
