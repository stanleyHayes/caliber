import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { PageControls } from './PageControls';

describe('PageControls', () => {
  it('renders nothing for a single page (no noisy controls)', () => {
    const { container } = render(<PageControls page={1} pageCount={1} onChange={() => {}} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('renders pagination and reports the chosen 1-based page', () => {
    const onChange = vi.fn();
    render(<PageControls page={1} pageCount={3} onChange={onChange} />);
    fireEvent.click(screen.getByRole('button', { name: /go to page 2/i }));
    expect(onChange).toHaveBeenCalledWith(2);
  });
});
