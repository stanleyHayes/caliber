import { ThemeProvider } from '@mui/material';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { theme } from '../theme/theme';
import { ModeToggle } from './ModeToggle';

// jsdom implements neither matchMedia nor startViewTransition; with both absent
// (or reduced-motion true) ModeToggle takes its plain setMode fallback path,
// which is exactly the behaviour we assert here.
function mockMatchMedia(reduced: boolean) {
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockReturnValue({
      matches: reduced,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(), // legacy API still used by MUI's color-scheme hook
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }),
  );
}

function renderToggle() {
  return render(
    <ThemeProvider theme={theme} defaultMode="light">
      <ModeToggle />
    </ThemeProvider>,
  );
}

beforeEach(() => {
  localStorage.clear();
  mockMatchMedia(false);
});
afterEach(() => {
  vi.unstubAllGlobals();
  localStorage.clear();
});

describe('ModeToggle', () => {
  it('shows the action to switch to the opposite of the current (light) mode', () => {
    renderToggle();
    expect(screen.getByLabelText('switch to dark mode')).toBeInTheDocument();
  });

  it('toggles the color scheme on click', () => {
    renderToggle();
    fireEvent.click(screen.getByLabelText('switch to dark mode'));
    // After switching to dark, the control now offers a switch back to light.
    expect(screen.getByLabelText('switch to light mode')).toBeInTheDocument();
  });

  it('still toggles when prefers-reduced-motion is set (no circular-reveal animation)', () => {
    mockMatchMedia(true);
    renderToggle();
    fireEvent.click(screen.getByLabelText('switch to dark mode'));
    expect(screen.getByLabelText('switch to light mode')).toBeInTheDocument();
  });

  it('uses view transition when the browser supports it', async () => {
    const ready = Promise.resolve();
    const startViewTransition = vi.fn((cb: () => void) => {
      cb();
      return { ready };
    });
    Object.assign(document, { startViewTransition });

    renderToggle();
    fireEvent.click(screen.getByLabelText('switch to dark mode'));

    expect(startViewTransition).toHaveBeenCalled();
    await waitFor(() => expect(screen.getByLabelText('switch to light mode')).toBeInTheDocument());
  });
});
