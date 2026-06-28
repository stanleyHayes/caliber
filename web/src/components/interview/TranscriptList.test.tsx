import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { InterviewTurn } from '../../hooks/useInterview';
import { TranscriptList } from './TranscriptList';

const turns: InterviewTurn[] = [
  { ordinal: 1, question: 'Tell me about a system you built.', answer: 'I built a payments ledger.', competencyTag: 'System design' },
  { ordinal: 2, question: 'How do you handle failures?', answer: 'Retries with backoff.', competencyTag: '' },
];

describe('TranscriptList', () => {
  it('renders nothing when there are no turns', () => {
    const { container } = render(<TranscriptList turns={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders each turn with its question and answer', () => {
    render(<TranscriptList turns={turns} />);
    expect(screen.getByText('Tell me about a system you built.')).toBeInTheDocument();
    expect(screen.getByText('I built a payments ledger.')).toBeInTheDocument();
    expect(screen.getByText('How do you handle failures?')).toBeInTheDocument();
    expect(screen.getByText('Retries with backoff.')).toBeInTheDocument();
  });

  it('labels a turn with its competency tag, falling back to the question ordinal', () => {
    render(<TranscriptList turns={turns} />);
    expect(screen.getByText('System design')).toBeInTheDocument();
    // The second turn has no competency tag, so it falls back to "Q2".
    expect(screen.getByText('Q2')).toBeInTheDocument();
  });
});
