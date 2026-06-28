import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { Rubric } from '../../api/types';
import { RubricCard } from './RubricCard';

const rubric: Rubric = {
  competencies: [
    { name: 'Go', weight: 0.6, mustHave: true },
    { name: 'SQL', weight: 0.4, mustHave: false },
  ],
};

describe('RubricCard', () => {
  it('renders each competency with its name and weight as a percentage', () => {
    render(<RubricCard rubric={rubric} />);
    expect(screen.getByText('Scoring rubric')).toBeInTheDocument();
    expect(screen.getByText('Go')).toBeInTheDocument();
    expect(screen.getByText('SQL')).toBeInTheDocument();
    expect(screen.getByText('60%')).toBeInTheDocument();
    expect(screen.getByText('40%')).toBeInTheDocument();
  });

  it('flags only must-have competencies, so the gating signal is explicit', () => {
    render(<RubricCard rubric={rubric} />);
    // Exactly one competency is a must-have, so exactly one chip is shown.
    expect(screen.getAllByText('must-have')).toHaveLength(1);
  });

  it('renders cleanly with an empty rubric (no competencies)', () => {
    render(<RubricCard rubric={{ competencies: [] }} />);
    expect(screen.getByText('Scoring rubric')).toBeInTheDocument();
    expect(screen.queryByText('must-have')).not.toBeInTheDocument();
  });
});
