import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { DotsButton } from './DotsButton';

describe('DotsButton', () => {
  it('shows its label and is not busy when idle', () => {
    render(<DotsButton>Generate</DotsButton>);
    const btn = screen.getByRole('button', { name: 'Generate' });
    expect(btn).toBeEnabled();
    expect(btn).toHaveAttribute('aria-busy', 'false');
  });

  it('shows animated dots (not a spinner) and is busy + disabled while loading', () => {
    render(<DotsButton loading>Generate</DotsButton>);
    const btn = screen.getByRole('button');
    expect(btn).toBeDisabled();
    expect(btn).toHaveAttribute('aria-busy', 'true');
    // The width-stable dots carry an accessible "loading" label; the label is hidden.
    expect(screen.getByLabelText('loading')).toBeInTheDocument();
    expect(screen.queryByText('Generate')).not.toBeInTheDocument();
  });
});
