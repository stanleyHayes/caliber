import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { CardListSkeleton, CardSkeleton } from './Skeletons';

describe('Skeletons', () => {
  it('CardSkeleton exposes a labelled loading status region for screen readers', () => {
    render(<CardSkeleton />);
    expect(screen.getByRole('status', { name: 'Loading content' })).toBeInTheDocument();
  });

  it('CardListSkeleton renders one status region per placeholder card', () => {
    render(<CardListSkeleton count={4} />);
    expect(screen.getAllByRole('status')).toHaveLength(4);
  });
});
