import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { TimeToShortlistMetric } from '../../api/types';
import { TimeToShortlistHeadline } from './TimeToShortlistHeadline';

describe('TimeToShortlistHeadline', () => {
  it('renders the weeks-to-hours collapse derived from the metric', () => {
    const metric: TimeToShortlistMetric = { baselineHours: 504, currentHours: 2, improvementFactor: 252 };
    render(<TimeToShortlistHeadline metric={metric} />);
    expect(screen.getByText('252×')).toBeInTheDocument();
    // 504h baseline -> ~21 days; the headline contrasts that with the live hours.
    expect(screen.getByText(/From ~21 days to 2 hours/)).toBeInTheDocument();
  });
});
